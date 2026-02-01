package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Johnhi19/TreeSpotter_backend/models"
	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Connect() {

	// get connection properties for the mysql db
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	var err error

	// Try to connect multiple times with delays
	for i := 1; i <= 30; i++ {
		DB, err = sql.Open("mysql", dsn)
		if err == nil {
			err = DB.Ping()
		}

		if err == nil {
			log.Println("Connected to MySQL!")
			return
		}

		log.Printf("Waiting for DB... (%d/30): %v\n", i, err)
		time.Sleep(2 * time.Second)
	}
}

func Disconnect() {
	if err := DB.Close(); err != nil {
		log.Fatal("Error closing database connection:", err)
	} else {
		fmt.Println("Database connection closed.")
	}
}

func DeleteOneMeadowForUser(meadowId int, userID int) error {
	// First, get all tree IDs associated with the meadow
	meadow := FindOneMeadowByIdForUser(meadowId, userID)
	if meadow.ID == 0 {
		return fmt.Errorf("meadow with ID %d not found", meadowId)
	}

	// Delete all associated trees
	for _, treeId := range meadow.TreeIds {
		if err := deleteTreeOnly(treeId, userID); err != nil {
			fmt.Printf("Warning: Failed to delete tree ID %d: %v\n", treeId, err)
		}
	}

	// Now delete the meadow itself
	result, err := DB.Exec("DELETE FROM meadows WHERE ID = ? and user_id = ?", meadowId, userID)
	if err != nil {
		return fmt.Errorf("failed to delete meadow: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no meadow found with ID %d and user ID %d", meadowId, userID)
	}

	fmt.Printf("Deleted meadow for user %d with ID: %d\n", userID, meadowId)
	return nil
}

// Deletes the tree and updates the meadow's TreeIds accordingly
func DeleteOneTreeForUser(treeId int, userID int) error {
	// First, get the tree to know which meadow it belongs to
	tree := FindOneTreeById(treeId, userID)
	if tree.ID == 0 {
		return fmt.Errorf("tree with ID %d not found", treeId)
	}

	meadowId := tree.MeadowId

	// Delete the tree from the database
	if err := deleteTreeOnly(treeId, userID); err != nil {
		return err
	}

	// Remove tree ID from meadow's TreeIds
	if err := UpdateMeadowTreeIdsForUser(meadowId, int64(treeId), true, userID); err != nil {
		fmt.Printf("Warning: Tree deleted but failed to update meadow: %v\n", err)
		return err
	}

	return nil
}

func DeleteTreeImage(imageID int, userID int) error {
	var filePath string
	err := DB.QueryRow("SELECT path FROM images WHERE id = ? AND user_id = ?", imageID, userID).Scan(&filePath)
	if err != nil {
		return fmt.Errorf("failed to retrieve image path: %w", err)
	}

	// Delete the image file from the filesystem
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	fmt.Printf("Successfully deleted file: %s\n", filePath)

	result, err := DB.Exec("DELETE FROM images WHERE id = ? AND user_id = ?", imageID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no image found with ID %d and user ID %d", imageID, userID)
	}

	fmt.Printf("Deleted image for user %d with ID: %d\n", userID, imageID)
	return nil
}

func FindAllMeadowsForUser(userID int) []models.Meadow {
	var meadows []models.Meadow

	rows, err := DB.Query("SELECT ID, Location, Name, Size, TreeIds FROM meadows WHERE user_id = ?", userID)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var med models.Meadow
		if err := rows.Scan(&med.ID, &med.Location, &med.Name, &med.Size, &med.TreeIds); err != nil {
			panic(err)
		}
		meadows = append(meadows, med)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	return meadows
}

func FindAllTreesForMeadow(meadowId int, userID int) []models.Tree {
	var trees []models.Tree

	meadow := FindOneMeadowByIdForUser(meadowId, userID)

	if len(meadow.TreeIds) == 0 {
		return []models.Tree{}
	}

	placeholders := make([]string, len(meadow.TreeIds))
	args := make([]any, len(meadow.TreeIds))
	for i, treeId := range meadow.TreeIds {
		placeholders[i] = "?"
		args[i] = treeId
	}

	query := fmt.Sprintf("SELECT ID, PlantDate, MeadowId, Position, Type FROM trees WHERE ID IN (%s) AND user_id = ?",
		strings.Join(placeholders, ","))

	args = append(args, userID)

	rows, err := DB.Query(query, args...)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var tree models.Tree
		if err := rows.Scan(&tree.ID, &tree.PlantDate, &tree.MeadowId, &tree.Position, &tree.Type); err != nil {
			panic(err)
		}
		trees = append(trees, tree)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	return trees
}

func FindOneMeadowByIdForUser(meadowId int, userID int) models.Meadow {
	var meadow models.Meadow

	row := DB.QueryRow("SELECT ID, Location, Name, Size, TreeIds FROM meadows WHERE ID = ? AND user_id = ?", meadowId, userID)
	if err := row.Scan(&meadow.ID, &meadow.Location, &meadow.Name, &meadow.Size, &meadow.TreeIds); err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No meadow found with ID:", meadowId)
			return meadow
		}
		panic(err)
	}
	fmt.Println("Found meadow:", meadow)
	return meadow
}

func FindOneTreeById(treeId int, userID int) models.Tree {
	var tree models.Tree

	row := DB.QueryRow("SELECT ID, PlantDate, MeadowId, Position, Type FROM trees WHERE ID = ? AND user_id = ?", treeId, userID)
	if err := row.Scan(&tree.ID, &tree.PlantDate, &tree.MeadowId, &tree.Position, &tree.Type); err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("No tree found with ID: %d and user ID: %d\n", treeId, userID)
			return tree
		}
		fmt.Printf("Type of tree.PlantDate: %T\n", tree.PlantDate)
		panic(err)
	}
	fmt.Println("Found tree:", tree)
	return tree
}

