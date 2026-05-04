# Carryless — Architecture Review & Cross-Platform Strategy

> **Status:** Draft v0. This document was authored during a deep architectural review session. Tasks marked **OPEN QUESTION** require your input before implementation can proceed safely.

---

## 0. Document context

This file replaces a (referenced but never committed) `PLAN.md`. The original file does not exist in git history; commit `d018e74` only removed it from `.gitignore`. The brief asked for a review of:

- A 4-tier sharing system (view / add / edit / admin)
- A `pack_sub_item_checks` table for per-pack sub-item state
- A Phase 0 test plan
- A mobile-conversion strategy

Only the mobile/cross-platform strategy can be planned without seeing the original document. Sections 2 and 3 below cover what I learned from the code itself (so the review is grounded in reality, not vapor). Sections 4–7 are the actionable plan and the questions I need answered.

---

## 1. Executive summary

1. **The codebase is owner-only.** Every authorization check in `packs.go`, `items.go`, etc. is `WHERE user_id = ?` or `pack.UserID != userID`. There is no ACL table, no sharing concept, no role abstraction. Adding sharing is a deep, invasive change — every read and mutation path needs an "effective permission" check, not just one new table.
2. **The migration strategy is fragile.** `database.Migrate()` runs ~17 sequential, idempotent functions on every startup, several of which recreate tables (SQLite ALTER limitations). It works today, but it cannot tolerate concurrent writers and has no version table. Before adding a sharing schema, this should be replaced or formalised.
3. **HTML and JSON are already mixed in the same handlers.** Page handlers return HTML; mutation handlers return JSON; some return redirects with `?error=foo` query strings. There is no clean API surface to expose to a mobile app. **The right move before mobile is to extract a versioned `/api/v1/*` namespace that returns only JSON, and let the existing HTML routes thin-wrap it.**
4. **Auth is cookie+CSRF only.** Mobile clients cannot reasonably participate in this scheme. A token-exchange endpoint is the lowest-risk way to add bearer-token auth without breaking web sessions.
5. **SQLite is single-writer.** The current `sql.Open(...)` does not enable WAL or set a busy timeout. Under multi-user (let alone multi-device) load this will produce `SQLITE_BUSY` errors. Trivial to fix; must fix before sharing or mobile.

---

## 2. Architectural observations from the existing code

### 2.1 Authentication & sessions

| File | Finding |
|---|---|
| `middleware/middleware.go:394` (`AuthRequired`) | Reads `session_id` cookie, validates against `sessions` table, sets `user`/`user_id`/`db` on context. No bearer-token path. |
| `database/auth.go:152` (`ValidateSession`) | Sliding-window: every validated request renews the session by calling `RenewSession`. Good for UX, bad for stateless tokens. |
| `database/auth.go:288` (`CreateCSRFToken`) | CSRF tokens are persisted in SQLite, expire in 1h, and are **single-use** (deleted on validate at `auth.go:332`). Every page render mints a fresh token. |
| `middleware/middleware.go:280` (`CSRF`) | Skipped entirely in dev mode (`cfg.IsDevelopment()`). Also accepts the token via `X-CSRF-Token` header **or** `csrf_token` form field. |
| `database/auth.go:124` (`CreateSession`) | Session IDs are hex-encoded 32-byte tokens. Fine for cookies, fine as bearer tokens — no JWT needed. |

**Implication for mobile:** the existing `sessions` table is already a perfectly serviceable opaque-bearer-token store. We do not need JWT; we need an exchange endpoint that returns the session ID for native clients to send as `Authorization: Bearer <token>`.

### 2.2 Authorization model

There is **no abstraction**. Every database-layer function takes `userID` and joins/filters by it directly:

- `packs.go:323` `UpdatePack`: `WHERE id = ? AND user_id = ?`
- `packs.go:393` `DeletePack`: same pattern
- `packs.go:416` `AddItemToPack`: explicit `if pack.UserID != userID` check
- `items.go:145` `GetItem`: `WHERE i.id = ? AND i.user_id = ?`
- `items.go:213` `UpdateItem`: same

