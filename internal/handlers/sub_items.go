package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"carryless/internal/database"

	"github.com/gin-gonic/gin"
)

// POST /api/items/:id/sub-items
func handleCreateSubItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	itemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	// Verify ownership
	if _, err := database.GetItem(db, userID, itemID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}

	sub, err := database.CreateSubItem(db, itemID, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create sub-item"})
		return
	}
	c.JSON(http.StatusOK, sub)
}

// PUT /api/items/:id/sub-items/:sub_id
func handleUpdateSubItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	subID, err := strconv.Atoi(c.Param("sub_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sub-item id"})
		return
	}
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	if err := database.UpdateSubItem(db, subID, userID, name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "sub-item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update sub-item"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// DELETE /api/items/:id/sub-items/:sub_id
func handleDeleteSubItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	subID, err := strconv.Atoi(c.Param("sub_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sub-item id"})
		return
	}

	if err := database.DeleteSubItem(db, subID, userID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "sub-item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete sub-item"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// POST /packs/:id/items/:item_id/sub-items/:sub_id/toggle
func handleToggleSubItemCheck(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)

	subID, err := strconv.Atoi(c.Param("sub_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sub-item id"})
		return
	}

	// Verify the caller has at least view permission on the pack
	perm, pack := packPermission(c, packID)
	if pack == nil || perm == "none" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	isChecked := c.PostForm("is_checked") == "true" || c.PostForm("is_checked") == "1"
	if err := database.ToggleSubItemCheck(db, packID, subID, isChecked); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to toggle sub-item"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_checked": isChecked})
}
