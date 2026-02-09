package main

import (
	"log"
	"net/http"
	"os"

	adapthttp "github.com/george/golinks/internal/adapter/http"
	"github.com/george/golinks/internal/adapter/postgres"
	"github.com/george/golinks/internal/domain"
	"github.com/gorilla/mux"
)

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is required")
	}

	repo, err := postgres.NewRepository(connStr)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	svc := domain.NewLinkService(repo)
	handler := adapthttp.NewHandler(svc)
	r := mux.NewRouter()
	handler.RegisterRoutes(r)

	port := os.Getenv("GOLINKS_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("GoLinks server starting on http://localhost:%s", port)
	log.Printf("Admin UI available at http://localhost:%s/admin", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
