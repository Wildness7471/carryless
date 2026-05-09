package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"carryless/internal/database"
	"carryless/internal/logger"
	"carryless/internal/models"

	"github.com/gin-gonic/gin"
)

// packPermission returns the caller's effective permission on the given pack and
// the pack itself (fetched without user filter). Returns ("none", nil) if not found.
func packPermission(c *gin.Context, packID string) (string, *models.Pack) {
	db := c.MustGet("db").(*sql.DB)
	userID := c.MustGet("user_id").(int)
	pack, err := database.GetPack(db, packID)
	if err != nil {
		return "none", nil
	}
	perm := database.GetUserSharePermission(db, packID, userID)
	pack.UserPermission = perm
	return perm, pack
}

// respond sends HTML or JSON depending on the Accept header.
func respond(c *gin.Context, status int, tmpl string, data gin.H) {
	if c.GetHeader("Accept") == "application/json" {
		c.JSON(status, data)
	} else {
		c.HTML(status, tmpl, data)
	}
}

func forbidden403(c *gin.Context) {
	user, _ := c.Get("user")
	if c.GetHeader("Accept") == "application/json" || c.GetHeader("Authorization") != "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	} else {
		c.HTML(http.StatusForbidden, "403.html", gin.H{
			"Title": "Access Denied - Carryless",
			"User":  user,
		})
	}
}

func handlePacks(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	packs, err := database.GetPacks(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "packs.html", gin.H{
			"Title": "Packs - Carryless",
			"User":  user,
			"Error": "Failed to load packs",
		})
		return
	}

	sharedPacks, err := database.GetPacksSharedWithUser(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "packs.html", gin.H{
			"Title": "Packs - Carryless",
			"User":  user,
			"Error": "Failed to load shared packs",
		})
		return
	}

	// Get user pack labels for the labels bar
	userPackLabels, err := database.GetUserPackLabels(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "packs.html", gin.H{
			"Title": "Packs - Carryless",
			"User":  user,
			"Error": "Failed to load pack labels",
		})
		return
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "packs.html", gin.H{
			"Title": "Packs - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	respond(c, http.StatusOK, "packs.html", gin.H{
		"Title":          "Packs - Carryless",
		"User":           user,
		"Packs":          packs,
		"SharedPacks":    sharedPacks,
		"UserPackLabels": userPackLabels,
		"CSRFToken":      csrfToken.Token,
	})
}

func handleNewPackPage(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "new_pack.html", gin.H{
			"Title": "New Pack - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "new_pack.html", gin.H{
		"Title":     "New Pack - Carryless",
		"User":      user,
		"CSRFToken": csrfToken.Token,
	})
}

func handleCreatePack(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	name := strings.TrimSpace(c.PostForm("name"))
	isPublicStr := c.PostForm("is_public")

	if name == "" {
		c.HTML(http.StatusBadRequest, "new_pack.html", gin.H{
			"Title": "New Pack - Carryless",
			"User":  user,
			"Error": "Pack name is required",
		})
		return
	}

	if len(name) > 200 {
		c.HTML(http.StatusBadRequest, "new_pack.html", gin.H{
			"Title": "New Pack - Carryless",
			"User":  user,
			"Error": "Pack name must be less than 200 characters",
		})
		return
	}

	isPublic := isPublicStr == "true" || isPublicStr == "1"

	_, err := database.CreatePackWithPublic(db, userID, name, isPublic)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "new_pack.html", gin.H{
			"Title": "New Pack - Carryless",
			"User":  user,
			"Error": "Failed to create pack",
		})
		return
	}

	c.Redirect(http.StatusFound, "/packs")
}

