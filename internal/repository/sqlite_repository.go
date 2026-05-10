package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"comparify/internal/model"
	"comparify/pkg/customerror"

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

// specsTableConfig descreve, de forma declarativa, como persistir as specs de um tipo de produto.
// Cada coluna em columns recebe o valor extraído de Product.Specifications na mesma posição.
type specsTableConfig struct {
	columns []string
	values  func(model.Product) []any
}

var specsConfigByTable = map[string]specsTableConfig{
	"smartphone_specs": {
		columns: []string{"model_version", "battery_capacity", "camera_specs", "memory", "storage_capacity", "brand", "operating_system"},
		values: func(product model.Product) []any {
			specs := product.Specifications
			return []any{specs["modelVersion"], specs["batteryCapacity"], specs["cameraSpecifications"], specs["memory"], specs["storageCapacity"], specs["brand"], specs["operatingSystem"]}
		},
	},
	"fridge_specs": {
		columns: []string{"capacity", "energy_class", "brand"},
		values: func(product model.Product) []any {
			specs := product.Specifications
			return []any{specs["capacity"], specs["energyClass"], specs["brand"]}
		},
	},
	"microwave_specs": {
		columns: []string{"capacity", "power", "brand"},
		values: func(product model.Product) []any {
			specs := product.Specifications
			return []any{specs["capacity"], specs["power"], specs["brand"]}
		},
	},
	"speaker_specs": {
		columns: []string{"battery_capacity", "connectivity", "brand"},
		values: func(product model.Product) []any {
			specs := product.Specifications
			return []any{specs["batteryCapacity"], specs["connectivity"], specs["brand"]}
		},
	},
}

