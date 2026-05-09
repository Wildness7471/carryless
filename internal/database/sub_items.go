package database

import (
	"database/sql"
	"fmt"

	"carryless/internal/models"
)

func CreateSubItem(db *sql.DB, itemID int, name string) (*models.ItemSubItem, error) {
	var maxOrder int
	_ = db.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM item_sub_items WHERE item_id = ?`, itemID).Scan(&maxOrder)

	result, err := db.Exec(
		`INSERT INTO item_sub_items (item_id, name, sort_order) VALUES (?, ?, ?)`,
		itemID, name, maxOrder+1,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sub-item: %w", err)
	}
	id, _ := result.LastInsertId()
	sub := &models.ItemSubItem{ID: int(id), ItemID: itemID, Name: name, SortOrder: maxOrder + 1}
	return sub, nil
}

func UpdateSubItem(db *sql.DB, subItemID, ownerUserID int, name string) error {
	result, err := db.Exec(
		`UPDATE item_sub_items SET name = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND item_id IN (SELECT id FROM items WHERE user_id = ?)`,
		name, subItemID, ownerUserID,
	)
	if err != nil {
		return fmt.Errorf("failed to update sub-item: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("sub-item not found")
	}
	return nil
}

func DeleteSubItem(db *sql.DB, subItemID, ownerUserID int) error {
	result, err := db.Exec(
		`DELETE FROM item_sub_items WHERE id = ? AND item_id IN (SELECT id FROM items WHERE user_id = ?)`,
		subItemID, ownerUserID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete sub-item: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("sub-item not found")
	}
	return nil
}

func GetSubItems(db *sql.DB, itemID int) ([]models.ItemSubItem, error) {
	rows, err := db.Query(
		`SELECT id, item_id, name, sort_order, created_at, updated_at
		 FROM item_sub_items WHERE item_id = ? ORDER BY sort_order, id`,
		itemID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query sub-items: %w", err)
	}
	defer rows.Close()

	var subs []models.ItemSubItem
	for rows.Next() {
		var s models.ItemSubItem
		if err := rows.Scan(&s.ID, &s.ItemID, &s.Name, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan sub-item: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// GetSubItemsForPack returns sub-items for a given item, with IsChecked populated
// from the pack_sub_item_checks table for the given packID.
func GetSubItemsForPack(db *sql.DB, itemID int, packID string) ([]models.ItemSubItem, error) {
	rows, err := db.Query(
		`SELECT s.id, s.item_id, s.name, s.sort_order, s.created_at, s.updated_at,
		        COALESCE(c.is_checked, 0)
		 FROM item_sub_items s
		 LEFT JOIN pack_sub_item_checks c ON c.sub_item_id = s.id AND c.pack_id = ?
		 WHERE s.item_id = ?
		 ORDER BY s.sort_order, s.id`,
		packID, itemID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query sub-items for pack: %w", err)
	}
	defer rows.Close()

	var subs []models.ItemSubItem
	for rows.Next() {
		var s models.ItemSubItem
		var isChecked int
		if err := rows.Scan(&s.ID, &s.ItemID, &s.Name, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt, &isChecked); err != nil {
			return nil, fmt.Errorf("failed to scan sub-item: %w", err)
		}
		s.IsChecked = isChecked != 0
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// GetSubItemsForPackBulk returns all sub-items for a slice of item IDs,
// keyed by item ID, with IsChecked populated from pack_sub_item_checks for the given pack.
func GetSubItemsForPackBulk(db *sql.DB, packID string, itemIDs []int) (map[int][]models.ItemSubItem, error) {
	if len(itemIDs) == 0 {
		return map[int][]models.ItemSubItem{}, nil
	}

	// Build IN clause
	args := make([]interface{}, 0, len(itemIDs)+1)
	args = append(args, packID)
	ph := make([]string, len(itemIDs))
	for i, id := range itemIDs {
		ph[i] = "?"
		args = append(args, id)
	}

	query := `SELECT s.id, s.item_id, s.name, s.sort_order, s.created_at, s.updated_at,
	                 COALESCE(c.is_checked, 0)
	          FROM item_sub_items s
	          LEFT JOIN pack_sub_item_checks c ON c.sub_item_id = s.id AND c.pack_id = ?
	          WHERE s.item_id IN (` + joinStrings(ph, ",") + `)
	          ORDER BY s.item_id, s.sort_order, s.id`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk query sub-items: %w", err)
	}
	defer rows.Close()

	result := make(map[int][]models.ItemSubItem)
	for rows.Next() {
		var s models.ItemSubItem
		var isChecked int
		if err := rows.Scan(&s.ID, &s.ItemID, &s.Name, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt, &isChecked); err != nil {
			return nil, fmt.Errorf("failed to scan bulk sub-item: %w", err)
		}
		s.IsChecked = isChecked != 0
		result[s.ItemID] = append(result[s.ItemID], s)
	}
	return result, rows.Err()
}

func joinStrings(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

func ToggleSubItemCheck(db *sql.DB, packID string, subItemID int, isChecked bool) error {
	checked := 0
	if isChecked {
		checked = 1
	}
	_, err := db.Exec(
		`INSERT INTO pack_sub_item_checks (pack_id, sub_item_id, is_checked, updated_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(pack_id, sub_item_id) DO UPDATE SET is_checked = excluded.is_checked, updated_at = CURRENT_TIMESTAMP`,
		packID, subItemID, checked,
	)
	if err != nil {
		return fmt.Errorf("failed to toggle sub-item check: %w", err)
	}
	return nil
}
