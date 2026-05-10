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

-- Criação da tabela de especificações para smartphones
CREATE TABLE IF NOT EXISTS smartphone_specs (
    model TEXT PRIMARY KEY,
    model_version TEXT,
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

