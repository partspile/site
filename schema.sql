-- Schema for project.db

-- Make table
CREATE TABLE Make (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

-- Year table
CREATE TABLE Year (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL UNIQUE
);

-- Model table
CREATE TABLE Model (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
);

-- Engine table
CREATE TABLE Engine (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    description TEXT NOT NULL
);

-- Car table
CREATE TABLE Car (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    make_id INTEGER NOT NULL,
    year_id INTEGER NOT NULL,
    model_id INTEGER NOT NULL,
    FOREIGN KEY (make_id) REFERENCES Make(id),
    FOREIGN KEY (year_id) REFERENCES Year(id),
    FOREIGN KEY (model_id) REFERENCES Model(id),
    UNIQUE (make_id, year_id, model_id)
);

-- CarEngine table
CREATE TABLE CarEngine (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    car_id INTEGER NOT NULL,
    engine_id INTEGER NOT NULL,
    FOREIGN KEY (car_id) REFERENCES Car(id),
    FOREIGN KEY (engine_id) REFERENCES Engine(id),
    UNIQUE (car_id, engine_id)
);

-- PartCategory table
CREATE TABLE PartCategory (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

-- PartSubCategory table
CREATE TABLE PartSubCategory (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (category_id) REFERENCES PartCategory(id),
    UNIQUE (category_id, name)
);

-- Ad table
CREATE TABLE Ad (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    description TEXT,
    price REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    category_id INTEGER,
    subcategory_id INTEGER,
    FOREIGN KEY (category_id) REFERENCES PartCategory(id),
    FOREIGN KEY (subcategory_id) REFERENCES PartSubCategory(id)
);

-- AdCarEngine join table
CREATE TABLE AdCarEngine (
    ad_id INTEGER NOT NULL,
    car_engine_id INTEGER NOT NULL,
    FOREIGN KEY (ad_id) REFERENCES Ad(id),
    FOREIGN KEY (car_engine_id) REFERENCES CarEngine(id),
    PRIMARY KEY (ad_id, car_engine_id)
);

-- Indexes for efficient queries and pagination
CREATE INDEX idx_car_make_year_model ON Car(make_id, year_id, model_id);
CREATE INDEX idx_carengine_car_engine ON CarEngine(car_id, engine_id);
CREATE INDEX idx_ad_created_at_id ON Ad(created_at, id);
CREATE INDEX idx_adcarengine_ad_id ON AdCarEngine(ad_id);
CREATE INDEX idx_adcarengine_car_engine_id ON AdCarEngine(car_engine_id);
CREATE INDEX idx_ad_category_subcategory ON Ad(category_id, subcategory_id);
CREATE INDEX idx_partcategory_name ON PartCategory(name);
CREATE INDEX idx_partsubcategory_category_name ON PartSubCategory(category_id, name); 