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
    specs_table TEXT NOT NULL -- nome da tabela de especializações deste produto
);

-- Tabela unificada de marcas: elimina UNION entre tabelas de specs no filtro brand
CREATE TABLE IF NOT EXISTS product_brands (
    product_id TEXT PRIMARY KEY,
    brand TEXT,
    FOREIGN KEY (product_id) REFERENCES products (id)
);

-- Criação da tabela de especificações para smartphones
CREATE TABLE IF NOT EXISTS smartphone_specs (
    product_id TEXT PRIMARY KEY,
    battery_capacity TEXT,
    camera_specs TEXT,
    memory TEXT,
    storage_capacity TEXT,
    brand TEXT,
    model_version TEXT,
    operating_system TEXT,
    FOREIGN KEY (product_id) REFERENCES products (id)
);

-- Criação da tabela de especificações para frigideiras
CREATE TABLE IF NOT EXISTS fridge_specs (
    product_id TEXT PRIMARY KEY,
    capacity TEXT,
    energy_class TEXT,
    brand TEXT,
    model_version TEXT,
    FOREIGN KEY (product_id) REFERENCES products (id)
);

-- Criação da tabela de especificações para micro-ondas
CREATE TABLE IF NOT EXISTS microwave_specs (
    product_id TEXT PRIMARY KEY,
    capacity TEXT,
    power TEXT,
    brand TEXT,
    model_version TEXT,
    FOREIGN KEY (product_id) REFERENCES products (id)
);

-- Criação da tabela de especificações para alto-falantes
CREATE TABLE IF NOT EXISTS speaker_specs (
    product_id TEXT PRIMARY KEY,
    battery_capacity TEXT,
    connectivity TEXT,
    brand TEXT,
    FOREIGN KEY (product_id) REFERENCES products (id)
);

-- Índices para busca eficiente
CREATE INDEX IF NOT EXISTS idx_products_color ON products (color);
CREATE INDEX IF NOT EXISTS idx_products_type ON products (type);
CREATE INDEX IF NOT EXISTS idx_products_specs_table ON products (specs_table);

-- Índice de brand na tabela unificada
CREATE INDEX IF NOT EXISTS idx_product_brands_brand ON product_brands (brand);

-- Índices de brand nas tabelas de specs (mantidos para consultas diretas)
CREATE INDEX IF NOT EXISTS idx_smartphone_specs_brand ON smartphone_specs (brand);
CREATE INDEX IF NOT EXISTS idx_fridge_specs_brand ON fridge_specs (brand);
CREATE INDEX IF NOT EXISTS idx_microwave_specs_brand ON microwave_specs (brand);
CREATE INDEX IF NOT EXISTS idx_speaker_specs_brand ON speaker_specs (brand);