func handlePackDetail(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	userID := c.MustGet("user_id").(int)
	user := c.MustGet("user")

	perm, _ := packPermission(c, packID)
	if perm == "none" {
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"Title": "Pack Not Found - Carryless",
			"User":  user,
		})
		return
	}

	pack, err := database.GetPackWithItems(db, packID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "pack_detail.html", gin.H{
			"Title": "Pack Detail - Carryless",
			"User":  user,
			"Error": "Failed to load pack",
		})
		return
	}
	pack.UserPermission = perm

	// Load shares for the manage-shares link (visible to admin/owner)
	if database.CanAdmin(perm) {
		shares, _ := database.GetPackShares(db, packID)
		pack.Shares = shares
	}

	items, err := database.GetItems(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "pack_detail.html", gin.H{
			"Title": "Pack Detail - Carryless",
			"User":  user,
			"Pack":  pack,
			"Error": "Failed to load available items",
		})
		return
	}

	// Get linked items count for each item to mark HasLinkedItems
	itemLinksCount, err := database.GetItemsLinkedCount(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "pack_detail.html", gin.H{
			"Title": "Pack Detail - Carryless",
			"User":  user,
			"Pack":  pack,
			"Error": "Failed to load linked items info",
		})
		return
	}

	// Mark items that have linked items
	for i := range items {
		if count, exists := itemLinksCount[items[i].ID]; exists && count > 0 {
			items[i].HasLinkedItems = true
		}
	}

	categoryWeights := make(map[string]int)
	categoryWornWeights := make(map[string]int)
	labelWeights := make(map[string]int)
	labelColors := make(map[string]string)
	itemsInPack := make(map[int]bool)
	totalWeight := 0
	totalWornWeight := 0
	totalItemCount := 0

	for _, packItem := range pack.Items {
		categoryName := packItem.Item.Category.Name
		itemsInPack[packItem.Item.ID] = true
		packWeight := packItem.Item.WeightGrams * (packItem.Count - packItem.WornCount)
		wornWeight := packItem.Item.WeightGrams * packItem.WornCount
		totalItemCount += packItem.Count
		
		if packWeight > 0 {
			categoryWeights[categoryName] += packWeight
			totalWeight += packWeight
		}
		if wornWeight > 0 {
			categoryWornWeights[categoryName] += wornWeight
			totalWornWeight += wornWeight
		}
		
		// Calculate label weights using the actual label assignment counts
		for _, itemLabel := range packItem.Labels {
			labelWeights[itemLabel.PackLabel.Name] += packItem.Item.WeightGrams * itemLabel.Count
			labelColors[itemLabel.PackLabel.Name] = itemLabel.PackLabel.Color
		}
	}

	// Load sub-items for all items in the pack (one query)
	itemIDs := make([]int, 0, len(pack.Items))
	for _, pi := range pack.Items {
		itemIDs = append(itemIDs, pi.Item.ID)
	}
	subItemMap, _ := database.GetSubItemsForPackBulk(db, packID, itemIDs)
	for i := range pack.Items {
		pack.Items[i].Item.SubItems = subItemMap[pack.Items[i].Item.ID]
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "pack_detail.html", gin.H{
			"Title": "Pack Detail - Carryless",
			"User":  user,
			"Pack":  pack,
			"Error": "Failed to generate security token",
		})
		return
	}

	respond(c, http.StatusOK, "pack_detail.html", gin.H{
		"Title":               "Pack Detail - Carryless",
		"User":                user,
		"Pack":                pack,
		"UserPermission":      perm,
		"CanWrite":            database.CanWrite(perm),
		"CanEdit":             database.CanEdit(perm),
		"CanAdmin":            database.CanAdmin(perm),
		"IsOwner":             perm == "owner",
		"CurrentUserID":       userID,
		"Items":               items,
		"ItemsInPack":         itemsInPack,
		"CategoryWeights":     categoryWeights,
		"CategoryWornWeights": categoryWornWeights,
		"LabelWeights":        labelWeights,
		"LabelColors":         labelColors,
		"TotalWeight":         totalWeight,
		"TotalWornWeight":     totalWornWeight,
		"TotalItemCount":      totalItemCount,
		"CSRFToken":           csrfToken.Token,
	})
}

