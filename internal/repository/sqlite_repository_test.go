package repository

import (
	"context"
	"database/sql"
	"testing"

	"comparify/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

const testSchema = `
CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    image_url TEXT,
    description TEXT,
    price REAL NOT NULL,
    rating REAL,
    size TEXT,
    weight TEXT,
    color TEXT,
    type TEXT NOT NULL,
    model TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS product_type_specs (
    product_type TEXT PRIMARY KEY,
    specs_table TEXT NOT NULL UNIQUE
);
INSERT OR IGNORE INTO product_type_specs (product_type, specs_table) VALUES
    ('celular', 'smartphone_specs'),
    ('geladeira', 'fridge_specs'),
    ('micro-ondas', 'microwave_specs'),
    ('caixa de som', 'speaker_specs');
CREATE TABLE IF NOT EXISTS smartphone_specs (
    model TEXT PRIMARY KEY,
    battery_capacity TEXT,
    camera_specs TEXT,
    memory TEXT,
    storage_capacity TEXT,
    brand TEXT,
    operating_system TEXT
);
CREATE TABLE IF NOT EXISTS fridge_specs (
    model TEXT PRIMARY KEY,
    capacity TEXT,
    energy_class TEXT,
    brand TEXT
);
CREATE TABLE IF NOT EXISTS microwave_specs (
    model TEXT PRIMARY KEY,
    capacity TEXT,
    power TEXT,
    brand TEXT
);
CREATE TABLE IF NOT EXISTS speaker_specs (
    model TEXT PRIMARY KEY,
    battery_capacity TEXT,
    connectivity TEXT,
    brand TEXT
);`

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(testSchema); err != nil {
		db.Close()
		t.Fatalf("setup schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
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

func TestListByIDs_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	products, err := repo.ListByIDs(context.Background(), []string{})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("esperava lista vazia, obteve %d produtos", len(products))
	}
}

func TestListByIDs_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	seedTestProduct(t, db, model.Product{ID: "p1", Name: "Produto 1", Price: 1000, Type: "celular", Model: "m1"})
	seedTestProduct(t, db, model.Product{ID: "p2", Name: "Produto 2", Price: 2000, Type: "celular", Model: "m2"})

	products, err := repo.ListByIDs(context.Background(), []string{"p1", "p2"})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("esperava 2 produtos, obteve %d", len(products))
	}
	if products[0].ID != "p1" || products[1].ID != "p2" {
		t.Errorf("ordem dos produtos não preservada: %v", []string{products[0].ID, products[1].ID})
	}
}

func TestListByIDs_OrderPreserved(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	seedTestProduct(t, db, model.Product{ID: "p1", Name: "Produto 1", Price: 1000, Type: "celular", Model: "m1"})
	seedTestProduct(t, db, model.Product{ID: "p2", Name: "Produto 2", Price: 2000, Type: "celular", Model: "m2"})

	// Solicitar na ordem inversa
	products, err := repo.ListByIDs(context.Background(), []string{"p2", "p1"})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if products[0].ID != "p2" || products[1].ID != "p1" {
		t.Errorf("esperava ordem p2,p1 mas obteve: %s,%s", products[0].ID, products[1].ID)
	}
}