This is fine for an owner-only world. For sharing, every one of these signatures has to change to check effective permission, not ownership. The existing tests (`database_test.go`, 290 lines) only cover owner cases.

### 2.3 Handler shape

`handlers/handlers.go:16` `SetupRoutes` mixes HTML and JSON responses by route, not by namespace:

- `GET /packs/:id` → HTML (`packs.go:123`)
- `POST /packs/:id/items` → JSON (`packs.go:566`)
- `POST /inventory/items/:id/delete` → 302 redirect with `?error=foo` (`inventory.go:558`)
- `PATCH /api/items/:id` → JSON (`inventory.go:1242`)
- `POST /packs/:id/note` → JSON, also returns a renewed CSRF token (`packs.go:533`)

Some endpoints are already de-facto JSON APIs (`/api/items/:id`, `/api/items/:id/links`, all label endpoints). They're just not under a versioned namespace, and they accept form bodies + URL param IDs, so they aren't documented or stable.

### 2.4 Database / migrations

`database/database.go:25` `Migrate` is a hand-rolled migration runner:

- Runs a list of `CREATE TABLE IF NOT EXISTS` statements every startup
- Then calls 17 individual `addXColumn` / `createXTable` functions
- Several use the SQLite "create temp / copy / drop / rename" dance because SQLite can't `ALTER COLUMN` or `DROP COLUMN` (e.g. `removePurchaseDateColumn` at line 234, `updatePackItemsSchema` at line 290)
- No version table, no rollback, no transactional safety across the full migration

**Risk for sharing:** if you add `pack_shares` and a migration to backfill it, a partial failure mid-migration leaves the DB inconsistent with no easy recovery.

### 2.5 SQLite configuration

`database/database.go:13`:
```go
sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
```

Missing:
- `_journal_mode=WAL` — without this, readers block writers and writers block readers
- `_busy_timeout=5000` — without this, concurrent writes fail immediately with `SQLITE_BUSY`
- `_synchronous=NORMAL` — safe with WAL, ~5× faster
- No `db.SetMaxOpenConns(1)` for the writer — the Go `database/sql` pool will open multiple writer connections, which SQLite cannot parallelize

This is fine for a single user editing through one browser tab, but immediately breaks the moment two devices write at once.

### 2.6 Frontend

- Templates are server-rendered Go `html/template` (29 files in `templates/`)
- One JS file: `static/js/app.js` (622 lines), vanilla
- One CSS file: `static/css/style.css` (3066 lines)
- No build step, no framework, no SPA

The site is essentially a multi-page app. This is good for mobile-web (small, fast) and bad for offline / native shell — the templates would need to be replaced by JSON consumers if a mobile app or richer web client is in scope.

---

## 3. Why the (planned) sharing feature is harder than it looks

Even without seeing the proposed schema, every option for sharing collides with what's in `database/`:

- `AddItemToPack` (`packs.go:416`) takes an `itemID` and asserts both pack ownership *and* item ownership. If a "share with add permission" user adds an item, **whose inventory is the item created in?** The current schema has no answer — `pack_items.item_id` references `items.id`, which is `user_id`-scoped.
- `GetPacks` (`packs.go:116`) is `WHERE user_id = ?`. Shared packs have to appear here, which means a `UNION` against a `pack_shares` table.
- `DeletePack` cascades to `pack_items`, `pack_labels`, `item_labels`, `pack_label_assignments` via `ON DELETE CASCADE`. Sharing has to decide: does deleting a pack revoke shares (yes) and notify shared users (TBD)?
- `DuplicatePack` (`packs.go:667`) currently runs as the owner. If a "view" user duplicates, the duplicate must land in *their* inventory, requiring deep-copying items they don't own.