func handlePublicPack(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	
	user, _ := c.Get("user")

	pack, err := database.GetPackWithItems(db, packID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.HTML(http.StatusNotFound, "404.html", gin.H{
				"Title": "Pack Not Found - Carryless",
				"User":  user,
			})
			return
		}
		c.HTML(http.StatusInternalServerError, "public_pack.html", gin.H{
			"Title": "Pack Detail - Carryless",
			"User":  user,
			"Error": "Failed to load pack",
		})
		return
	}

	if !pack.IsPublic {
		c.HTML(http.StatusForbidden, "403.html", gin.H{
			"Title": "Access Denied - Carryless",
			"User":  user,
		})
		return
	}

	categoryWeights := make(map[string]int)
	categoryWornWeights := make(map[string]int)
	labelWeights := make(map[string]int)
	labelColors := make(map[string]string)
	totalWeight := 0
	totalWornWeight := 0
	totalItemCount := 0

	for _, packItem := range pack.Items {
		categoryName := packItem.Item.Category.Name
		packWeight := packItem.Item.WeightGrams * (packItem.Count - packItem.WornCount)
		wornWeight := packItem.Item.WeightGrams * packItem.WornCount
		totalItemCount += packItem.Count
		
		if packWeight > 0 {
			categoryWeights[categoryName] += packWeight
			totalWeight += packWeight
		}
		if wornWeight > 0 {
			categoryWornWeights[categoryName] += wornWeight
			totalWornWeight += wornWeight
		}
		
		// Calculate label weights using the actual label assignment counts
		for _, itemLabel := range packItem.Labels {
			labelWeights[itemLabel.PackLabel.Name] += packItem.Item.WeightGrams * itemLabel.Count
			labelColors[itemLabel.PackLabel.Name] = itemLabel.PackLabel.Color
		}
	}

	var csrfToken string
	if userID, hasUserID := c.Get("user_id"); hasUserID {
		if token, err := database.CreateCSRFToken(db, userID.(int)); err == nil {
			csrfToken = token.Token
		}
	}

	c.HTML(http.StatusOK, "public_pack.html", gin.H{
		"Title":               pack.Name + " - Carryless",
		"User":                user,
		"Pack":                pack,
		"CategoryWeights":     categoryWeights,
		"CategoryWornWeights": categoryWornWeights,
		"LabelWeights":        labelWeights,
		"LabelColors":         labelColors,
		"TotalWeight":         totalWeight,
		"TotalWornWeight":     totalWornWeight,
		"TotalItemCount":      totalItemCount,
		"CSRFToken":           csrfToken,
	})
}

func handlePublicPackByShortID(c *gin.Context) {
	shortID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	
	user, _ := c.Get("user")

	pack, err := database.GetPackByShortID(db, shortID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.HTML(http.StatusNotFound, "404.html", gin.H{
				"Title": "Pack Not Found - Carryless",
				"User":  user,
			})
			return
		}
		c.HTML(http.StatusInternalServerError, "public_pack.html", gin.H{
			"Title": "Pack Detail - Carryless",
			"User":  user,
			"Error": "Failed to load pack",
		})
		return
	}

	if !pack.IsPublic {
		c.HTML(http.StatusForbidden, "403.html", gin.H{
			"Title": "Access Denied - Carryless",
			"User":  user,
		})
		return
	}

	// Get pack items
	packWithItems, err := database.GetPackWithItems(db, pack.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public_pack.html", gin.H{
			"Title": "Pack Detail - Carryless",
			"User":  user,
			"Error": "Failed to load pack items",
		})
		return
	}

	categoryWeights := make(map[string]int)
	categoryWornWeights := make(map[string]int)
	labelWeights := make(map[string]int)
	labelColors := make(map[string]string)
	totalWeight := 0
	totalWornWeight := 0
	totalItemCount := 0

	for _, packItem := range packWithItems.Items {
		categoryName := packItem.Item.Category.Name
		packWeight := packItem.Item.WeightGrams * (packItem.Count - packItem.WornCount)
		wornWeight := packItem.Item.WeightGrams * packItem.WornCount
		totalItemCount += packItem.Count
		
		if packWeight > 0 {
			categoryWeights[categoryName] += packWeight
			totalWeight += packWeight
		}
		if wornWeight > 0 {
			categoryWornWeights[categoryName] += wornWeight
			totalWornWeight += wornWeight
		}
		
		// Calculate label weights using the actual label assignment counts
		for _, itemLabel := range packItem.Labels {
			labelWeights[itemLabel.PackLabel.Name] += packItem.Item.WeightGrams * itemLabel.Count
			labelColors[itemLabel.PackLabel.Name] = itemLabel.PackLabel.Color
		}
	}

	var csrfToken string
	if userID, hasUserID := c.Get("user_id"); hasUserID {
		if token, err := database.CreateCSRFToken(db, userID.(int)); err == nil {
			csrfToken = token.Token
		}
	}

	c.HTML(http.StatusOK, "public_pack.html", gin.H{
		"Title":               packWithItems.Name + " - Carryless",
		"User":                user,
		"Pack":                packWithItems,
		"CategoryWeights":     categoryWeights,
		"CategoryWornWeights": categoryWornWeights,
		"LabelWeights":        labelWeights,
		"LabelColors":         labelColors,
		"TotalWeight":         totalWeight,
		"TotalWornWeight":     totalWornWeight,
		"TotalItemCount":      totalItemCount,
		"CSRFToken":           csrfToken,
	})
}

