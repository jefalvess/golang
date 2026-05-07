package repository

import (
	"context"
	"database/sql"
	"testing"

	"comparify/internal/model"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		require.NoError(t, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openTestDB(t)
	schema, err := LoadSchema("../../data/schema.sql")
	require.NoError(t, err)
	require.NoError(t, MigrateSchema(db, schema))
	return db
}

func TestMigrateSchema(t *testing.T) {
	db := openTestDB(t)

	err := MigrateSchema(db, `CREATE TABLE IF NOT EXISTS schema_test (id TEXT PRIMARY KEY);`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO schema_test (id) VALUES ('ok')`)
	require.NoError(t, err)
}

func seedTestProduct(t *testing.T, db *sql.DB, product model.Product) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO products (id, name, image_url, description, price, rating, size, weight, color, type, model) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		product.ID, product.Name, product.ImageURL, product.Description, product.Price,
		product.Rating, product.Size, product.Weight, product.Color, product.Type, product.Model,
	)
	if err != nil {
		t.Fatalf("seed product %s: %v", product.ID, err)
	}
}

func seedTestSmartphoneSpecs(t *testing.T, db *sql.DB, modelName, batteryCapacity, brand string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT OR IGNORE INTO smartphone_specs (model, battery_capacity, brand) VALUES (?, ?, ?)`,
		modelName, batteryCapacity, brand,
	)
	if err != nil {
		t.Fatalf("seed smartphone specs for model %s: %v", modelName, err)
	}
}

func TestNewSQLiteRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	if repo == nil {
		t.Fatal("esperava repositório não-nulo")
	}
}

func TestSQLiteRepository_ListByIDs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	seedTestProduct(t, db, model.Product{ID: "p1", Name: "Produto 1", Price: 1000, Type: "celular", Model: "m1"})
	seedTestProduct(t, db, model.Product{ID: "p2", Name: "Produto 2", Price: 2000, Type: "celular", Model: "m2"})
	tests := []struct {
		name    string
		ids     []string
		wantIDs []string
		wantErr error
	}{
		{"vazio", []string{}, nil, nil},
		{"dois produtos na ordem", []string{"p1", "p2"}, []string{"p1", "p2"}, nil},
		{"dois produtos ordem invertida", []string{"p2", "p1"}, []string{"p2", "p1"}, nil},
		{"produto inexistente", []string{"naoexiste"}, nil, ErrProductNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			products, err := repo.ListByIDs(context.Background(), tt.ids)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, products)
			} else {
				require.NoError(t, err)
				if tt.wantIDs != nil {
					ids := make([]string, len(products))
					for i, p := range products {
						ids[i] = p.ID
					}
					assert.Equal(t, tt.wantIDs, ids)
				} else {
					assert.Empty(t, products)
				}
			}
		})
	}
}

func TestSQLiteRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	seedTestProduct(t, db, model.Product{ID: "p1", Name: "Produto Teste", Price: 999.90, Type: "celular", Model: "m1"})
	tests := []struct {
		name    string
		id      string
		wantID  string
		wantErr error
	}{
		{"produto existe", "p1", "p1", nil},
		{"produto não existe", "naoexiste", "", ErrProductNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product, err := repo.GetByID(context.Background(), tt.id)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, product.ID)
			}
		})
	}
}

func TestSQLiteRepository_SeedSQLite(t *testing.T) {
	db := setupTestDB(t)
	products := []model.Product{{ID: "s1", Name: "Smartphone Seed", Price: 2500, Type: "celular", Model: "modelS1", Specifications: map[string]string{"batteryCapacity": "4000mAh", "brand": "TestBrand"}}}
	err := SeedSQLite(context.Background(), db, products)
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	product, err := repo.GetByID(context.Background(), "s1")
	require.NoError(t, err)
	assert.Equal(t, "Smartphone Seed", product.Name)
}

func TestSQLiteRepository_GetSpecificationsByModel(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	seedTestSmartphoneSpecs(t, db, "modelX", "5000mAh", "BrandX")
	tests := []struct {
		name      string
		model     string
		typeProd  string
		wantKey   string
		wantValue string
		wantEmpty bool
	}{
		{"modelo existente", "modelX", "celular", "battery_capacity", "5000mAh", false},
		{"modelo inexistente", "naoexiste", "celular", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specs, err := repo.GetSpecificationsByModel(context.Background(), tt.model, tt.typeProd)
			require.NoError(t, err)
			if tt.wantEmpty {
				assert.Empty(t, specs)
			} else {
				assert.Equal(t, tt.wantValue, specs[tt.wantKey])
			}
		})
	}
}

func TestSQLiteRepository_GetSpecificationsBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	seedTestSmartphoneSpecs(t, db, "modelA", "3000mAh", "BrandA")
	seedTestSmartphoneSpecs(t, db, "modelB", "4000mAh", "BrandB")
	tests := []struct {
		name      string
		models    []string
		typeProd  string
		wantLen   int
		wantKey   string
		wantValue string
		wantErr   bool
	}{
		{"batch vazio", []string{}, "celular", 0, "", "", false},
		{"batch sucesso", []string{"modelA", "modelB"}, "celular", 2, "modelA", "3000mAh", false},
		{"tipo não suportado", []string{"m1"}, "tipo-invalido", 0, "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetSpecificationsBatch(context.Background(), tt.models, tt.typeProd)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.wantLen)
				if tt.wantKey != "" && tt.wantValue != "" {
					assert.Equal(t, tt.wantValue, result[tt.wantKey]["battery_capacity"])
				}
			}
		})
	}
}

func TestSQLiteRepository_ResolveProductModel(t *testing.T) {
	tests := []struct {
		name     string
		input    model.Product
		expected string
	}{
		{"usa Model", model.Product{ID: "p1", Name: "Nome", Model: "  modelX  "}, "modelX"},
		{"usa modelVersion", model.Product{ID: "p1", Name: "Nome", Model: "", Specifications: map[string]string{"modelVersion": "versionY"}}, "versionY"},
		{"usa Name", model.Product{ID: "p1", Name: "  Produto Nome  ", Model: ""}, "Produto Nome"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveProductModel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