These are all **product questions**, not just schema questions. They are listed as OPEN QUESTIONs in §6.

---

## 4. Cross-platform strategy: desktop web + mobile web + native app

The goal you stated is one backend serving:

1. The current desktop web app (must keep working)
2. A mobile web app (responsive, possibly PWA)
3. Native iOS and Android apps

The strategy below assumes that's a hard requirement, not a "maybe."

### 4.1 Recommended architecture

**One Go backend, two response surfaces:**

```
┌─────────────────────────────────────────────────┐
│ Go / Gin                                        │
├─────────────────────────────────────────────────┤
│ /api/v1/*   ← JSON-only, bearer or cookie auth  │
│              consumed by: native apps, mobile   │
│              web (via fetch), future SPA        │
├─────────────────────────────────────────────────┤
│ /...        ← HTML server-rendered (current)    │
│              consumed by: desktop web today,    │
│              eventually thinned to a shell that │
│              calls /api/v1                      │
└─────────────────────────────────────────────────┘
```

**Why not "one handler returns HTML or JSON based on Accept":** it sounds clean but it merges two different validation flows (form bodies + multipart vs. JSON), two different error models (HTML re-render with errors vs. JSON `{"errors": {...}}`), and two different success flows (302 redirect vs. 200 with a body). The existing code already shows the pain — `handleCreateItem` does HTML re-renders on validation errors while `handlePatchItem` returns JSON errors. Forcing both through one handler creates branchy code that's hard to test and easy to break.

**Why not "full SPA migration":** highest risk, biggest blast radius, biggest UI rewrite, no incremental delivery. Off the table for v1.

**Why not "separate API service":** SQLite is single-machine. Deploying a second process that points at the same file makes the WAL / locking situation worse, not better.

### 4.2 Phased implementation

#### Phase A — Foundation (no user-visible changes)

A1. **Fix SQLite config.** Add `_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL` to `database.Initialize`. Set `db.SetMaxOpenConns(...)` — see OPEN QUESTION 5.1.
A2. **Replace migration runner.** Adopt a version-tracked migration table (`schema_migrations(version, applied_at)`) and run each migration in a transaction. Backfill an entry per existing implicit migration.
A3. **Add a structured error type.** Replace the `if strings.Contains(err.Error(), "not found")` pattern (used 30+ times in handlers) with a typed error so JSON handlers can map them cleanly. This is mechanical but enables every later step.
A4. **Add `internal/api` package.** Empty for now. Establishes the file structure before there's pressure to ship features into it.

#### Phase B — API extraction

B1. **Create `/api/v1/auth/login`** — accepts JSON `{email, password}`, returns `{token, user, expires_at}`. The token is just a session ID from the existing `sessions` table; no JWT.
B2. **Create `BearerAuth` middleware** — alongside `AuthRequired`, accepts `Authorization: Bearer <token>` and looks it up in `sessions`. Same DB call as cookie auth, just a different transport.
B3. **Disable CSRF for bearer-token requests.** CSRF only matters for cookie-bearing requests; bearer tokens are immune. The middleware should detect `Authorization: Bearer` and short-circuit CSRF.
B4. **Mirror existing JSON endpoints under `/api/v1/`.** `GET /api/v1/items`, `GET /api/v1/packs`, `GET /api/v1/packs/:id`, etc. These wrap the existing `database.*` functions and return clean JSON. The HTML routes keep working unchanged.
B5. **Document the API.** Hand-written OpenAPI spec in `docs/api/openapi.yaml`, served at `/api/v1/openapi.json`. Generates clients for Swift/Kotlin later.

#### Phase C — Sharing (only after A and B)

