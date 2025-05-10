package main

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
)

func fileHandler(w http.ResponseWriter, r *http.Request) {
	filePath := "." + r.URL.Path
	ext := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, filePath)
}

// Middleware для обработки CORS
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Устанавливаем заголовки CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Обработка предварительных запросов (preflight)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Передаем запрос следующему обработчику
		next(w, r)
	}
}

func main() {
	fs := http.FileServer(http.Dir("./client")) // Раздаем файлы из текущей папки
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./client"))))

	http.Handle("/", fs)

	// Обработчик для index.html
	http.HandleFunc("/portfolio", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./client/portfolio.html")
	}))

	http.HandleFunc("/portfolio/settings", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "Missing portfolio name", http.StatusBadRequest)
			return
		}
		http.ServeFile(w, r, "./client/portfolio-settings.html")
	}))

	// Обработчик для portfolio.html
	/*http.HandleFunc("/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./client/index.html")
	}))*/

	fmt.Println("Client server started on :8000")
	http.ListenAndServe(":8000", nil)
}
