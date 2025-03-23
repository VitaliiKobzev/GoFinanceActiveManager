package main

import (
	"fmt"
	"net/http"
)

// Middleware для обработки CORS
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Устанавливаем заголовки CORS
		w.Header().Set("Access-Control-Allow-Origin", "*") // Разрешаем все домены
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
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
	// Обработчик для index.html
	http.HandleFunc("/portfolio", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./client/index.html")
	}))

	// Обработчик для portfolio.html
	http.HandleFunc("/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./client/portfolio.html")
	}))

	fmt.Println("Client server started on :8000")
	http.ListenAndServe(":8000", nil)
}