C1. Schema (assuming the answer to OPEN QUESTION 6.1 is "tier-based"):
```sql
CREATE TABLE pack_shares (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pack_id TEXT NOT NULL,
    shared_with_user_id INTEGER NOT NULL,
    permission TEXT NOT NULL CHECK(permission IN ('view','add','edit','admin')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_by_user_id INTEGER NOT NULL,
    FOREIGN KEY (pack_id) REFERENCES packs(id) ON DELETE CASCADE,
    FOREIGN KEY (shared_with_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(pack_id, shared_with_user_id)
);
```
C2. Introduce `func EffectivePermission(db, userID, packID) (Permission, error)` that returns `owner > admin > edit > add > view > none`. Every existing `pack.UserID != userID` check gets replaced by `perm < required`.
C3. Decide and document the inventory question (OPEN QUESTION 6.4) before writing code: "when shared user with `add` adds an item, whose item is it?"
C4. UI: pack-detail page gains a "Share" panel (web). API: `POST /api/v1/packs/:id/shares`, `DELETE /api/v1/packs/:id/shares/:user_id`.

#### Phase D — Mobile clients

D1. **Decide the shell technology** — see OPEN QUESTION 7.1. Options: native Swift/Kotlin, React Native, Flutter, Capacitor (PWA wrapper). Each has very different implications for what % of code is shared.
D2. Implement the chosen shell against `/api/v1/`.
D3. Real-time updates for shared packs: **start with poll-on-focus + manual pull-to-refresh.** WebSockets and push notifications are a separate, much-larger project. See OPEN QUESTION 8.

#### Phase E — Mobile-web polish

E1. Audit existing templates for mobile breakpoints (most likely already partially responsive — confirm in browser).
E2. Add a Web App Manifest + service worker → installable PWA. Requires no native shell.
E3. Decide whether mobile-web and native ship simultaneously or staggered.

### 4.3 File structure

```
internal/
  api/                  ← new in Phase A
    v1/
      auth.go           ← /api/v1/auth/*
      items.go
      packs.go
      shares.go         ← Phase C
    middleware.go       ← BearerAuth, JSON error envelope
    errors.go           ← typed errors → HTTP status mapping
  handlers/             ← unchanged: HTML pages
  database/             ← unchanged signatures during Phase A/B; gain perm-checked variants in C
  models/               ← unchanged
  middleware/           ← gains BearerAuth alongside cookie auth
```

The existing `handlers/` package keeps rendering HTML by calling `database.*` directly. The new `api/v1/` package calls the same `database.*` functions and returns JSON. **Both call sites use the same business logic.** This is what prevents drift.

---

## 5. SQLite under multi-client load

Concrete configuration to set in `database.Initialize`:

```go
db, err := sql.Open("sqlite3",
    dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_foreign_keys=on")
db.SetMaxOpenConns(1) // for writers; consider a separate read pool
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(0)
```

WAL plus a 5s busy timeout gives you concurrent reads, serialized writes, and graceful retry. This is sufficient for thousands of users on one server as long as no single transaction holds the write lock for long.

**Watch-outs in current code:**
- `DuplicatePack` (`packs.go:667`) opens a transaction and runs many queries inside it. Fine, but if it grows further it could become a write-lock hot-spot.
- `BulkUpdateItems` (`items.go:951`) also transactional — same caveat.
- The CSRF token table is hot (insert per page render, delete per validate). On WAL this is fine, but it's a candidate for cleanup if it ever shows up in profiling.

---

## 6. OPEN QUESTIONS — sharing & permissions

> **These need answers before Phase C begins. Each one materially changes the schema or the handler logic.**

### 6.1 Permission model granularity
Is the 4-tier model (view / add / edit / admin) final, or do you want fewer/more? Specifically:
- **Q:** Is "add" distinct from "edit" because shared users should be able to add items but not edit/remove existing ones? Or is it about adding *items they own* vs. adding *anyone's items*?
- **Q:** What can `admin` do that the owner cannot? Or is `admin` "everything except delete the pack and re-share"?

### 6.2 Sharing target
- **Q:** Share by username, email, or invite link? (Email implies email lookup; invite link implies a tokenised join URL.)
- **Q:** Can a pack be shared publicly to "anyone with the link can edit," or only to specific accounts?

