package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const (
	maxUploadSize = 500 * 1024 * 1024
	uploadPath = "./uploads"
	staticPath = "./static"
)

func main() {
	if err := os.MkdirAll(uploadPath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/upload", handleUpload)

	port := ":8080"
	log.Printf("Server starting on port %s", port)
	log.Printf("Uploads will be saved to: %s", uploadPath)
	log.Printf("Open http://localhost%s in your browser to upload videos", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		indexPath := filepath.Join(staticPath, "index.html")
		http.ServeFile(w, r, indexPath)
		return
	}
	
	http.NotFound(w, r)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, fmt.Sprintf("File too large or invalid form: %v", err), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("video")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Printf("Received upload: %s, Size: %d bytes, Content-Type: %s",
		handler.Filename, handler.Size, handler.Header.Get("Content-Type"))

	uuid := uuid.Must(uuid.NewRandom())
	ext := filepath.Ext(handler.Filename)
	baseFilename := handler.Filename[:len(handler.Filename)-len(ext)]
	newFilename := fmt.Sprintf("%s_%d%s", baseFilename, uuid, ext)
	destPath := filepath.Join(uploadPath, newFilename)

	dst, err := os.Create(destPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create destination file: %v", err), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	bytesWritten, err := io.Copy(dst, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully saved: %s (%d bytes)", destPath, bytesWritten)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Upload successful!\n")
	fmt.Fprintf(w, "Filename: %s\n", newFilename)
	fmt.Fprintf(w, "Size: %d bytes\n", bytesWritten)
	fmt.Fprintf(w, "Saved to: %s\n", destPath)
}
