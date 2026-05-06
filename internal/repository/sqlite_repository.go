package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"comparify/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

// typeToSpecsTable mapeia o tipo do produto para o nome da tabela de specs.
// Usado apenas no seed (escrita). Leituras usam o campo specs_table gravado no produto.
var typeToSpecsTable = map[string]string{
	"celular":      "smartphone_specs",
	"geladeira":    "fridge_specs",
	"micro-ondas":  "microwave_specs",
	"caixa de som": "speaker_specs",
}

// validSpecsTables é a lista de tabelas de specs permitidas.
// Evita SQL injection ao usar specs_table como nome de tabela em queries dinâmicas.
var validSpecsTables = map[string]struct{}{
	"smartphone_specs": {},
	"fridge_specs":     {},
	"microwave_specs":  {},
	"speaker_specs":    {},
}

type SQLiteRepository struct {
	db *sql.DB
}

func (r *SQLiteRepository) ListByFilters(filters map[string]string) ([]model.Product, error) {
	var (
		query string
		args  []any
	)
	whereClauses := []string{}
	for filterKey, filterValue := range filters {
		filterValues := strings.Split(filterValue, ",")
		valuePlaceholders := make([]string, len(filterValues))
		for i := range filterValues {
			valuePlaceholders[i] = "?"
		}
		switch filterKey {
		case "brand":
			// Consulta a tabela unificada product_brands — sem UNION entre tabelas de specs
			brandPlaceholders := make([]string, len(filterValues))
			for i := range filterValues {
				brandPlaceholders[i] = "?"
			}
			whereClauses = append(whereClauses, fmt.Sprintf(
				"id IN (SELECT product_id FROM product_brands WHERE brand IN (%s))",
				join(brandPlaceholders, ","),
			))
			for _, brandValue := range filterValues {
				args = append(args, strings.TrimSpace(brandValue))
			}
			continue
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s IN (%s)", filterKey, join(valuePlaceholders, ",")))
		for _, filterVal := range filterValues {
			args = append(args, strings.TrimSpace(filterVal))
		}
	}
	query = "SELECT id, name, image_url, description, price, rating, size, weight, color, type, specs_table FROM products"
	if len(whereClauses) > 0 {
		query += " WHERE " + join(whereClauses, " AND ")
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []model.Product
	for rows.Next() {
		var scannedProduct model.Product
		if err := rows.Scan(&scannedProduct.ID, &scannedProduct.Name, &scannedProduct.ImageURL, &scannedProduct.Description, &scannedProduct.Price, &scannedProduct.Rating, &scannedProduct.Size, &scannedProduct.Weight, &scannedProduct.Color, &scannedProduct.Type, &scannedProduct.SpecsTable); err != nil {
			return nil, err
		}
		products = append(products, scannedProduct)
	}
	if len(products) == 0 {
		return nil, ErrProductNotFound
	}
	return products, nil
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) ListByIDs(ids []string) ([]model.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf("SELECT id, name, image_url, description, price, rating, size, weight, color, type, specs_table FROM products WHERE id IN (%s)",
		join(placeholders, ","))
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.ImageURL, &p.Description, &p.Price, &p.Rating, &p.Size, &p.Weight, &p.Color, &p.Type, &p.SpecsTable); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if len(products) != len(ids) {
		return nil, ErrProductNotFound
	}
	return products, nil
}

func (r *SQLiteRepository) GetByID(id string) (model.Product, error) {
	row := r.db.QueryRow("SELECT id, name, image_url, description, price, rating, size, weight, color, type, specs_table FROM products WHERE id = ?", id)
	var p model.Product
	if err := row.Scan(&p.ID, &p.Name, &p.ImageURL, &p.Description, &p.Price, &p.Rating, &p.Size, &p.Weight, &p.Color, &p.Type, &p.SpecsTable); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Product{}, ErrProductNotFound
		}
		return model.Product{}, err
	}
	return p, nil
}