func GetTreeImageDb(treeID int, userID int) []models.Image {
	var images []models.Image

	rows, err := DB.Query("SELECT id, path, description, datetime FROM images WHERE tree_id = ? AND user_id = ?", treeID, userID)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var img models.Image
		if err := rows.Scan(&img.ID, &img.Path, &img.Description, &img.Datetime); err != nil {
			panic(err)
		}
		// update path to include leading slash
		img.Path = fmt.Sprintf("/%s", img.Path)
		images = append(images, img)
	}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	return images
}

func InsertOneMeadowForUser(meadow models.Meadow, userID int) any {
	result, err := DB.Exec("INSERT INTO meadows (Location, Name, Size, TreeIds, user_id) VALUES (?, ?, ?, ?, ?)",
		meadow.Location, meadow.Name, meadow.Size, meadow.TreeIds, userID)
	if err != nil {
		panic(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Inserted a meadow for the user %d with ID: %d\n", userID, id)
	return id
}

func InsertOneTreeForUser(tree models.Tree, userID int) int64 {
	result, err := DB.Exec("INSERT INTO trees (PlantDate, MeadowId, Position, Type, user_id) VALUES (?, ?, ?, ?, ?)",
		tree.PlantDate, tree.MeadowId, tree.Position, tree.Type, userID)
	if err != nil {
		panic(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Inserted a tree for the user %d with ID: %d\n", userID, id)
	return id
}

func UpdateMeadowTreeIdsForUser(meadowId int, treeId int64, shouldDelete bool, userID int) error {
	// Get current meadow
	meadow := FindOneMeadowByIdForUser(meadowId, userID)

	fmt.Printf("Current TreeIds for meadow %d: %v\n", meadowId, meadow.TreeIds)

	// Check whether to add or remove the tree ID
	if shouldDelete {
		newTreeIds := make([]int, 0, len(meadow.TreeIds))
		for _, id := range meadow.TreeIds {
			if id != int(treeId) {
				newTreeIds = append(newTreeIds, id)
			}
		}
		meadow.TreeIds = newTreeIds
	} else {
		meadow.TreeIds = append(meadow.TreeIds, int(treeId))
	}

	fmt.Printf("New TreeIds for meadow %d: %v\n", meadowId, meadow.TreeIds)

	// Value() method will automatically be called for TreeIds
	result, err := DB.Exec("UPDATE meadows SET TreeIds = ? WHERE ID = ? AND user_id = ?", meadow.TreeIds, meadowId, userID)
	if err != nil {
		fmt.Printf("ERROR executing UPDATE: %v\n", err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("UPDATE affected %d rows\n", rowsAffected)

	if rowsAffected == 0 {
		return fmt.Errorf("no meadow found with ID %d", meadowId)
	}

	fmt.Printf("Successfully updated meadow %d with tree ID: %d\n", meadowId, treeId)
	return nil
}

func UpdateMeadowForUser(meadow models.Meadow, userID int) {

	result, err := DB.Exec("UPDATE meadows SET Location = ?, Name = ?, Size = ? WHERE ID = ? AND user_id = ?",
		meadow.Location, meadow.Name, meadow.Size, meadow.ID, userID)
	if err != nil {
		panic(err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("UPDATE affected %d rows\n", rowsAffected)

	fmt.Printf("Successfully updated meadow %d\n", meadow.ID)
}

func UpdateTreeForUser(tree models.Tree, userID int) {

	result, err := DB.Exec("UPDATE trees SET PlantDate = ?, Position = ?, Type = ? WHERE ID = ? AND user_id = ?",
		tree.PlantDate, tree.Position, tree.Type, tree.ID, userID)
	if err != nil {
		panic(err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("UPDATE affected %d rows\n", rowsAffected)

	fmt.Printf("Successfully updated tree %d\n", tree.ID)
}

func UpdateTreeImageDescriptionDb(imageID int, description string, userID int) error {
	result, err := DB.Exec("UPDATE images SET description = ? WHERE id = ? AND user_id = ?",
		description, imageID, userID)
	if err != nil {
		return fmt.Errorf("failed to update image description: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no image found with ID %d and user ID %d", imageID, userID)
	}

	fmt.Printf("Updated image description for image %d\n", imageID)
	return nil
}

func UpdateTreeImageDatetimeDb(imageID int, datetime time.Time, userID int) error {
	result, err := DB.Exec("UPDATE images SET datetime = ? WHERE id = ? AND user_id = ?",
		datetime, imageID, userID)
	if err != nil {
		return fmt.Errorf("failed to update image datetime: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no image found with ID %d and user ID %d", imageID, userID)
	}

	fmt.Printf("Updated image datetime for image %d\n", imageID)
	return nil
}

func UploadImageDb(path string, description string, userID int, treeID int) error {
	_, err := DB.Exec("INSERT INTO images (path, description, user_id, tree_id) VALUES (?, ?, ?, ?)",
		path, description, userID, treeID)
	if err != nil {
		return fmt.Errorf("failed to upload image to database: %w", err)
	}

	fmt.Printf("Uploaded image for user %d with path: %s\n", userID, path)
	return nil
}

// Deletes only the tree from the database, does not update meadow's TreeIds
func deleteTreeOnly(treeId int, userID int) error {
	result, err := DB.Exec("DELETE FROM trees WHERE ID = ? AND user_id = ?", treeId, userID)
	if err != nil {
		return fmt.Errorf("failed to delete tree: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no tree found with ID %d and user ID %d", treeId, userID)
	}

	fmt.Printf("Deleted tree for user %d with ID: %d\n", userID, treeId)
	return nil
}
