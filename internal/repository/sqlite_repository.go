package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"comparify/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteRepository struct {
	db *sql.DB
}

type productScanner interface {
	Scan(dest ...any) error
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

const specsModelColumn = "model"

func (r *SQLiteRepository) ListByIDs(ctx context.Context, ids []string) ([]model.Product, error) {
	if len(ids) == 0 {
		return []model.Product{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for index, id := range ids {
		placeholders[index] = "?"
		args[index] = id
	}

	query := fmt.Sprintf(
		"SELECT id, name, image_url, description, price, rating, size, weight, color, type, model FROM products WHERE id IN (%s)",
		strings.Join(placeholders, ","),
	)
	products, err := r.queryProducts(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if len(products) != len(ids) {
		return nil, ErrProductNotFound
	}

	productsByID := make(map[string]model.Product, len(products))
	for _, product := range products {
		productsByID[product.ID] = product
	}

	// O IN do SQLite não preserva a ordem de entrada; reordenamos para refletir o compare pedido.
	orderedProducts := make([]model.Product, 0, len(ids))
	for _, id := range ids {
		product, ok := productsByID[id]
		if !ok {
			return nil, ErrProductNotFound
		}
		orderedProducts = append(orderedProducts, product)
	}

	return orderedProducts, nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (model.Product, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, name, image_url, description, price, rating, size, weight, color, type, model FROM products WHERE id = ?", id)
	product, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Product{}, ErrProductNotFound
		}
		return model.Product{}, err
	}
	return product, nil
}

func (r *SQLiteRepository) queryProducts(ctx context.Context, query string, args ...any) ([]model.Product, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]model.Product, 0)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func scanProduct(scanner productScanner) (model.Product, error) {
	var product model.Product
	if err := scanner.Scan(
		&product.ID,
		&product.Name,
		&product.ImageURL,
		&product.Description,
		&product.Price,
		&product.Rating,
		&product.Size,
		&product.Weight,
		&product.Color,
		&product.Type,
		&product.Model,
	); err != nil {
		return model.Product{}, err
	}

	return product, nil
}

// SeedSQLite insere produtos de seed no banco SQLite dentro de uma única transação.
func SeedSQLite(ctx context.Context, db *sql.DB, products []model.Product) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin seed transaction: %w", err)
	}
	defer tx.Rollback()

	type specsInserter func(tx *sql.Tx, p model.Product) error
	specsInserters := map[string]specsInserter{
		"smartphone_specs": func(tx *sql.Tx, p model.Product) error {
			_, err := tx.Exec(`INSERT OR IGNORE INTO smartphone_specs (model, battery_capacity, camera_specs, memory, storage_capacity, brand, operating_system) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				resolveProductModel(p), p.Specifications["batteryCapacity"], p.Specifications["cameraSpecifications"],
				p.Specifications["memory"], p.Specifications["storageCapacity"], p.Specifications["brand"],
				p.Specifications["operatingSystem"])
			return err
		},
		"fridge_specs": func(tx *sql.Tx, p model.Product) error {
			_, err := tx.Exec(`INSERT OR IGNORE INTO fridge_specs (model, capacity, energy_class, brand) VALUES (?, ?, ?, ?)`,
				resolveProductModel(p), p.Specifications["capacity"], p.Specifications["energyClass"],
				p.Specifications["brand"])
			return err
		},
		"microwave_specs": func(tx *sql.Tx, p model.Product) error {
			_, err := tx.Exec(`INSERT OR IGNORE INTO microwave_specs (model, capacity, power, brand) VALUES (?, ?, ?, ?)`,
				resolveProductModel(p), p.Specifications["capacity"], p.Specifications["power"],
				p.Specifications["brand"])
			return err
		},
		"speaker_specs": func(tx *sql.Tx, p model.Product) error {
			_, err := tx.Exec(`INSERT OR IGNORE INTO speaker_specs (model, battery_capacity, connectivity, brand) VALUES (?, ?, ?, ?)`,
				resolveProductModel(p), p.Specifications["batteryCapacity"], p.Specifications["connectivity"],
				p.Specifications["brand"])
			return err
		},
	}

	for _, product := range products {
		specsTable, err := specsTableForTypeTx(tx, product.Type)
		if err != nil {
			return err
		}
		modelName := resolveProductModel(product)
		// O produto sempre ganha sua própria linha, mas as specs são reaproveitadas por model.
		_, err = tx.Exec(`INSERT INTO products (id, name, image_url, description, price, rating, size, weight, color, type, model)
                           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			product.ID, product.Name, product.ImageURL, product.Description, product.Price,
			product.Rating, product.Size, product.Weight, product.Color, product.Type, modelName)
		if err != nil {
			return fmt.Errorf("failed to insert product %s: %w", product.ID, err)
		}
		if err := specsInserters[specsTable](tx, product); err != nil {
			return fmt.Errorf("failed to insert specs for product %s: %w", product.ID, err)
		}
	}

	return tx.Commit()
}

// resolveProductModel normaliza a chave usada para reaproveitar especificações por modelo.
func resolveProductModel(product model.Product) string {
	if strings.TrimSpace(product.Model) != "" {
		return strings.TrimSpace(product.Model)
	}

	if modelVersion := strings.TrimSpace(product.Specifications["modelVersion"]); modelVersion != "" {
		return modelVersion
	}

	return strings.TrimSpace(product.Name)
}

func (r *SQLiteRepository) GetSpecificationsByModel(ctx context.Context, modelName, productType string) (map[string]string, error) {
	batch, err := r.GetSpecificationsBatch(ctx, []string{modelName}, productType)
	if err != nil {
		return nil, err
	}
	specs, ok := batch[modelName]
	if !ok {
		return map[string]string{}, nil
	}
	return specs, nil
}

// GetSpecificationsBatch busca specs de múltiplos modelos da mesma tabela em uma única query.
// A tabela é derivada diretamente do tipo do produto.
// Retorna map[model]map[coluna]valor — reutiliza a mesma linha de specs para vários produtos.
func (r *SQLiteRepository) GetSpecificationsBatch(ctx context.Context, models []string, productType string) (map[string]map[string]string, error) {
	if len(models) == 0 {
		return map[string]map[string]string{}, nil
	}
	specsTable, err := r.specsTableForType(ctx, productType)
	if err != nil {
		return nil, err
	}

	placeholders := make([]string, len(models))
	args := make([]any, len(models))
	for i, modelName := range models {
		placeholders[i] = "?"
		args[i] = modelName
	}
	// SELECT * — colunas descobertas dinamicamente via rows.Columns(), sem hardcode
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", specsTable, specsModelColumn, strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]string, len(models))
	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		rowSpecs := make(map[string]string, len(columns)-1)
		var currentModel string
		for i, col := range columns {
			if col == specsModelColumn {
				currentModel = values[i].String
				continue
			}
			if values[i].Valid {
				rowSpecs[col] = values[i].String
			}
		}
		result[currentModel] = rowSpecs
	}
	return result, nil
}

func (r *SQLiteRepository) specsTableForType(ctx context.Context, productType string) (string, error) {
	row := r.db.QueryRowContext(ctx, "SELECT specs_table FROM product_type_specs WHERE product_type = ?", productType)
	var specsTable string
	if err := row.Scan(&specsTable); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("unsupported product type: %s", productType)
		}
		return "", err
	}

	return specsTable, nil
}

func specsTableForTypeTx(tx *sql.Tx, productType string) (string, error) {
	row := tx.QueryRow("SELECT specs_table FROM product_type_specs WHERE product_type = ?", productType)
	var specsTable string
	if err := row.Scan(&specsTable); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("unsupported product type: %s", productType)
		}
		return "", err
	}

	return specsTable, nil
}