### 6.3 Cascading on owner deletion
- **Q:** When a pack owner deletes a pack that's shared with N users, what happens?
  - (a) Pack is deleted; shared users lose access (current `ON DELETE CASCADE` behavior; their UI just shows "pack no longer exists").
  - (b) Pack is preserved in a "deleted by owner" archive for shared users.
  - (c) Owner cannot delete a shared pack while it's still shared — must revoke first.

### 6.4 Item ownership when shared user adds (CRITICAL)
A shared user with `add` permission clicks "add my tent to this pack." The existing schema has `pack_items.item_id → items.id`, and `items.user_id` is the owner.
- **Option A:** They add their own items only. Items in the pack now reference items belonging to multiple users. Display has to show "Alice's Tent" vs. "Bob's Sleeping bag." Deleting Alice cascades delete her items, partially gutting the shared pack.
- **Option B:** Adding to a shared pack snapshots the item into the pack-owner's inventory. Avoids cross-user references. Costs disk; loses the link back to the original.
- **Option C:** Items become first-class shared objects (own ACL table). Most powerful, biggest schema change.
- **Q:** Which?

### 6.5 Shared user's own pack list
- **Q:** Does `GetPacks` for user X return packs shared *with* X? If yes — distinct visual treatment? Sort priority?

### 6.6 Sub-items
The brief mentions a `pack_sub_item_checks` table. I have no schema to evaluate.
- **Q:** What is a "sub-item"? An item-of-an-item (e.g., the stove inside the cookset)? A checklist sub-step? A tag-like grouping?
- **Q:** Why pack-scoped check state? If a user packs the same item in two packs, are the sub-items independent in each pack? (If yes, the table makes sense; if no, the table belongs on `items` or on `item_links`.)
- **Q:** Could the existing `item_links` table (`database.go:904`) cover this? It already links items hierarchically.

### 6.7 Activity log / audit
- **Q:** Should shared packs have an activity feed ("Bob added Tarp at 14:32")? This is a separate table and a non-trivial UI but is typical for collaborative apps.

---

## 7. OPEN QUESTIONS — cross-platform & mobile

### 7.1 Native shell technology (CRITICAL — biggest single decision)
- **Option A — Native Swift + native Kotlin.** Best UX, most code, two teams' worth of work.
- **Option B — React Native.** One JS codebase, native components, shares some logic with web if web becomes a SPA.
- **Option C — Flutter.** One Dart codebase, custom rendering, no JS reuse.
- **Option D — Capacitor / PWA-in-a-shell.** Wraps the existing web pages; near-zero new code. UX is "web in a frame."
- **Option E — Skip native, ship a great PWA.** Installable on iOS and Android; no app stores; no Apple developer fees; no review cycle.
- **Q:** Which path? The answer is a function of (a) how much you care about app-store presence, (b) whether you need native APIs (push, biometrics, share sheet, GPX file pickers), (c) how much engineering time you have.

### 7.2 Offline support
- **Q:** Should mobile clients work offline? Editing a packing list while in the woods is a real use case for this app specifically.
- If yes: the API needs idempotency keys, conflict resolution, and a sync model. This is **major scope**. Probably defer to v2.
- If no: the mobile experience is read-only without connectivity, which for this app is a real product compromise.

### 7.3 Authentication UX on mobile
- **Q:** OK to require login on every install? Or do we want refresh tokens with long-lived (30d+) access tokens? Current sessions are 7d sliding — would extending to 30d be acceptable for native?
- **Q:** Biometric unlock (FaceID / fingerprint) for the app — desired in v1 or later?

### 7.4 Push notifications
- Required if shared packs have any "real-time" feel.
- **Q:** v1: no push, manual refresh? v1.5: push when someone edits a shared pack? v2: full push for trip updates, share invites, etc.?

