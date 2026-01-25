package main

import (
	"log"
	"net/http"
	"os"

	"github.com/george/golinks/handlers"
	"github.com/george/golinks/storage"
	"github.com/gorilla/mux"
)

func main() {
	// Initialize storage
	dbPath := os.Getenv("GOLINKS_DB_PATH")
	if dbPath == "" {
		dbPath = "./golinks.db"
	}

	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Create handler with storage
	h := handlers.NewHandler(store)

	// Setup router
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/links", h.ListLinks).Methods("GET")
	api.HandleFunc("/links", h.CreateLink).Methods("POST")
	api.HandleFunc("/links/{shortcode}", h.GetLink).Methods("GET")
	api.HandleFunc("/links/{shortcode}", h.UpdateLink).Methods("PUT")
	api.HandleFunc("/links/{shortcode}", h.DeleteLink).Methods("DELETE")
	api.HandleFunc("/links/{shortcode}/stats", h.GetLinkStats).Methods("GET")

	// Static files and UI
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	r.HandleFunc("/admin", h.AdminPage).Methods("GET")
	r.HandleFunc("/admin/", h.AdminPage).Methods("GET")

	// Redirect handler (catch-all, must be last)
	r.HandleFunc("/{shortcode}", h.Redirect).Methods("GET")
	r.HandleFunc("/", h.HomePage).Methods("GET")

	// Get port from environment or default
	port := os.Getenv("GOLINKS_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("GoLinks server starting on http://localhost:%s", port)
	log.Printf("Admin UI available at http://localhost:%s/admin", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
