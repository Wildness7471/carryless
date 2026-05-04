# Collaborative Packing Features — Implementation Plan

## Progress Tracking

Update the status column as work is completed. When resuming after a usage reset, check this table first to know where to continue.

| Phase | Task | Status |
|-------|------|--------|
| 0 | Expand `database_test.go` — auth & sessions | ⬜ Not started |
| 0 | Expand `database_test.go` — categories | ⬜ Not started |
| 0 | Expand `database_test.go` — items | ⬜ Not started |
| 0 | Expand `database_test.go` — packs | ⬜ Not started |
| 0 | Expand `database_test.go` — labels | ⬜ Not started |
| 0 | Expand `database_test.go` — item_links | ⬜ Not started |
| 0 | Expand `database_test.go` — trips | ⬜ Not started |
| 0 | Handler tests — inventory CSV import/export (already partial) | ⬜ Not started |
| 1 | DB migration — `pack_shares` table | ⬜ Not started |
| 1 | `internal/database/pack_shares.go` — CRUD functions | ⬜ Not started |
| 1 | `internal/database/pack_shares_test.go` — tests | ⬜ Not started |
| 1 | `PackShare` model in `internal/models/models.go` | ⬜ Not started |
| 1 | `GET /api/users/search` route + handler | ⬜ Not started |
| 1 | `internal/handlers/pack_shares.go` — share management handlers | ⬜ Not started |
| 1 | Permission middleware helper (`GetUserPackPermission`) | ⬜ Not started |
| 1 | Enforce permissions in existing pack handlers | ⬜ Not started |
| 1 | `templates/pack_shares.html` — share management UI | ⬜ Not started |
| 1 | Show shared packs on `/packs` page | ⬜ Not started |
| 1 | Enforce owner-only inventory delete (shared users blocked) | ⬜ Not started |
| 1 | Add "Remove from inventory" option in pack detail item modal (owner only) | ⬜ Not started |
| 1 | Handler tests — permission enforcement | ⬜ Not started |
| 2 | `GET /packs/compare` route + `handlePackCompare` | ⬜ Not started |
| 2 | `templates/pack_compare.html` — side-by-side columns | ⬜ Not started |
| 2 | `static/js/pack_compare.js` — drag-and-drop + per-column add | ⬜ Not started |
| 2 | Pack picker on `/packs` page (checkbox multi-select + Compare button) | ⬜ Not started |
| 2 | Handler tests — compare permission checks | ⬜ Not started |
| 3 | DB migration — `item_sub_items` table | ⬜ Not started |
| 3 | DB migration — `pack_sub_item_checks` table | ⬜ Not started |
| 3 | `internal/database/sub_items.go` — CRUD + check toggle | ⬜ Not started |
| 3 | `internal/database/sub_items_test.go` — tests | ⬜ Not started |
| 3 | `ItemSubItem` model + extend `Item` struct | ⬜ Not started |
| 3 | Sub-item editor in item create/edit pages | ⬜ Not started |
| 3 | Sub-item API routes (create, update, delete, reorder) | ⬜ Not started |
| 3 | Pack checklist — show sub-items with indeterminate parent state | ⬜ Not started |
| 3 | `POST /packs/:id/items/:item_id/sub-items/:sub_id/toggle` route | ⬜ Not started |

**Status key:** ⬜ Not started | 🔄 In progress | ✅ Done

---

## Context

Carryless is a Go/Gin server-rendered app (SQLite, `html/template`, vanilla JS). Three major features are being added for collaborative group packing. Tests are written database-layer-first.

---

## Phase 0: Test Foundation (Database Layer First)

Write comprehensive tests for all existing database functions before any new features.

### File to expand
- `carryless/internal/database/database_test.go`

Functions to cover (match existing pattern — in-memory SQLite, standard `testing` package):
- `auth.go`: session expiry edge cases, duplicate username/email errors
- `packs.go`: CreatePack, GetPack, UpdatePack, DeletePack, DuplicatePack, GetPublicPack, SetPackPublic, TogglePackLock
- `items.go`: CreateItem, GetItem, UpdateItem, DeleteItem, DuplicateItem, GetItemsByUser, BulkDelete, BulkEdit
- `categories.go`: UpdateCategory, DeleteCategory, error on duplicate name
- `trips.go`: all CRUD, AddPackToTrip, checklist CRUD, transport CRUD
- `labels.go` / `pack_level_labels.go`: all label CRUD
- `item_links.go`: CreateItemLink, DeleteItemLink, GetItemLinks, self-link constraint

### Test helper pattern (already exists)
```go
func setupTestDB(t *testing.T) *sql.DB { /* in-memory SQLite */ }
```

---

