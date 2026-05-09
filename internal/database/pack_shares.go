package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"carryless/internal/models"
)

// GetUserSharePermission returns the effective permission level for userID on packID.
// Returns "owner" if the user owns the pack, the share permission if shared, or "none".
func GetUserSharePermission(db *sql.DB, packID string, userID int) string {
	var ownerID int
	err := db.QueryRow(`SELECT user_id FROM packs WHERE id = ?`, packID).Scan(&ownerID)
	if err != nil {
		return "none"
	}
	if ownerID == userID {
		return "owner"
	}

	var permission string
	err = db.QueryRow(
		`SELECT permission FROM pack_shares WHERE pack_id = ? AND shared_with_user_id = ?`,
		packID, userID,
	).Scan(&permission)
	if err != nil {
		return "none"
	}
	return permission
}

// CanWrite returns true for permissions that allow adding/removing items.
func CanWrite(permission string) bool {
	return permission == "owner" || permission == "admin" || permission == "edit" || permission == "add"
}

// CanEdit returns true for permissions that allow editing pack metadata.
func CanEdit(permission string) bool {
	return permission == "owner" || permission == "admin" || permission == "edit"
}

// CanAdmin returns true for permissions that allow managing shares.
func CanAdmin(permission string) bool {
	return permission == "owner" || permission == "admin"
}

func CreatePackShare(db *sql.DB, packID string, ownerID, targetUserID int, permission string) error {
	_, err := db.Exec(
		`INSERT INTO pack_shares (pack_id, owner_id, shared_with_user_id, permission)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(pack_id, shared_with_user_id) DO UPDATE SET permission = excluded.permission, updated_at = CURRENT_TIMESTAMP`,
		packID, ownerID, targetUserID, permission,
	)
	if err != nil {
		return fmt.Errorf("failed to create pack share: %w", err)
	}
	return nil
}

