package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"comparify/internal/handler"
	"comparify/internal/model"
	"comparify/internal/repository"
	"comparify/internal/server"
	"comparify/internal/service"
	"comparify/pkg/logger"

	_ "modernc.org/sqlite"
)

func main() {
	// Inicializa logger estruturado
	logger.Init()
	defer logger.Sync()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Abrir SQLite em memória
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Rodar o schema
	schema, err := os.ReadFile("data/schema.sql")
	if err != nil {
		log.Fatalf("failed to read schema: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		log.Fatalf("failed to exec schema: %v", err)
	}

	// Popular seed
	if err := repository.SeedSQLite(db, seedProducts()); err != nil {
		log.Fatalf("failed to seed sqlite: %v", err)
	}

	repo := repository.NewSQLiteRepository(db)
	service := service.NewProductService(repo)
	h := handler.NewHandler(service)
	srv := server.NewServer(h)

	log.Printf("item comparison API listening on :%s", port)
	if err := http.ListenAndServe(":"+port, srv.Routes()); err != nil {
		log.Fatal(err)
	}
}

// seedProducts lê os produtos do arquivo JSON
func seedProducts() []model.Product {
	data, err := os.ReadFile("data/products.json")
	if err != nil {
		log.Fatalf("failed to read products.json: %v", err)
	}
	var products []model.Product
	if err := json.Unmarshal(data, &products); err != nil {
		log.Fatalf("failed to unmarshal products.json: %v", err)
	}
	return products
}
