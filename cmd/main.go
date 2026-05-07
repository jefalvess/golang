package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"comparify/internal/handler"
	"comparify/internal/model"
	"comparify/internal/repository"
	"comparify/internal/server"
	"comparify/internal/service"
	"comparify/pkg/logger"
)

func main() {
	logger.Init()
	defer logger.Sync()

	if err := run(); err != nil {
		logger.Logger.Fatalw("failed to start application", "error", err)
	}
}

func run() error {
	port := resolvePort()

	db, err := openInMemoryDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := initializeDatabase(db); err != nil {
		return err
	}

	productRepository := repository.NewSQLiteRepository(db)
	productService := service.NewProductService(productRepository)
	productHandler := handler.NewHandler(productService)
	applicationServer := server.NewServer(productHandler)

	logger.Logger.Infow("item comparison API listening", "port", port)
	return applicationServer.Start(":" + port)
}

func resolvePort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "8080"
	}

	return port
}

func openInMemoryDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "file:comparify?mode=memory&cache=shared")
	if err != nil {
		return nil, fmt.Errorf("open sqlite in memory: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite in memory: %w", err)
	}

	return db, nil
}

func initializeDatabase(db *sql.DB) error {
	schema, err := os.ReadFile("data/schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}

	if err := seedDatabase(db); err != nil {
		return err
	}

	return nil
}

func seedDatabase(db *sql.DB) error {
	products, err := readSeedProducts("data/products.json")
	if err != nil {
		return err
	}

	if err := repository.SeedSQLite(db, products); err != nil {
		return fmt.Errorf("seed sqlite: %w", err)
	}

	return nil
}

func readSeedProducts(filePath string) ([]model.Product, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	var products []model.Product
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", filePath, err)
	}

	return products, nil
}
