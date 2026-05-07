-- Criação da tabela de produtos
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

-- Metadados de tipos: o banco define em qual tabela cada tipo guarda suas specs.
CREATE TABLE IF NOT EXISTS product_type_specs (
    product_type TEXT PRIMARY KEY,
    specs_table TEXT NOT NULL UNIQUE
);

INSERT OR IGNORE INTO product_type_specs (product_type, specs_table) VALUES
    ('celular', 'smartphone_specs'),
    ('geladeira', 'fridge_specs'),
    ('micro-ondas', 'microwave_specs'),
    ('caixa de som', 'speaker_specs');

-- Tabela unificada de marcas: elimina UNION entre tabelas de specs no filtro brand
CREATE TABLE IF NOT EXISTS product_brands (
    product_id TEXT PRIMARY KEY,
    brand TEXT,
    FOREIGN KEY (product_id) REFERENCES products (id)
);

-- Criação da tabela de especificações para smartphones
CREATE TABLE IF NOT EXISTS smartphone_specs (
    model TEXT PRIMARY KEY,
    battery_capacity TEXT,
    camera_specs TEXT,
    memory TEXT,
    storage_capacity TEXT,
    brand TEXT,
    operating_system TEXT
);

-- Criação da tabela de especificações para frigideiras
CREATE TABLE IF NOT EXISTS fridge_specs (
    model TEXT PRIMARY KEY,
    capacity TEXT,
    energy_class TEXT,
    brand TEXT
);

-- Criação da tabela de especificações para micro-ondas
CREATE TABLE IF NOT EXISTS microwave_specs (
    model TEXT PRIMARY KEY,
    capacity TEXT,
    power TEXT,
    brand TEXT
);

-- Criação da tabela de especificações para alto-falantes
CREATE TABLE IF NOT EXISTS speaker_specs (
    model TEXT PRIMARY KEY,
    battery_capacity TEXT,
    connectivity TEXT,
    brand TEXT
);

-- Índices para busca eficiente
CREATE INDEX IF NOT EXISTS idx_products_color ON products (color);
CREATE INDEX IF NOT EXISTS idx_products_type ON products (type);
CREATE INDEX IF NOT EXISTS idx_products_model ON products (model);
CREATE INDEX IF NOT EXISTS idx_product_type_specs_table ON product_type_specs (specs_table);

-- Índice composto para buscas multi-filtro (type + color)
CREATE INDEX IF NOT EXISTS idx_products_type_color ON products (type, color);

-- Índice de brand na tabela unificada
CREATE INDEX IF NOT EXISTS idx_product_brands_brand ON product_brands (brand);

-- Índices de brand nas tabelas de specs (mantidos para consultas diretas)
CREATE INDEX IF NOT EXISTS idx_smartphone_specs_brand ON smartphone_specs (brand);
CREATE INDEX IF NOT EXISTS idx_fridge_specs_brand ON fridge_specs (brand);
CREATE INDEX IF NOT EXISTS idx_microwave_specs_brand ON microwave_specs (brand);
CREATE INDEX IF NOT EXISTS idx_speaker_specs_brand ON speaker_specs (brand);