// join builds a comma-separated placeholder string for SQL IN clauses.
func join(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

// SeedSQLite insere produtos de seed no banco SQLite dentro de uma única transação.
func SeedSQLite(db *sql.DB, products []model.Product) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin seed transaction: %w", err)
	}
	defer tx.Rollback()

	type specsInserter func(tx *sql.Tx, p model.Product) error
	specsInserters := map[string]specsInserter{
		"smartphone_specs": func(tx *sql.Tx, p model.Product) error {
			_, err := tx.Exec(`INSERT INTO smartphone_specs (product_id, battery_capacity, camera_specs, memory, storage_capacity, brand, model_version, operating_system) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				p.ID, p.Specifications["batteryCapacity"], p.Specifications["cameraSpecifications"],
				p.Specifications["memory"], p.Specifications["storageCapacity"], p.Specifications["brand"],
				p.Specifications["modelVersion"], p.Specifications["operatingSystem"])
			return err
		},
		"fridge_specs": func(tx *sql.Tx, p model.Product) error {
			_, err := tx.Exec(`INSERT INTO fridge_specs (product_id, capacity, energy_class, brand, model_version) VALUES (?, ?, ?, ?, ?)`,
				p.ID, p.Specifications["capacity"], p.Specifications["energyClass"],
				p.Specifications["brand"], p.Specifications["modelVersion"])
			return err
		},
		"microwave_specs": func(tx *sql.Tx, p model.Product) error {
			_, err := tx.Exec(`INSERT INTO microwave_specs (product_id, capacity, power, brand, model_version) VALUES (?, ?, ?, ?, ?)`,
				p.ID, p.Specifications["capacity"], p.Specifications["power"],
				p.Specifications["brand"], p.Specifications["modelVersion"])
			return err
		},
		"speaker_specs": func(tx *sql.Tx, p model.Product) error {
			_, err := tx.Exec(`INSERT INTO speaker_specs (product_id, battery_capacity, connectivity, brand) VALUES (?, ?, ?, ?)`,
				p.ID, p.Specifications["batteryCapacity"], p.Specifications["connectivity"],
				p.Specifications["brand"])
			return err
		},
	}

	for _, product := range products {
		specsTable, ok := typeToSpecsTable[product.Type]
		if !ok {
			return fmt.Errorf("unknown product type: %s", product.Type)
		}
		_, err := tx.Exec(`INSERT INTO products (id, name, image_url, description, price, rating, size, weight, color, type, specs_table)
                           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			product.ID, product.Name, product.ImageURL, product.Description, product.Price,
			product.Rating, product.Size, product.Weight, product.Color, product.Type, specsTable)
		if err != nil {
			return fmt.Errorf("failed to insert product %s: %w", product.ID, err)
		}
		if err := specsInserters[specsTable](tx, product); err != nil {
			return fmt.Errorf("failed to insert specs for product %s: %w", product.ID, err)
		}
		if _, err := tx.Exec(`INSERT INTO product_brands (product_id, brand) VALUES (?, ?)`,
			product.ID, product.Specifications["brand"]); err != nil {
			return fmt.Errorf("failed to insert brand for product %s: %w", product.ID, err)
		}
	}

	return tx.Commit()
}

func (r *SQLiteRepository) GetSpecificationsByType(productID, productType string) (map[string]string, error) {
	batch, err := r.GetSpecificationsBatch([]string{productID}, productType)
	if err != nil {
		return nil, err
	}
	specs, ok := batch[productID]
	if !ok {
		return map[string]string{}, nil
	}
	return specs, nil
}

// GetSpecificationsBatch busca specs de múltiplos produtos da mesma tabela em uma única query.
// specsTable deve ser um valor lido do campo specs_table do produto (validado contra whitelist).
// Retorna map[productID]map[coluna]valor — resolve o problema N+1 no Compare.
func (r *SQLiteRepository) GetSpecificationsBatch(ids []string, specsTable string) (map[string]map[string]string, error) {
	if len(ids) == 0 {
		return map[string]map[string]string{}, nil
	}
	if _, ok := validSpecsTables[specsTable]; !ok {
		return nil, fmt.Errorf("unsupported specs table: %s", specsTable)
	}
	const productIDColumn = "product_id"

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	// SELECT * — colunas descobertas dinamicamente via rows.Columns(), sem hardcode
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", specsTable, productIDColumn, join(placeholders, ","))
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]string, len(ids))
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
		var currentProductID string
		for i, col := range columns {
			if col == productIDColumn {
				currentProductID = values[i].String
				continue
			}
			if values[i].Valid {
				rowSpecs[col] = values[i].String
			}
		}
		result[currentProductID] = rowSpecs
	}
	return result, nil
}