func handleEditPackPage(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")
	packID := c.Param("id")

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.HTML(http.StatusNotFound, "edit_pack.html", gin.H{
			"Title": "Edit Pack - Carryless",
			"User":  user,
			"Error": "Pack not found",
		})
		return
	}
	if !database.CanAdmin(perm) {
		forbidden403(c)
		return
	}
	_ = userID

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "edit_pack.html", gin.H{
			"Title": "Edit Pack - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "edit_pack.html", gin.H{
		"Title":     "Edit Pack - Carryless",
		"User":      user,
		"Pack":      pack,
		"CSRFToken": csrfToken.Token,
	})
}

func handleUpdatePack(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")
	packID := c.Param("id")

	name := strings.TrimSpace(c.PostForm("name"))
	isPublicStr := c.PostForm("is_public")

	if name == "" {
		pack, _ := database.GetPack(db, packID)
		c.HTML(http.StatusBadRequest, "edit_pack.html", gin.H{
			"Title": "Edit Pack - Carryless",
			"User":  user,
			"Pack":  pack,
			"Error": "Pack name is required",
		})
		return
	}

	if len(name) > 200 {
		pack, _ := database.GetPack(db, packID)
		c.HTML(http.StatusBadRequest, "edit_pack.html", gin.H{
			"Title": "Edit Pack - Carryless",
			"User":  user,
			"Pack":  pack,
			"Error": "Pack name must be less than 200 characters",
		})
		return
	}

	isPublic := isPublicStr == "true" || isPublicStr == "1"

	err := database.UpdatePack(db, userID, packID, name, isPublic)
	if err != nil {
		var errorMsg string
		if strings.Contains(err.Error(), "not found") {
			errorMsg = "Pack not found"
		} else {
			errorMsg = "Failed to update pack"
		}
		
		pack, _ := database.GetPack(db, packID)
		c.HTML(http.StatusBadRequest, "edit_pack.html", gin.H{
			"Title": "Edit Pack - Carryless",
			"User":  user,
			"Pack":  pack,
			"Error": errorMsg,
		})
		return
	}

	c.Redirect(http.StatusFound, "/packs")
}

func handleDeletePack(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	err := database.DeletePack(db, userID, packID)
	if err != nil {
		c.Redirect(http.StatusFound, "/packs")
		return
	}

	c.Redirect(http.StatusFound, "/packs")
}

func handleUpdatePackNote(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pack not found"})
		return
	}
	if !database.CanEdit(perm) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permission"})
		return
	}

	note := strings.TrimSpace(c.PostForm("note"))

	// Validate note length (500 character limit)
	if len(note) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Note must be less than 500 characters"})
		return
	}

	err := database.UpdatePackNote(db, pack.UserID, packID, note)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pack not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pack note"})
		return
	}

	response := gin.H{"message": "Pack note updated successfully"}

	// Include new CSRF token if available (set by CSRFWithRenewal middleware)
	if newToken, exists := c.Get("new_csrf_token"); exists {
		response["csrf_token"] = newToken
	}

	c.JSON(http.StatusOK, response)
}

