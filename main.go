package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	// Create application instance
	app := NewApp()

	// Create router
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/API").Subrouter()
	api.HandleFunc("/upload", app.UploadHandler).Methods("POST")
	api.HandleFunc("/download/{id}", app.DownloadHandler).Methods("GET")

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}).Methods("GET")

	// Start server
	port := "8080"
	fmt.Printf("Server starting on port %s\n", port)
	fmt.Println("Available endpoints:")
	fmt.Println("  POST /API/upload - Upload CSV file")
	fmt.Println("  GET  /API/download/{id} - Download processed file")
	fmt.Println("  GET  /health - Health check")

	log.Fatal(http.ListenAndServe(":"+port, router))
}