func TestListByIDs_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	_, err := repo.ListByIDs(context.Background(), []string{"naoexiste"})
	if err == nil {
		t.Fatal("esperava erro para produto inexistente")
	}
	if err != ErrProductNotFound {
		t.Errorf("esperava ErrProductNotFound, obteve: %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	seedTestProduct(t, db, model.Product{ID: "p1", Name: "Produto Teste", Price: 999.90, Type: "celular", Model: "m1"})

	product, err := repo.GetByID(context.Background(), "p1")
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if product.ID != "p1" || product.Name != "Produto Teste" {
		t.Errorf("produto retornado incorreto: %+v", product)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	_, err := repo.GetByID(context.Background(), "naoexiste")
	if err == nil {
		t.Fatal("esperava erro para produto inexistente")
	}
	if err != ErrProductNotFound {
		t.Errorf("esperava ErrProductNotFound, obteve: %v", err)
	}
}

func TestSeedSQLite_Success(t *testing.T) {
	db := setupTestDB(t)

	products := []model.Product{
		{
			ID:    "s1",
			Name:  "Smartphone Seed",
			Price: 2500,
			Type:  "celular",
			Model: "modelS1",
			Specifications: map[string]string{
				"batteryCapacity": "4000mAh",
				"brand":           "TestBrand",
			},
		},
	}

	if err := SeedSQLite(context.Background(), db, products); err != nil {
		t.Fatalf("esperava seed bem-sucedido, obteve erro: %v", err)
	}

	repo := NewSQLiteRepository(db)
	product, err := repo.GetByID(context.Background(), "s1")
	if err != nil {
		t.Fatalf("produto seedado não encontrado: %v", err)
	}
	if product.Name != "Smartphone Seed" {
		t.Errorf("nome incorreto: %s", product.Name)
	}
}

func TestGetSpecificationsByModel_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	seedTestSmartphoneSpecs(t, db, "modelX", "5000mAh", "BrandX")

	specs, err := repo.GetSpecificationsByModel(context.Background(), "modelX", "celular")
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if specs["battery_capacity"] != "5000mAh" {
		t.Errorf("esperava battery_capacity=5000mAh, obteve: %v", specs["battery_capacity"])
	}
}

func TestGetSpecificationsByModel_ModelNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	specs, err := repo.GetSpecificationsByModel(context.Background(), "naoexiste", "celular")
	if err != nil {
		t.Fatalf("esperava mapa vazio sem erro, obteve: %v", err)
	}
	if len(specs) != 0 {
		t.Errorf("esperava mapa vazio, obteve: %v", specs)
	}
}

func TestGetSpecificationsBatch_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	result, err := repo.GetSpecificationsBatch(context.Background(), []string{}, "celular")
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("esperava mapa vazio, obteve: %v", result)
	}
}

func TestGetSpecificationsBatch_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	seedTestSmartphoneSpecs(t, db, "modelA", "3000mAh", "BrandA")
	seedTestSmartphoneSpecs(t, db, "modelB", "4000mAh", "BrandB")

	result, err := repo.GetSpecificationsBatch(context.Background(), []string{"modelA", "modelB"}, "celular")
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("esperava 2 modelos, obteve %d", len(result))
	}
	if result["modelA"]["battery_capacity"] != "3000mAh" {
		t.Errorf("specs de modelA incorretas: %v", result["modelA"])
	}
}

func TestGetSpecificationsBatch_UnsupportedType(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	_, err := repo.GetSpecificationsBatch(context.Background(), []string{"m1"}, "tipo-invalido")
	if err == nil {
		t.Fatal("esperava erro para tipo de produto não suportado")
	}
}

func TestResolveProductModel_FromModel(t *testing.T) {
	product := model.Product{ID: "p1", Name: "Nome", Model: "  modelX  "}
	result := resolveProductModel(product)
	if result != "modelX" {
		t.Errorf("esperava 'modelX', obteve: %s", result)
	}
}

func TestResolveProductModel_FromModelVersion(t *testing.T) {
	product := model.Product{
		ID:    "p1",
		Name:  "Nome",
		Model: "",
		Specifications: map[string]string{
			"modelVersion": "versionY",
		},
	}
	result := resolveProductModel(product)
	if result != "versionY" {
		t.Errorf("esperava 'versionY', obteve: %s", result)
	}
}

func TestResolveProductModel_FromName(t *testing.T) {
	product := model.Product{ID: "p1", Name: "  Produto Nome  ", Model: ""}
	result := resolveProductModel(product)
	if result != "Produto Nome" {
		t.Errorf("esperava 'Produto Nome', obteve: %s", result)
	}
}