func handleAddItemToPack(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pack not found"})
		return
	}
	if !database.CanWrite(perm) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permission"})
		return
	}

	itemIDStr := c.PostForm("item_id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var addErr error
	if perm == "owner" {
		addErr = database.AddItemToPack(db, packID, itemID, userID)
	} else {
		addErr = database.AddItemToPackAsSharedUser(db, packID, itemID, userID)
	}
	if addErr != nil {
		if strings.Contains(addErr.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pack or item not found"})
			return
		}
		if strings.Contains(addErr.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Item already in pack"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add item to pack"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item added to pack successfully"})
}

func handleRemoveItemFromPack(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pack not found"})
		return
	}
	if !database.CanWrite(perm) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permission"})
		return
	}

	itemIDStr := c.Param("item_id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	err = database.RemoveItemFromPack(db, packID, itemID, pack.UserID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pack or item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove item from pack"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item removed from pack successfully"})
}

func handleToggleWorn(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pack not found"})
		return
	}
	if !database.CanWrite(perm) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permission"})
		return
	}

	itemIDStr := c.Param("item_id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	isWornStr := c.PostForm("is_worn")
	isWorn := isWornStr == "true" || isWornStr == "1"

	err = database.TogglePackItemWorn(db, packID, itemID, pack.UserID, isWorn)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pack or item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update worn status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Worn status updated successfully"})
}

func handleUpdateWornCount(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	perm, pack := packPermission(c, packID)
	if pack == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pack not found"})
		return
	}
	if !database.CanWrite(perm) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permission"})
		return
	}

	itemIDStr := c.Param("item_id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	wornCountStr := c.PostForm("worn_count")
	wornCount, err := strconv.Atoi(wornCountStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid worn count"})
		return
	}

	err = database.UpdatePackItemWornCount(db, packID, itemID, pack.UserID, wornCount)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pack or item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update worn count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Worn count updated successfully"})
}

func handleDuplicatePack(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	logger.Debug("Duplicate pack request",
		"user_id", userID,
		"pack_id", packID,
		"ip", c.ClientIP())

	newPack, err := database.DuplicatePack(db, userID, packID)
	if err != nil {
		logger.Error("Duplicate pack failed",
			"user_id", userID,
			"pack_id", packID,
			"error", err)
		if strings.Contains(err.Error(), "not found") {
			logger.Debug("Pack not found - redirecting to packs page")
			c.Redirect(http.StatusFound, "/packs")
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			logger.Warn("Unauthorized access attempt",
				"user_id", userID,
				"pack_id", packID)
			c.Redirect(http.StatusFound, "/packs")
			return
		}
		logger.Error("Unknown error during duplication",
			"user_id", userID,
			"pack_id", packID,
			"error", err)
		c.Redirect(http.StatusFound, "/packs")
		return
	}

	logger.Info("Pack duplication successful",
		"user_id", userID,
		"original_pack_id", packID,
		"new_pack_id", newPack.ID,
		"new_pack_name", newPack.Name)
	c.Redirect(http.StatusFound, "/packs")
}

func handleCreatePackLabel(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	name := c.PostForm("name")
	color := c.PostForm("color")
	if color == "" {
		color = "#6b7280" // Default gray color
	}

	if strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Label name is required"})
		return
	}

	_, err := database.CreatePackLabel(db, packID, strings.TrimSpace(name), color, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Label name already exists"})
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label created successfully"})
}

