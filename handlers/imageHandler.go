package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
)

// set the upload path inside the container
var uploadPath = "./uploads"

func UploadImageHandler(w http.ResponseWriter, r *http.Request) *os.File {
	// Limit file size to 10MB. This line saves you from those accidental 100MB uploads!
	r.ParseMultipartForm(10 << 20)

	// Retrieve the file from form data
	file, handler, err := r.FormFile("treeImage")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return nil
	}
	defer file.Close()

	// Read the file into a byte slice to validate its type
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file")
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return nil
	}

	if !isValidFileType(fileBytes) {
		fmt.Println("Invalid file type")
		http.Error(w, "Invalid file type", http.StatusUnsupportedMediaType)
		return nil
	}

	fmt.Fprintf(w, "Uploaded File: %s\n", handler.Filename)
	fmt.Fprintf(w, "File Size: %d\n", handler.Size)
	fmt.Fprintf(w, "MIME Header: %v\n", handler.Header)

	// Now let’s save it locally
	dst, err := createFile(handler.Filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		http.Error(w, "Error creating the file", http.StatusInternalServerError)
		return nil
	}
	defer dst.Close()

	// Copy the uploaded file to the destination file
	if _, err := dst.ReadFrom(file); err != nil {
		fmt.Println("Error copying file:", err)
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return nil
	}

	fmt.Printf("Successfully saved file: %s\n", handler.Filename)
	return dst
}

func createFile(filename string) (*os.File, error) {
	// Ensure the upload directory exists
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Build the file path and create it
	dst, err := os.Create(filepath.Join(uploadPath, filename))
	if err != nil {
		return nil, err
	}

	return dst, nil
}

// only allow jpg, png, jpeg
func isValidFileType(file []byte) bool {
	fileType := http.DetectContentType(file)

	allowed := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
	}

	return allowed[fileType]
}