### 7.5 GPX & file uploads on mobile
The trip feature already supports GPX upload via `multipart/form-data` (`handlers/trips.go`). Mobile file pickers work, but iOS sandboxing makes this fiddly.
- **Q:** Is GPX upload an MVP-mobile feature or v2?

### 7.6 Public pack pages
`/p/:short_id` already works without auth. Mobile native could simply open these in an in-app browser — should they?
- **Q:** Native deep links for `/p/:short_id` and `/t/:short_id`?

---

## 8. OPEN QUESTIONS — real-time & sharing UX

### 8.1 Conflict resolution
Two users editing the same pack:
- **Option A — Last-write-wins.** Simple, occasional surprise edits.
- **Option B — Optimistic locking via `updated_at`.** Reject the second writer, ask them to refresh. Annoying but safe.
- **Option C — Operational-transform / CRDT.** Big-tech-collab-doc level effort. Out of scope.
- **Q:** A or B?

### 8.2 Update propagation to other devices
- **Option A — Polling on focus / pull-to-refresh.** Trivial. Recommended for v1.
- **Option B — Long-poll with `If-Modified-Since`.** Cheap, no WebSocket infra.
- **Option C — Server-Sent Events (SSE).** One-way streaming, dead simple in Go, works through proxies.
- **Option D — WebSockets.** Two-way, more infra, real-time presence indicators.
- **Q:** A for v1 (Phase D)? Bump to C/D when shared packs see real co-editing?

---

## 9. OPEN QUESTIONS — meta

### 9.1 Test strategy
- Existing tests: `database_test.go` (290 lines, ~6 tests), `inventory_test.go` (handler-level).
- **Q:** Acceptable target before Phase C ships? My recommendation: every new `database.*` function gets a unit test, every API route gets a smoke test, no requirement on existing untested code.

### 9.2 API versioning
- Recommendation: `/api/v1`, no deprecation policy until there's a `v2`.
- **Q:** Agree?

### 9.3 Backwards compatibility
- **Q:** Is breaking the existing CSV export/import format permitted in this work? (It's currently 5/10/11/12 column tolerant — that's already accumulating cruft.)

### 9.4 Naming & semantics drift
- `pack.is_locked` is described in DB code as the toggle for "archive status" (`packs.go:1057`). Two different mental models for one column.
- **Q:** Which one is canonical? Worth fixing the names before mobile ships, since the API will expose them.

---

## 10. Suggested next session

After you answer 6.1, 6.4, 7.1, 7.2 (the four critical ones), I can:

1. Write a concrete schema migration for sharing (with a real `pack_shares` table and the `EffectivePermission` function).
2. Write the actual `/api/v1` route table with request/response shapes.
3. Decide whether sub-items folds into `item_links` or needs `pack_sub_item_checks` — I cannot decide that without 6.6.
4. Pick a real shell technology in 7.1 and scope the iOS/Android work.

Until those answers exist, any code written would be guesswork.

---

## Appendix A — Concrete code-level items I'd start with regardless

These are independent of every OPEN QUESTION:

| # | Change | File | Risk |
|---|---|---|---|
| 1 | Add `_journal_mode=WAL&_busy_timeout=5000` | `database/database.go:13` | none |
| 2 | Replace `if strings.Contains(err.Error(), "not found")` with typed `ErrNotFound` | all of `handlers/` | low; mechanical |
| 3 | Stop using `?error=foo` redirects; return JSON for delete/duplicate | `handlers/inventory.go:558`, `:610` | low; needs JS adjustment |
| 4 | Add `schema_migrations` table; gate each existing migration on it | `database/database.go:25` | medium; requires careful first deploy |
| 5 | Move `handlePatchItem` and the existing `/api/items/*` routes under `/api/v1/items` with redirects from old paths | `handlers/handlers.go:64` | low; old paths can stay via aliasing |

These can ship independently and de-risk Phase B.
