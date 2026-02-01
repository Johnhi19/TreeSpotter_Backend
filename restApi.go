package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Johnhi19/TreeSpotter_backend/db"
	"github.com/Johnhi19/TreeSpotter_backend/handlers"
	"github.com/Johnhi19/TreeSpotter_backend/middleware"

	"github.com/Johnhi19/TreeSpotter_backend/models"
	"github.com/gin-gonic/gin"
)

func main() {
	db.Connect()
	defer db.Disconnect()

	router := gin.Default()

	// Serve images statically
	router.Static("/uploads", "./uploads")

	// Public (no auth)
	public := router.Group("/")
	{
		public.POST("/login", handlers.Login)
		public.POST("/register", handlers.Register)
	}

	// Protected (requires JWT)
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.DELETE("/trees/:id", removeTree)
		protected.DELETE("/meadows/:id", removeMeadow)
		protected.DELETE("/trees/images/:imageId", removeTreeImage)

		protected.GET("/meadows/:id", findMeadowByID)
		protected.GET("/meadows", getBasicInfoOfAllMeadows)
		protected.GET("/meadows/:id/trees", getTreesOfMeadow)
		protected.GET("/trees/:id", findTreeByID)
		protected.GET("/trees/:id/images", getTreeImages)

		protected.POST("/meadows", insertMeadow)
		protected.POST("/trees", insertTree)
		protected.POST("trees/:id/uploadImage", uploadImage)

		protected.PUT("/meadows/:id", updateMeadow)
		protected.PUT("/trees/:id", updateTree)
		protected.PUT("/trees/images/:imageId", updateTreeImage)
	}

	go func() {
		if err := router.Run(":8080"); err != nil {
			panic(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	db.Disconnect()
}

func findMeadowByID(c *gin.Context) {
	userID := c.GetInt("user_id")

	meadowId := c.Param("id")

	intMeadowID, err := strconv.Atoi(meadowId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	meadow := db.FindOneMeadowByIdForUser(intMeadowID, userID)
	c.IndentedJSON(http.StatusOK, meadow)
}

func findTreeByID(c *gin.Context) {
	userID := c.GetInt("user_id")

	treeId := c.Param("id")

	intTreeID, err := strconv.Atoi(treeId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	tree := db.FindOneTreeById(intTreeID, userID)
	c.IndentedJSON(http.StatusOK, tree)
}

func getBasicInfoOfAllMeadows(c *gin.Context) {
	userID := c.GetInt("user_id")

	meadows := db.FindAllMeadowsForUser(userID)
	c.IndentedJSON(http.StatusOK, meadows)
}

func getTreesOfMeadow(c *gin.Context) {
	userID := c.GetInt("user_id")

	meadowId := c.Param("id")

	intMeadowID, err := strconv.Atoi(meadowId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	trees := db.FindAllTreesForMeadow(intMeadowID, userID)
	c.IndentedJSON(http.StatusOK, trees)
}

func insertMeadow(c *gin.Context) {
	var meadow models.Meadow

	userID := c.GetInt("user_id")

	if err := c.ShouldBindJSON(&meadow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	insertedID := db.InsertOneMeadowForUser(meadow, userID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Meadow inserted successfully",
		"id":      insertedID,
	})
}

func insertTree(c *gin.Context) {
	var tree models.Tree

	userID := c.GetInt("user_id")

	if err := c.ShouldBindJSON(&tree); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Insert the tree
	insertedID := db.InsertOneTreeForUser(tree, userID)

	// Update the meadow's TreeIds list by adding the tree ID
	if err := db.UpdateMeadowTreeIdsForUser(tree.MeadowId, insertedID, false, userID); err != nil {
		fmt.Printf("ERROR executing UPDATE: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tree inserted but failed to update meadow"})
		return
	}

	fmt.Printf("Updated Meadow %d with new Tree ID %d\n", tree.MeadowId, insertedID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tree inserted successfully",
		"id":      insertedID,
	})
}

func removeMeadow(c *gin.Context) {
	userID := c.GetInt("user_id")

	// Get meadow ID from URL parameter
	meadowId := c.Param("id")
	intMeadowID, err := strconv.Atoi(meadowId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	fmt.Printf("Attempting to delete meadow with ID: %d\n", intMeadowID)

	// Delete the meadow (which also updates the trees)
	if err := db.DeleteOneMeadowForUser(intMeadowID, userID); err != nil {
		fmt.Printf("ERROR deleting meadow: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("Meadow %d deleted successfully\n", intMeadowID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Meadow deleted successfully",
		"id":      intMeadowID,
	})
}

func removeTree(c *gin.Context) {
	userID := c.GetInt("user_id")

	// Get tree ID from URL parameter
	id := c.Param("id")
	intID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	fmt.Printf("Attempting to delete tree with ID: %d\n", intID)

	// Delete the tree (which also updates the meadow)
	if err := db.DeleteOneTreeForUser(intID, userID); err != nil {
		fmt.Printf("ERROR deleting tree: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("Tree %d deleted successfully\n", intID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Tree deleted successfully",
		"id":      intID,
	})
}

func removeTreeImage(c *gin.Context) {
	userID := c.GetInt("user_id")

	// Get tree ID from URL parameter
	imageId := c.Param("imageId")
	intID, err := strconv.Atoi(imageId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Image ID format"})
		return
	}

	fmt.Printf("Attempting to delete image with ID: %d\n", intID)

	// Delete the image
	if err := db.DeleteTreeImage(intID, userID); err != nil {
		fmt.Printf("ERROR deleting image: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("Image %d deleted successfully\n", intID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Image deleted successfully",
		"imageId": intID,
	})
}

func updateMeadow(c *gin.Context) {
	userID := c.GetInt("user_id")

	var meadow models.Meadow

	// Bind the JSON body to the meadow struct
	if err := c.ShouldBindJSON(&meadow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update the meadow
	db.UpdateMeadowForUser(meadow, userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Meadow updated successfully",
	})
}

func updateTree(c *gin.Context) {
	userID := c.GetInt("user_id")

	var tree models.Tree

	// Bind the JSON body to the tree struct
	if err := c.ShouldBindJSON(&tree); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update the tree
	db.UpdateTreeForUser(tree, userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Tree updated successfully",
	})
}

func updateTreeImage(c *gin.Context) {
	userID := c.GetInt("user_id")

	imageId := c.Param("imageId")
	intImageID, err := strconv.Atoi(imageId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	newDescription := c.PostForm("newDescription")
	newDatetime := c.PostForm("newDatetime")

	if newDescription != "null" {
		db.UpdateTreeImageDescriptionDb(intImageID, newDescription, userID)
	} else if newDatetime != "null" {
		parsedTime, err := time.Parse(time.RFC3339, newDatetime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid datetime format"})
			return
		}
		db.UpdateTreeImageDatetimeDb(intImageID, parsedTime, userID)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid fields to update"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Image updated successfully",
	})

}

func getTreeImages(c *gin.Context) {
	userID := c.GetInt("user_id")

	treeId := c.Param("id")

	intTreeID, err := strconv.Atoi(treeId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	images := db.GetTreeImageDb(intTreeID, userID)

	fmt.Printf("Successfully retrieved %d images for user %d and tree %d\n", len(images), userID, intTreeID)

	c.JSON(http.StatusOK, images)

}

func uploadImage(c *gin.Context) {
	userID := c.GetInt("user_id")

	treeId := c.Param("id")

	intTreeID, err := strconv.Atoi(treeId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	description := c.PostForm("description")

	file := handlers.UploadImageHandler(c.Writer, c.Request)
	if file == nil {
		return
	}

	// Optionally, you can store the image info in the database
	err = db.UploadImageDb(file.Name(), description, userID, intTreeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image info to database"})
		return
	}

	fmt.Printf("User %d uploaded file: %s\n", userID, file.Name())

	c.JSON(http.StatusOK, gin.H{
		"message": "Image uploaded successfully",
		"path":    file.Name(),
	})
}