func handleUpdatePackLabel(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	
	labelID, err := strconv.Atoi(c.Param("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	name := c.PostForm("name")
	color := c.PostForm("color")
	if color == "" {
		color = "#6b7280" // Default gray color
	}

	if strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Label name is required"})
		return
	}

	err = database.UpdatePackLabel(db, labelID, strings.TrimSpace(name), color, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Label name already exists"})
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label updated successfully"})
}

func handleDeletePackLabel(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	
	labelID, err := strconv.Atoi(c.Param("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	err = database.DeletePackLabel(db, labelID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label deleted successfully"})
}

func handleAssignLabelToItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	
	packItemID, err := strconv.Atoi(c.Param("item_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	labelID, err := strconv.Atoi(c.PostForm("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	err = database.AssignLabelToPackItem(db, packItemID, labelID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Item or label not found"})
			return
		}
		if strings.Contains(err.Error(), "same pack") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Label does not belong to the same pack"})
			return
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Label already assigned to this item"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label assigned successfully"})
}

func handleRemoveLabelFromItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	
	packItemID, err := strconv.Atoi(c.Param("item_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	labelID, err := strconv.Atoi(c.Param("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	err = database.RemoveLabelFromPackItem(db, packItemID, labelID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label assignment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label removed successfully"})
}

func handlePackChecklist(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)

	user, _ := c.Get("user")
	userIDVal, hasUserID := c.Get("user_id")

	pack, err := database.GetPackWithItems(db, packID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.HTML(http.StatusNotFound, "404.html", gin.H{"Title": "Pack Not Found - Carryless", "User": user})
			return
		}
		c.HTML(http.StatusInternalServerError, "checklist.html", gin.H{
			"Title": "Pack Checklist - Carryless", "User": user, "Error": "Failed to load pack",
		})
		return
	}

	// Access: public packs are open; private packs require view+ share permission
	if !pack.IsPublic {
		if !hasUserID {
			c.HTML(http.StatusForbidden, "403.html", gin.H{"Title": "Access Denied - Carryless", "User": user})
			return
		}
		uid := userIDVal.(int)
		perm := database.GetUserSharePermission(db, packID, uid)
		if perm == "none" {
			c.HTML(http.StatusForbidden, "403.html", gin.H{"Title": "Access Denied - Carryless", "User": user})
			return
		}
	}

	// Load sub-items for checklist display
	itemIDs := make([]int, 0, len(pack.Items))
	for _, pi := range pack.Items {
		itemIDs = append(itemIDs, pi.Item.ID)
	}
	subItemMap, _ := database.GetSubItemsForPackBulk(db, packID, itemIDs)
	for i := range pack.Items {
		pack.Items[i].Item.SubItems = subItemMap[pack.Items[i].Item.ID]
	}

	totalItems := 0
	for _, packItem := range pack.Items {
		totalItems += packItem.Count
	}

	isOwner := hasUserID && pack.UserID == userIDVal.(int)

	var csrfTok string
	if hasUserID {
		if tok, err := database.CreateCSRFToken(db, userIDVal.(int)); err == nil {
			csrfTok = tok.Token
		}
	}

	c.HTML(http.StatusOK, "checklist.html", gin.H{
		"Title":      pack.Name + " Checklist - Carryless",
		"User":       user,
		"Pack":       pack,
		"TotalItems": totalItems,
		"IsOwner":    isOwner,
		"CSRFToken":  csrfTok,
	})
}

func handlePackChecklistByShortID(c *gin.Context) {
	shortID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	
	user, _ := c.Get("user")
	userID, hasUserID := c.Get("user_id")

	pack, err := database.GetPackByShortID(db, shortID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.HTML(http.StatusNotFound, "404.html", gin.H{
				"Title": "Pack Not Found - Carryless",
				"User":  user,
			})
			return
		}
		c.HTML(http.StatusInternalServerError, "checklist.html", gin.H{
			"Title": "Pack Checklist - Carryless",
			"User":  user,
			"Error": "Failed to load pack",
		})
		return
	}

	if !pack.IsPublic {
		c.HTML(http.StatusForbidden, "403.html", gin.H{
			"Title": "Access Denied - Carryless",
			"User":  user,
		})
		return
	}

	// Get pack items using the full UUID
	packWithItems, err := database.GetPackWithItems(db, pack.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "checklist.html", gin.H{
			"Title": "Pack Checklist - Carryless",
			"User":  user,
			"Error": "Failed to load pack items",
		})
		return
	}

	// Load sub-items for checklist display
	itemIDs := make([]int, 0, len(packWithItems.Items))
	for _, pi := range packWithItems.Items {
		itemIDs = append(itemIDs, pi.Item.ID)
	}
	subItemMap, _ := database.GetSubItemsForPackBulk(db, pack.ID, itemIDs)
	for i := range packWithItems.Items {
		packWithItems.Items[i].Item.SubItems = subItemMap[packWithItems.Items[i].Item.ID]
	}

	totalItems := 0
	for _, packItem := range packWithItems.Items {
		totalItems += packItem.Count
	}

	isOwner := hasUserID && packWithItems.UserID == userID.(int)

	var csrfTok string
	if hasUserID {
		if tok, err := database.CreateCSRFToken(db, userID.(int)); err == nil {
			csrfTok = tok.Token
		}
	}

	c.HTML(http.StatusOK, "checklist.html", gin.H{
		"Title":      packWithItems.Name + " Checklist - Carryless",
		"User":       user,
		"Pack":       packWithItems,
		"TotalItems": totalItems,
		"IsOwner":    isOwner,
		"CSRFToken":  csrfTok,
	})
}