## Phase 1: Pack Sharing

### New DB table: `pack_shares`
```sql
CREATE TABLE pack_shares (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    pack_id               TEXT NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
    owner_id              INTEGER NOT NULL REFERENCES users(id),
    shared_with_user_id   INTEGER NOT NULL REFERENCES users(id),
    permission            TEXT NOT NULL CHECK(permission IN ('view','add','edit','admin')),
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pack_id, shared_with_user_id)
);
```

Add migration in `carryless/internal/database/database.go`.

### New model in `carryless/internal/models/models.go`
```go
type PackShare struct {
    ID               int
    PackID           string
    OwnerID          int
    SharedWithUserID int
    Permission       string  // "view" | "add" | "edit" | "admin"
    SharedWithUser   *User
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

### New file: `carryless/internal/database/pack_shares.go`
Functions:
- `CreatePackShare(db, packID, ownerID, targetUserID, permission) error`
- `UpdatePackShare(db, packID, ownerID, targetUserID, permission) error`
- `DeletePackShare(db, packID, ownerID, targetUserID) error`
- `GetPackShares(db, packID) ([]PackShare, error)`
- `GetPacksSharedWithUser(db, userID) ([]Pack, error)`
- `GetUserByUsernameOrEmail(db, query) (*User, error)`
- `GetUserSharePermission(db, packID, userID) (string, error)` — returns "owner"|"admin"|"edit"|"add"|"view"|"none"

### New file: `carryless/internal/handlers/pack_shares.go`
Routes to register in `carryless/internal/handlers/handlers.go`:
- `GET  /packs/:id/shares`          → show share management page
- `POST /packs/:id/shares`          → invite user (body: username_or_email, permission)
- `POST /packs/:id/shares/:user_id` → update permission
- `DELETE /packs/:id/shares/:user_id` → revoke access
- `GET  /api/users/search?q=`       → JSON user search (for autocomplete)

### Changes to `carryless/internal/handlers/packs.go`
- `handlePackDetail`: load `PackShares`, pass to template; check permission for edit controls
- `handleAddItemToPack`, `handleRemoveItemFromPack`, worn toggle routes: require "add" or higher
- `handlePacks`: also query `GetPacksSharedWithUser` and display in a "Shared with me" section

### Item deletion rules — critical distinction

**Removing an item from a pack** (`DELETE /packs/:id/items/:item_id`) — unlinks the item from the pack only. The item remains in the owner's inventory. Shared users with "add" permission or higher may do this.

**Deleting an item from inventory** (`POST /inventory/items/:id/delete`) — permanently removes the item from the owner's inventory and all packs it appears in. **Only the item's owner (the user who created it) may do this.** Shared users are never permitted to delete inventory items, regardless of their pack permission level. Enforce this with an ownership check (`item.UserID == currentUser.ID`) in `handleDeleteItem`.

**New: "Remove from inventory" from pack view** — the pack detail page currently has no way to delete an item from inventory without navigating to `/inventory`. Add a "Remove from inventory" option (visible only to the item's owner) in the item's edit popover/modal on `pack_detail.html`. This calls the existing `handleDeleteItem` route. Shared users see no such option.

### New template: `carryless/templates/pack_shares.html`
- Search box (username or email) with AJAX autocomplete → `/api/users/search?q=`
- Permission tier dropdown: View / Add items / Full edit / Admin
- List of current shares with edit/revoke buttons

---

## Phase 2: Multi-Pack Side-by-Side Comparison

### Where it lives
`/packs/compare?packs=uuid1,uuid2,uuid3` — accessible from the `/packs` page via a multi-select + Compare button. Supports 2 or more packs. Works for owned packs and packs shared with at least "view" permission.

### New handler in `carryless/internal/handlers/packs.go`
`handlePackCompare`:
- Read `?packs=` query param (comma-separated UUIDs, max ~6 for readability)
- Verify user has at least "view" permission on each
- Load all packs with items
- Render `templates/pack_compare.html`

Route: `GET /packs/compare` (add to authorized group in `handlers.go`)

### New template: `carryless/templates/pack_compare.html`
- Horizontal columns (one per pack), sticky header row
- Each column shows: pack name, total weight, item list, per-column add-item search
- Drag-and-drop: HTML5 native drag API — dragging an item moves it (calls `DELETE` on source + `POST` on target via existing routes)
- Columns respect permission: "view"-only columns have drag targets and add forms disabled
- Respond to column count to use scrollable flex row (no max width, scroll horizontally on small screens)

### New file: `carryless/static/js/pack_compare.js`
- Drag-and-drop logic between columns
- Per-column add-item AJAX (reuse existing autocomplete pattern from `pack_detail.html`)
- Live weight total updates after moves

### `/packs` page changes
- Add checkbox to each pack row
- "Compare selected" button appears when 2+ are checked, navigates to `/packs/compare?packs=...`

---

## Phase 3: Sub-Items

### Scope
Sub-items are lightweight checklist children that live inside a parent inventory item. They are NOT standalone inventory items. They appear in pack prep/checklist mode only.

### New DB tables (add to migration in `database.go`)
```sql
CREATE TABLE item_sub_items (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    item_id    INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE pack_sub_item_checks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    pack_id     TEXT NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
    sub_item_id INTEGER NOT NULL REFERENCES item_sub_items(id) ON DELETE CASCADE,
    is_checked  BOOLEAN NOT NULL DEFAULT 0,
    UNIQUE(pack_id, sub_item_id)
);
```

### Model changes in `carryless/internal/models/models.go`
```go
type ItemSubItem struct {
    ID        int
    ItemID    int
    Name      string
    SortOrder int
    CreatedAt time.Time
    UpdatedAt time.Time
}
```
Add `SubItems []ItemSubItem` to `Item` struct.

### New file: `carryless/internal/database/sub_items.go`
Functions:
- `CreateSubItem(db, itemID, name, sortOrder) (*ItemSubItem, error)`
- `UpdateSubItem(db, id, name, sortOrder) error`
- `DeleteSubItem(db, id) error`
- `GetSubItemsByItem(db, itemID) ([]ItemSubItem, error)`
- `ReorderSubItems(db, itemID, orderedIDs []int) error`
- `ToggleSubItemCheck(db, packID, subItemID) error`
- `GetSubItemChecks(db, packID, itemIDs []int) (map[int]bool, error)`

### Handler changes: `carryless/internal/handlers/inventory.go`
- Load sub-items in `handleEditItemPage`
- Save sub-items (add/remove) in `handleUpdateItem`

New API routes (add to `handlers.go`):
- `POST   /api/items/:id/sub-items`                → `handleCreateSubItem`
- `PUT    /api/items/:id/sub-items/:sub_id`         → `handleUpdateSubItem`
- `DELETE /api/items/:id/sub-items/:sub_id`         → `handleDeleteSubItem`
- `POST   /api/items/:id/sub-items/reorder`         → `handleReorderSubItems`
- `POST   /packs/:id/items/:item_id/sub-items/:sub_id/toggle` → `handleToggleSubItemCheck`

### Pack checklist/prep mode changes
In `carryless/templates/pack_detail.html` (checklist view) and/or `pack_checklist.html`:
- Under each item, show indented sub-item list with individual checkboxes
- Parent item checkbox uses JS `indeterminate` property when partially checked
- Show completion count next to parent (e.g., "2/3")
- Parent cannot be marked fully checked while sub-items are unchecked (enforced in JS)

---

## Critical Files to Modify

| File | Change |
|------|--------|
| `carryless/internal/database/database.go` | Add 3 new table migrations |
| `carryless/internal/models/models.go` | Add `PackShare`, `ItemSubItem`; extend `Item` |
| `carryless/internal/handlers/handlers.go` | Register all new routes |
| `carryless/internal/handlers/packs.go` | Permission checks, compare handler, shared packs on list |
| `carryless/internal/handlers/inventory.go` | Sub-item CRUD in item edit flow |
| `carryless/internal/database/database_test.go` | Expand to full existing coverage |
| `carryless/templates/packs.html` | Multi-select + Compare button |
| `carryless/templates/pack_detail.html` | Sub-item checklist UI |

## New Files to Create

| File | Purpose |
|------|---------|
| `carryless/internal/database/pack_shares.go` | Pack sharing DB functions |
| `carryless/internal/database/sub_items.go` | Sub-item DB functions |
| `carryless/internal/handlers/pack_shares.go` | Share management handlers |
| `carryless/internal/database/pack_shares_test.go` | Share DB tests |
| `carryless/internal/database/sub_items_test.go` | Sub-item DB tests |
| `carryless/templates/pack_shares.html` | Sharing management UI |
| `carryless/templates/pack_compare.html` | Side-by-side compare UI |
| `carryless/static/js/pack_compare.js` | Compare page drag-and-drop JS |

---

## Verification

- `cd carryless && go test ./...` — all tests must pass at each phase before moving to the next
- Phase 1: create a pack, share it with a second test account at each tier, verify restrictions hold
- Phase 2: select 3 packs, open compare view, drag item between columns, confirm DB updated correctly
- Phase 3: add sub-items to an inventory item, open pack checklist, verify partial → full completion states
- Cascade deletes: delete a pack → `pack_shares` rows removed; delete an item → `item_sub_items` removed