// ListByIDs retorna os produtos correspondentes aos ids solicitados.
func (r *SQLiteRepository) ListByIDs(ctx context.Context, ids []string) ([]model.Product, error) {
	const componentName = "SQLiteRepository.ListByIDs"

	if len(ids) == 0 {
		return []model.Product{}, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	query := fmt.Sprintf(
		"SELECT id, name, image_url, description, price, rating, size, weight, color, type, model FROM products WHERE id IN (%s)",
		placeholders,
	)
	products, err := r.queryProducts(ctx, query, args...)
	if err != nil {
		return nil, customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}
	if len(products) != len(ids) {
		return nil, customerror.ThrowNew(componentName, customerror.NotFoundError, ErrProductNotFound)
	}

	if err := r.populateSpecifications(ctx, products); err != nil {
		return nil, customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}

	// O IN do SQLite não preserva a ordem de entrada; reordenamos para refletir o compare pedido.
	// A checagem len(products) != len(ids) acima já garante que todos os ids foram encontrados.
	productsByID := make(map[string]model.Product, len(products))
	for _, product := range products {
		productsByID[product.ID] = product
	}
	ordered := make([]model.Product, len(ids))
	for idx, id := range ids {
		ordered[idx] = productsByID[id]
	}
	return ordered, nil
}

// ListAll retorna todos os produtos cadastrados.
func (r *SQLiteRepository) ListAll(ctx context.Context) ([]model.Product, error) {
	const componentName = "SQLiteRepository.ListAll"

	products, err := r.queryProducts(ctx,
		"SELECT id, name, image_url, description, price, rating, size, weight, color, type, model FROM products",
	)
	if err != nil {
		return nil, customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}
	if err := r.populateSpecifications(ctx, products); err != nil {
		return nil, customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}
	return products, nil
}

// populateSpecifications preenche product.Specifications in-place para uma lista de produtos.
// Agrupa por tipo e executa as queries em paralelo — uma por tipo, nunca N+1.
// Nota: o paralelismo real depende de MaxOpenConns > 1 no pool de conexões.
// Tipos sem tabela mapeada são silenciosamente ignorados (specs ficam nil).
func (r *SQLiteRepository) populateSpecifications(ctx context.Context, products []model.Product) error {
	// Formato: productType → modelKey → índices em products (permite escrita sem N+1).
	indexByType := make(map[string]map[string][]int)
	for idx, product := range products {
		if indexByType[product.Type] == nil {
			indexByType[product.Type] = make(map[string][]int)
		}
		indexByType[product.Type][product.Model] = append(indexByType[product.Type][product.Model], idx)
	}

	// Cada productType é independente — busca specs em paralelo.
	// Seguro: goroutines de tipos distintos escrevem em índices disjuntos de products.
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	for productType, modelIndices := range indexByType {
		wg.Add(1)
		go func(productType string, modelIndices map[string][]int) {
			defer wg.Done()
			if err := r.fetchAndApplySpecsForType(ctx, productType, modelIndices, products); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(productType, modelIndices)
	}
	wg.Wait()
	return firstErr
}

// fetchAndApplySpecsForType busca as specs de um tipo e preenche products in-place.
// modelIndices mapeia cada modelKey para os índices em products que pertencem a ele.
// Seguro para uso concorrente: cada productType escreve em índices disjuntos.
func (r *SQLiteRepository) fetchAndApplySpecsForType(
	ctx context.Context,
	productType string,
	modelIndices map[string][]int,
	products []model.Product,
) error {
	componentName := fmt.Sprintf("SQLiteRepository.fetchAndApplySpecsForType[%s]", productType)

	specsTable, err := r.lookupSpecsTable(ctx, productType)
	if err != nil {
		// Tipo sem tabela de specs mapeada é ignorado silenciosamente.
		return nil
	}

	modelKeys := make([]string, 0, len(modelIndices))
	for modelKey := range modelIndices {
		modelKeys = append(modelKeys, modelKey)
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(modelKeys)), ",")
	args := make([]any, len(modelKeys))
	for idx, modelKey := range modelKeys {
		args[idx] = modelKey
	}

	// SELECT * para descobrir colunas dinamicamente; model é a PK e serve de chave de lookup.
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", specsTable, specsModelColumn, placeholders)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}

	// Localizar a posição da coluna model para usá-la como chave, sem depender de índice fixo.
	modelColumnIdx := -1
	for idx, col := range columns {
		if col == specsModelColumn {
			modelColumnIdx = idx
			break
		}
	}

	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		scanDest := make([]any, len(columns))
		for idx := range values {
			scanDest[idx] = &values[idx]
		}
		if err := rows.Scan(scanDest...); err != nil {
			return customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
		}

		rowModel := values[modelColumnIdx].String
		specs := make(map[string]string, len(columns)-1)
		for colIdx, colName := range columns {
			// Excluir a coluna model das specs expostas ao domínio.
			if colIdx != modelColumnIdx && values[colIdx].Valid {
				specs[colName] = values[colIdx].String
			}
		}

		for _, productIdx := range modelIndices[rowModel] {
			products[productIdx].Specifications = specs
		}
	}
	if err := rows.Err(); err != nil {
		return customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}

	return nil
}

func (r *SQLiteRepository) queryProducts(ctx context.Context, query string, args ...any) ([]model.Product, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, customerror.ThrowNew("SQLiteRepository.queryProducts", customerror.RequestExecutionError, err)
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, customerror.ThrowNew("SQLiteRepository.queryProducts", customerror.RequestExecutionError, err)
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, customerror.ThrowNew("SQLiteRepository.queryProducts", customerror.RequestExecutionError, err)
	}

	return products, nil
}

func scanProduct(scanner productScanner) (model.Product, error) {
	var product model.Product
	err := scanner.Scan(
		&product.ID, &product.Name, &product.ImageURL, &product.Description,
		&product.Price, &product.Rating, &product.Size, &product.Weight,
		&product.Color, &product.Type, &product.Model,
	)
	return product, err
}