func UpdatePackShare(db *sql.DB, packID string, ownerID, targetUserID int, permission string) error {
	result, err := db.Exec(
		`UPDATE pack_shares SET permission = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE pack_id = ? AND owner_id = ? AND shared_with_user_id = ?`,
		permission, packID, ownerID, targetUserID,
	)
	if err != nil {
		return fmt.Errorf("failed to update pack share: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("share not found")
	}
	return nil
}

func DeletePackShare(db *sql.DB, packID string, ownerID, targetUserID int) error {
	result, err := db.Exec(
		`DELETE FROM pack_shares WHERE pack_id = ? AND owner_id = ? AND shared_with_user_id = ?`,
		packID, ownerID, targetUserID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete pack share: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("share not found")
	}
	return nil
}

func GetPackShares(db *sql.DB, packID string) ([]models.PackShare, error) {
	rows, err := db.Query(
		`SELECT ps.id, ps.pack_id, ps.owner_id, ps.shared_with_user_id, ps.permission,
		        u.id, u.username, u.email, ps.created_at, ps.updated_at
		 FROM pack_shares ps
		 JOIN users u ON u.id = ps.shared_with_user_id
		 WHERE ps.pack_id = ?
		 ORDER BY ps.created_at ASC`,
		packID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query pack shares: %w", err)
	}
	defer rows.Close()

	var shares []models.PackShare
	for rows.Next() {
		var s models.PackShare
		u := &models.User{}
		err := rows.Scan(
			&s.ID, &s.PackID, &s.OwnerID, &s.SharedWithUserID, &s.Permission,
			&u.ID, &u.Username, &u.Email,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pack share: %w", err)
		}
		s.SharedWithUser = u
		shares = append(shares, s)
	}
	return shares, rows.Err()
}

func GetPacksSharedWithUser(db *sql.DB, userID int) ([]models.Pack, error) {
	rows, err := db.Query(
		`SELECT p.id, p.user_id, p.name, COALESCE(p.note,''), p.is_public,
		        COALESCE(p.is_locked,FALSE), COALESCE(p.short_id,''), p.created_at, p.updated_at,
		        ps.permission
		 FROM pack_shares ps
		 JOIN packs p ON p.id = ps.pack_id
		 WHERE ps.shared_with_user_id = ?
		 ORDER BY p.updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query shared packs: %w", err)
	}
	defer rows.Close()

	var packs []models.Pack
	for rows.Next() {
		var p models.Pack
		err := rows.Scan(
			&p.ID, &p.UserID, &p.Name, &p.Note, &p.IsPublic,
			&p.IsLocked, &p.ShortID, &p.CreatedAt, &p.UpdatedAt,
			&p.UserPermission,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shared pack: %w", err)
		}
		packs = append(packs, p)
	}
	return packs, rows.Err()
}

func GetUserByUsernameOrEmail(db *sql.DB, query string) (*models.User, error) {
	user := &models.User{}
	err := db.QueryRow(
		`SELECT id, username, email, COALESCE(currency,'$'), COALESCE(is_admin,FALSE),
		        COALESCE(is_activated,FALSE), created_at, updated_at
		 FROM users WHERE username = ? OR email = ?`,
		query, query,
	).Scan(
		&user.ID, &user.Username, &user.Email, &user.Currency,
		&user.IsAdmin, &user.IsActivated, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	return user, nil
}

func SearchUsers(db *sql.DB, query string, excludeUserID int) ([]models.User, error) {
	like := "%" + query + "%"
	rows, err := db.Query(
		`SELECT id, username, email FROM users
		 WHERE (username LIKE ? OR email LIKE ?) AND id != ? AND is_activated = TRUE
		 ORDER BY username LIMIT 10`,
		like, like, excludeUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func CreatePackInvite(db *sql.DB, packID string, ownerID int, permission string) (*models.PackInvite, error) {
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate invite token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	result, err := db.Exec(
		`INSERT INTO pack_invites (pack_id, owner_id, token, permission, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		packID, ownerID, token, permission, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create pack invite: %w", err)
	}
	id, _ := result.LastInsertId()
	return &models.PackInvite{
		ID:         int(id),
		PackID:     packID,
		OwnerID:    ownerID,
		Token:      token,
		Permission: permission,
		ExpiresAt:  expiresAt,
		CreatedAt:  time.Now(),
	}, nil
}

func GetPackInviteByToken(db *sql.DB, token string) (*models.PackInvite, error) {
	inv := &models.PackInvite{}
	err := db.QueryRow(
		`SELECT id, pack_id, owner_id, token, permission, expires_at, created_at
		 FROM pack_invites WHERE token = ? AND expires_at > CURRENT_TIMESTAMP`,
		token,
	).Scan(&inv.ID, &inv.PackID, &inv.OwnerID, &inv.Token, &inv.Permission, &inv.ExpiresAt, &inv.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invite not found or expired")
		}
		return nil, fmt.Errorf("failed to query invite: %w", err)
	}
	return inv, nil
}

// RedeemPackInvite creates a pack_share row for the redeemer. The invite token
// remains valid until it expires so the link can be shared with multiple people.
func RedeemPackInvite(db *sql.DB, token string, redeemerUserID int) error {
	inv, err := GetPackInviteByToken(db, token)
	if err != nil {
		return err
	}
	if inv.OwnerID == redeemerUserID {
		return fmt.Errorf("cannot redeem your own invite")
	}
	return CreatePackShare(db, inv.PackID, inv.OwnerID, redeemerUserID, inv.Permission)
}

func GetPackInvites(db *sql.DB, packID string) ([]models.PackInvite, error) {
	rows, err := db.Query(
		`SELECT id, pack_id, owner_id, token, permission, expires_at, created_at
		 FROM pack_invites WHERE pack_id = ? AND expires_at > CURRENT_TIMESTAMP
		 ORDER BY created_at DESC`,
		packID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query pack invites: %w", err)
	}
	defer rows.Close()

	var invites []models.PackInvite
	for rows.Next() {
		var inv models.PackInvite
		if err := rows.Scan(&inv.ID, &inv.PackID, &inv.OwnerID, &inv.Token,
			&inv.Permission, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan invite: %w", err)
		}
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

// AddItemToPackAsSharedUser adds an item from the requesting user's inventory to a
// pack they have write access to (verified at the handler layer). Unlike AddItemToPack,
// this skips the pack-ownership check and uses the requesting user's ID for item lookup.
func AddItemToPackAsSharedUser(db *sql.DB, packID string, itemID, requestingUserID int) error {
	_, err := GetItem(db, requestingUserID, itemID)
	if err != nil {
		return fmt.Errorf("item not found")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := addSingleItemToPackTx(tx, packID, itemID); err != nil {
		tx.Rollback()
		return err
	}

	linkedItemIDs, err := GetLinkedItemIDs(db, itemID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get linked items: %w", err)
	}
	for _, linkedID := range linkedItemIDs {
		if err := addSingleItemToPackTx(tx, packID, linkedID); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to add linked item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return updatePackTimestamp(db, packID)
}

func DeletePackInvite(db *sql.DB, inviteID, ownerID int) error {
	result, err := db.Exec(
		`DELETE FROM pack_invites WHERE id = ? AND owner_id = ?`,
		inviteID, ownerID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete pack invite: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("invite not found")
	}
	return nil
}
