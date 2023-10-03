package dropper

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	dropperport = "8080"
)

func serveFile(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Path[1:]
	filePath := filepath.Join("dropper", fileName)

	_, err := os.Stat(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Error opening the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Error sending file data", http.StatusInternalServerError)
		return
	}
}

func Dropstart() {
	http.HandleFunc("/", serveFile)
	http.ListenAndServe(fmt.Sprintf(":%s", dropperport), nil)
}