// SeedSQLite insere produtos de seed no banco SQLite dentro de uma única transação.
func SeedSQLite(ctx context.Context, db *sql.DB, products []model.Product) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin seed transaction: %w", err)
	}
	defer tx.Rollback()

	for _, product := range products {
		specsTable, err := lookupSpecsTableTx(tx, product.Type)
		if err != nil {
			return err
		}
		modelName := resolveModelKey(product)
		// O produto sempre ganha sua própria linha, mas as specs são reaproveitadas por model.
		_, err = tx.Exec(`INSERT INTO products (id, name, image_url, description, price, rating, size, weight, color, type, model)
                           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			product.ID, product.Name, product.ImageURL, product.Description, product.Price,
			product.Rating, product.Size, product.Weight, product.Color, product.Type, modelName)
		if err != nil {
			return fmt.Errorf("failed to insert product %s: %w", product.ID, err)
		}
		if err := insertSpecifications(tx, specsTable, modelName, product); err != nil {
			return fmt.Errorf("failed to insert specs for product %s: %w", product.ID, err)
		}
	}

	return tx.Commit()
}

// insertSpecs monta um INSERT a partir da config do tipo, evitando uma função por tabela.
func insertSpecifications(tx *sql.Tx, specsTable, modelName string, product model.Product) error {
	cfg, ok := specsConfigByTable[specsTable]
	if !ok {
		return fmt.Errorf("no specs config for table %s", specsTable)
	}
	columns := append([]string{specsModelColumn}, cfg.columns...)
	args := append([]any{modelName}, cfg.values(product)...)
	placeholders := strings.TrimRight(strings.Repeat("?,", len(columns)), ",")
	query := fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (%s)", specsTable, strings.Join(columns, ","), placeholders)
	_, err := tx.Exec(query, args...)
	return err
}

// resolveModelKey normaliza a chave usada para reaproveitar especificações por modelo.
func resolveModelKey(product model.Product) string {
	if modelStr := strings.TrimSpace(product.Model); modelStr != "" {
		return modelStr
	}
	if modelStr := strings.TrimSpace(product.Specifications["modelVersion"]); modelStr != "" {
		return modelStr
	}
	return strings.TrimSpace(product.Name)
}

// GetSpecificationsBatch retorna um mapa model->specs para os modelos informados de um type.
// É uma consulta única na tabela de specs correspondente ao tipo.
func (r *SQLiteRepository) GetSpecificationsBatch(ctx context.Context, models []string, productType string) (map[string]map[string]string, error) {
	componentName := fmt.Sprintf("SQLiteRepository.GetSpecificationsBatch[%s]", productType)

	specsByModel := make(map[string]map[string]string, len(models))
	if len(models) == 0 {
		return specsByModel, nil
	}

	specsTable, err := r.lookupSpecsTable(ctx, productType)
	if err != nil {
		return nil, err
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(models)), ",")
	args := make([]any, len(models))
	for idx, modelKey := range models {
		args[idx] = modelKey
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", specsTable, specsModelColumn, placeholders)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}

	modelColumnIdx := -1
	for idx, col := range columns {
		if col == specsModelColumn {
			modelColumnIdx = idx
			break
		}
	}

	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		scanDest := make([]any, len(columns))
		for idx := range values {
			scanDest[idx] = &values[idx]
		}
		if err := rows.Scan(scanDest...); err != nil {
			return nil, customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
		}

		rowModel := values[modelColumnIdx].String
		specs := make(map[string]string, len(columns)-1)
		for colIdx, col := range columns {
			if colIdx != modelColumnIdx && values[colIdx].Valid {
				specs[col] = values[colIdx].String
			}
		}
		specsByModel[rowModel] = specs
	}
	if err := rows.Err(); err != nil {
		return nil, customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}
	return specsByModel, nil
}

func (r *SQLiteRepository) lookupSpecsTable(ctx context.Context, productType string) (string, error) {
	var specsTable string
	err := r.db.QueryRowContext(ctx,
		"SELECT specs_table FROM product_type_specs WHERE product_type = ?", productType,
	).Scan(&specsTable)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("unsupported product type: %s", productType)
	}
	return specsTable, err
}

func lookupSpecsTableTx(tx *sql.Tx, productType string) (string, error) {
	var specsTable string
	err := tx.QueryRow(
		"SELECT specs_table FROM product_type_specs WHERE product_type = ?", productType,
	).Scan(&specsTable)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("unsupported product type: %s", productType)
	}
	return specsTable, err
}
