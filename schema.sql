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
    name TEXT NOT NULL
);

-- Car table
CREATE TABLE Car (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    make_id INTEGER NOT NULL,
    year_id INTEGER NOT NULL,
    model_id INTEGER NOT NULL,
    engine_id INTEGER NOT NULL,
    FOREIGN KEY (make_id) REFERENCES Make(id),
    FOREIGN KEY (year_id) REFERENCES Year(id),
    FOREIGN KEY (model_id) REFERENCES Model(id),
    FOREIGN KEY (engine_id) REFERENCES Engine(id),
    UNIQUE (make_id, year_id, model_id, engine_id)
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
    subcategory_id INTEGER,
    FOREIGN KEY (subcategory_id) REFERENCES PartSubCategory(id)
);

-- AdCar join table
CREATE TABLE AdCar (
    ad_id INTEGER NOT NULL,
    car_id INTEGER NOT NULL,
    FOREIGN KEY (ad_id) REFERENCES Ad(id),
    FOREIGN KEY (car_id) REFERENCES Car(id),
    PRIMARY KEY (ad_id, car_id)
);

-- Indexes for efficient queries and pagination
CREATE INDEX idx_car_make_year_model_engine ON Car(make_id, year_id, model_id, engine_id);
CREATE INDEX idx_ad_created_at_id ON Ad(created_at, id);
CREATE INDEX idx_adcar_ad_id ON AdCar(ad_id);
CREATE INDEX idx_adcar_car_id ON AdCar(car_id);
CREATE INDEX idx_partcategory_name ON PartCategory(name);
CREATE INDEX idx_partsubcategory_category_name ON PartSubCategory(category_id, name); 