# Carryless — Collaborative Features Implementation

## Resume Instructions

If you hit a usage limit, check this file first. The status table shows exactly where to continue. Run `go test ./...` to confirm current state before resuming.

---

## Progress Tracker

| Phase | Task | Status |
|-------|------|--------|
| 0 | Auth edge case tests (duplicate user/email, session expiry) | ✅ |
| 0 | Pack advanced tests (duplicate, lock, public, add/remove items) | ✅ |
| 0 | Item advanced tests (duplicate, bulk delete/edit) | ✅ |
| 0 | Category edge case tests (duplicate name error) | ✅ |
| 0 | Label tests (pack labels, user pack labels, assign/remove) | ✅ |
| 0 | Item link tests (create, delete, get, self-link error) | ✅ |
| 0 | Trip tests (CRUD, add pack, checklist, transport) | ✅ |
| 1a | WAL mode + busy_timeout added to SQLite DSN | ✅ |
| 1a | `CreateBearerToken` + `ValidateBearerToken` in auth.go | ✅ |
| 1a | Bearer token accepted in middleware (alongside cookie) | ✅ |
| 1a | `POST /api/auth/token` handler + route | ✅ |
| 1b | `PackShare` + `PackInvite` models in models.go | ✅ |
| 1b | `pack_shares` + `pack_invites` table migrations | ✅ |
| 1b | `internal/database/pack_shares.go` DB functions | ✅ |
| 1b | `internal/database/pack_shares_test.go` tests | ✅ |
| 1b | `packPermission()` helper in handlers | ✅ |
| 1b | `respond()` content-negotiation helper in handlers | ✅ |
| 1b | Existing pack handlers updated with permission checks | ✅ |
| 1b | `internal/handlers/pack_shares.go` share handlers | ✅ |
| 1b | Routes registered in handlers.go | ✅ |
| 1b | `GET /api/users/search` JSON endpoint | ✅ |
| 1b | `templates/pack_shares.html` sharing UI | ✅ |
| 1b | `/packs` page shows "Shared with me" section | ✅ |
| 1b | Owner-only inventory delete enforced | ✅ |
| 2 | `handlePackCompare` handler | ⬜ |
| 2 | `GET /packs/compare` route registered | ⬜ |
| 2 | `templates/pack_compare.html` side-by-side UI | ⬜ |
| 2 | `static/js/pack_compare.js` drag-and-drop + move buttons | ⬜ |
| 2 | `/packs` page: checkbox multi-select + Compare button | ⬜ |
| 3 | `item_sub_items` + `pack_sub_item_checks` migrations | ⬜ |
| 3 | `ItemSubItem` model + `SubItems` field on `Item` | ⬜ |
| 3 | `internal/database/sub_items.go` DB functions | ⬜ |
| 3 | `internal/database/sub_items_test.go` tests | ⬜ |
| 3 | Sub-item editor in item create/edit pages | ⬜ |
| 3 | Sub-item API routes registered | ⬜ |
| 3 | Pack checklist UI: indented sub-items, indeterminate state | ⬜ |
| 3 | `localStorage` offline fallback + sync-on-reconnect | ⬜ |

**Status key:** ⬜ Not started | 🔄 In progress | ✅ Done

---

## Key Decisions (from planning session)

- **Offline:** Near-term requirement. Phase 3 adds localStorage fallback for sub-item check state with sync-on-reconnect.
- **Mobile/JSON:** Use content-negotiation (`Accept: application/json`) on pack endpoints so the same routes serve both the web app and a future mobile client. No frontend rewrite.
- **Bearer tokens:** Add `POST /api/auth/token` in Phase 1a so mobile clients can exchange a cookie session for a long-lived token.
- **Invite links:** In Phase 1b (not deferred). Shareable URL anyone can open to join a pack.
- **Compare UI:** Both HTML5 drag-and-drop AND a "Move to →" button for mobile/accessibility.
- **WAL mode:** Add as first Phase 1a commit. DSN: `?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL`

---

## Architecture Notes

### Content Negotiation Pattern (Phase 1b+)
```go
func respond(c *gin.Context, status int, tmpl string, data gin.H) {
    if c.GetHeader("Accept") == "application/json" {
        c.JSON(status, data)
    } else {
        c.HTML(status, tmpl, data)
    }
}
```

### Permission Helper Pattern (Phase 1b+)
```go
// Returns "owner"|"admin"|"edit"|"add"|"view"|"none" + the pack (nil if none)
func packPermission(c *gin.Context, packID string) (string, *models.Pack) { ... }
```

### New Tables Summary
```sql
-- Phase 1b
pack_shares   (id, pack_id, owner_id, shared_with_user_id, permission, created_at, updated_at)
pack_invites  (id, pack_id, owner_id, token, permission, expires_at, created_at)

-- Phase 3
item_sub_items       (id, item_id, name, sort_order, created_at, updated_at)
pack_sub_item_checks (id, pack_id, sub_item_id, is_checked)
```

### New Files Summary
| File | Phase | Purpose |
|------|-------|---------|
| `internal/database/pack_shares.go` | 1b | Share + invite DB functions |
| `internal/database/pack_shares_test.go` | 1b | Share DB tests |
| `internal/handlers/pack_shares.go` | 1b | Share management handlers |
| `templates/pack_shares.html` | 1b | Sharing UI |
| `templates/pack_compare.html` | 2 | Side-by-side compare UI |
| `static/js/pack_compare.js` | 2 | Drag-and-drop + move button JS |
| `internal/database/sub_items.go` | 3 | Sub-item DB functions |
| `internal/database/sub_items_test.go` | 3 | Sub-item DB tests |

### Files Modified Each Phase
| File | Changes |
|------|---------|
| `internal/database/database.go` | WAL DSN (1a), new table migrations (1b, 3) |
| `internal/database/auth.go` | Bearer token functions (1a) |
| `internal/middleware/middleware.go` | Accept bearer token header (1a) |
| `internal/models/models.go` | PackShare, PackInvite (1b); ItemSubItem + SubItems on Item (3) |
| `internal/handlers/handlers.go` | Register all new routes (1a, 1b, 2, 3) |
| `internal/handlers/packs.go` | Permission checks, compare handler, respond() helper (1b, 2) |
| `internal/handlers/inventory.go` | Sub-item CRUD, owner-only delete (1b, 3) |
| `internal/handlers/auth.go` | Bearer token handler (1a) |
| `templates/packs.html` | Shared packs section, compare checkboxes (1b, 2) |
| `templates/pack_detail.html` | Sub-item checklist UI (3) |

---

## Verification Commands

```bash
# Run all tests
cd /home/user/carryless && go test ./...

# Run only DB tests
go test ./internal/database/...

# Run with verbose output
go test ./internal/database/... -v

# Build check
go build ./...
```
