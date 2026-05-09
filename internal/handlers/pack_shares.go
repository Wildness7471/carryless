package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"carryless/internal/database"

	"github.com/gin-gonic/gin"
)

func handlePackSharesPage(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	userID := c.MustGet("user_id").(int)
	user := c.MustGet("user")

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.HTML(http.StatusNotFound, "404.html", gin.H{"Title": "Not Found - Carryless", "User": user})
		return
	}
	if !database.CanAdmin(perm) {
		forbidden403(c)
		return
	}

	shares, err := database.GetPackShares(db, packID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "pack_shares.html", gin.H{
			"Title": "Share Pack - Carryless", "User": user, "Pack": pack, "Error": "Failed to load shares",
		})
		return
	}

	invites, err := database.GetPackInvites(db, packID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "pack_shares.html", gin.H{
			"Title": "Share Pack - Carryless", "User": user, "Pack": pack, "Error": "Failed to load invites",
		})
		return
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "pack_shares.html", gin.H{
			"Title": "Share Pack - Carryless", "User": user, "Pack": pack, "Error": "Failed to generate token",
		})
		return
	}

	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	requestHost := scheme + "://" + c.Request.Host

	c.HTML(http.StatusOK, "pack_shares.html", gin.H{
		"Title":       "Share Pack - Carryless",
		"User":        user,
		"Pack":        pack,
		"Shares":      shares,
		"Invites":     invites,
		"CSRFToken":   csrfToken.Token,
		"RequestHost": requestHost,
	})
}

func handleCreatePackShare(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	userID := c.MustGet("user_id").(int)

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pack not found"})
		return
	}
	if !database.CanAdmin(perm) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	query := c.PostForm("username_or_email")
	permission := c.PostForm("permission")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username or email required"})
		return
	}
	if permission != "view" && permission != "add" && permission != "edit" && permission != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission"})
		return
	}

	target, err := database.GetUserByUsernameOrEmail(db, query)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if target.ID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot share with yourself"})
		return
	}

	if err := database.CreatePackShare(db, packID, userID, target.ID, permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create share"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "share created", "username": target.Username, "permission": permission})
}

func handleUpdatePackShare(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	userID := c.MustGet("user_id").(int)

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pack not found"})
		return
	}
	if !database.CanAdmin(perm) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	targetUserID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	permission := c.PostForm("permission")
	if permission != "view" && permission != "add" && permission != "edit" && permission != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission"})
		return
	}

	if err := database.UpdatePackShare(db, packID, userID, targetUserID, permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update share"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "share updated"})
}

func handleRevokePackShare(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	userID := c.MustGet("user_id").(int)

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pack not found"})
		return
	}
	if !database.CanAdmin(perm) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	targetUserID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := database.DeletePackShare(db, packID, userID, targetUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke share"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "share revoked"})
}

func handleCreateInviteLink(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	userID := c.MustGet("user_id").(int)

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pack not found"})
		return
	}
	if !database.CanAdmin(perm) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	permission := c.PostForm("permission")
	if permission != "view" && permission != "add" && permission != "edit" && permission != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission"})
		return
	}

	invite, err := database.CreatePackInvite(db, packID, userID, permission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invite"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": invite.Token, "expires_at": invite.ExpiresAt, "permission": invite.Permission})
}

func handleDeleteInviteLink(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	userID := c.MustGet("user_id").(int)

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pack not found"})
		return
	}
	if !database.CanAdmin(perm) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	inviteID, err := strconv.Atoi(c.Param("invite_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invite id"})
		return
	}

	if err := database.DeletePackInvite(db, inviteID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete invite"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invite deleted"})
}

// handleRedeemInvite handles GET /invite/:token — public route, auth required.
func handleRedeemInvite(c *gin.Context) {
	token := c.Param("token")
	db := c.MustGet("db").(*sql.DB)
	userID := c.MustGet("user_id").(int)
	user := c.MustGet("user")

	if err := database.RedeemPackInvite(db, token, userID); err != nil {
		c.HTML(http.StatusBadRequest, "home.html", gin.H{
			"Title": "Invalid Invite - Carryless",
			"User":  user,
			"Error": "This invite link is invalid, expired, or you already have access.",
		})
		return
	}

	c.Redirect(http.StatusFound, "/packs")
}

// handleUserSearch handles GET /api/users/search?q= — JSON endpoint.
func handleUserSearch(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	userID := c.MustGet("user_id").(int)

	q := c.Query("q")
	if len(q) < 2 {
		c.JSON(http.StatusOK, gin.H{"users": []struct{}{}})
		return
	}

	users, err := database.SearchUsers(db, q, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	type userResult struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	results := make([]userResult, len(users))
	for i, u := range users {
		results[i] = userResult{ID: u.ID, Username: u.Username, Email: u.Email}
	}
	c.JSON(http.StatusOK, gin.H{"users": results})
}