func handleTogglePackLock(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	isLockedStr := c.PostForm("is_locked")
	isLocked := isLockedStr == "true" || isLockedStr == "1"

	err := database.TogglePackLock(db, userID, packID, isLocked)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pack not found or unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update archive status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Archive status updated successfully"})
}

// User Pack Labels handlers (pack-level labels shared across user's packs)

func handleCreateUserPackLabel(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	name := c.PostForm("name")
	color := c.PostForm("color")
	if color == "" {
		color = "#6b7280" // Default gray color
	}

	if strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Label name is required"})
		return
	}

	label, err := database.CreateUserPackLabel(db, userID, strings.TrimSpace(name), color)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Label name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label created successfully", "label": label})
}

func handleUpdateUserPackLabel(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	labelID, err := strconv.Atoi(c.Param("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	name := c.PostForm("name")
	color := c.PostForm("color")
	if color == "" {
		color = "#6b7280" // Default gray color
	}

	if strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Label name is required"})
		return
	}

	err = database.UpdateUserPackLabel(db, labelID, strings.TrimSpace(name), color, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Label name already exists"})
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label updated successfully"})
}

func handleDeleteUserPackLabel(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	labelID, err := strconv.Atoi(c.Param("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	err = database.DeleteUserPackLabel(db, labelID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label deleted successfully"})
}

func handleAssignPackLevelLabel(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	labelID, err := strconv.Atoi(c.PostForm("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	err = database.AssignLabelToPack(db, packID, labelID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pack or label not found"})
			return
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Label already assigned to this pack"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label assigned successfully"})
}

func handleRemovePackLevelLabel(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	labelID, err := strconv.Atoi(c.Param("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	err = database.RemoveLabelFromPack(db, packID, labelID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label assignment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label removed successfully"})
}
// handlePackCompare handles GET /packs/compare?packs=id1,id2,...
// The user must have at least "view" permission on each pack.
func handlePackCompare(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	rawIDs := c.Query("packs")
	if rawIDs == "" {
		c.Redirect(http.StatusFound, "/packs")
		return
	}

	ids := strings.Split(rawIDs, ",")
	if len(ids) < 2 || len(ids) > 6 {
		respond(c, http.StatusBadRequest, "pack_compare.html", gin.H{
			"Title": "Compare Packs - Carryless",
			"User":  user,
			"Error": "Select between 2 and 6 packs to compare",
		})
		return
	}

	type comparePack struct {
		Pack            *models.Pack
		TotalWeight     int
		TotalWornWeight int
		TotalItemCount  int
		CategoryWeights map[string]int
	}

	packs := make([]comparePack, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		perm := database.GetUserSharePermission(db, id, userID)
		if perm == "none" {
			respond(c, http.StatusForbidden, "pack_compare.html", gin.H{
				"Title": "Compare Packs - Carryless",
				"User":  user,
				"Error": "You do not have access to one or more of the selected packs",
			})
			return
		}
		pack, err := database.GetPackWithItems(db, id)
		if err != nil {
			respond(c, http.StatusNotFound, "pack_compare.html", gin.H{
				"Title": "Compare Packs - Carryless",
				"User":  user,
				"Error": "One or more packs could not be found",
			})
			return
		}
		pack.UserPermission = perm

		totalWeight := 0
		totalWornWeight := 0
		totalItemCount := 0
		categoryWeights := make(map[string]int)

		for _, pi := range pack.Items {
			packWeight := pi.Item.WeightGrams * (pi.Count - pi.WornCount)
			wornWeight := pi.Item.WeightGrams * pi.WornCount
			totalWeight += packWeight
			totalWornWeight += wornWeight
			totalItemCount += pi.Count
			if pi.Item.Category != nil {
				categoryWeights[pi.Item.Category.Name] += packWeight + wornWeight
			}
		}

		packs = append(packs, comparePack{
			Pack:            pack,
			TotalWeight:     totalWeight,
			TotalWornWeight: totalWornWeight,
			TotalItemCount:  totalItemCount,
			CategoryWeights: categoryWeights,
		})
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "pack_compare.html", gin.H{
			"Title": "Compare Packs - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	respond(c, http.StatusOK, "pack_compare.html", gin.H{
		"Title":     "Compare Packs - Carryless",
		"User":      user,
		"Packs":     packs,
		"CSRFToken": csrfToken.Token,
		"PackIDs":   rawIDs,
	})
}
