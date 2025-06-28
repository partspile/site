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
    user_id INTEGER NOT NULL,
    click_count INTEGER DEFAULT 0,
    FOREIGN KEY (subcategory_id) REFERENCES PartSubCategory(id),
    FOREIGN KEY (user_id) REFERENCES User(id)
);

-- AdCar join table
CREATE TABLE AdCar (
    ad_id INTEGER NOT NULL,
    car_id INTEGER NOT NULL,
    FOREIGN KEY (ad_id) REFERENCES Ad(id),
    FOREIGN KEY (car_id) REFERENCES Car(id),
    PRIMARY KEY (ad_id, car_id)
);

-- FlaggedAd table for user-ad bookmarks
CREATE TABLE FlaggedAd (
    user_id INTEGER NOT NULL,
    ad_id INTEGER NOT NULL,
    flagged_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, ad_id),
    FOREIGN KEY (user_id) REFERENCES User(id),
    FOREIGN KEY (ad_id) REFERENCES Ad(id)
);
CREATE INDEX idx_flaggedad_user_id ON FlaggedAd(user_id);
CREATE INDEX idx_flaggedad_ad_id ON FlaggedAd(ad_id);

-- User table
CREATE TABLE User (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    phone TEXT NOT NULL UNIQUE,
    token_balance REAL NOT NULL DEFAULT 0.0,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_admin INTEGER NOT NULL DEFAULT 0
);

-- TokenTransaction table
CREATE TABLE TokenTransaction (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    type TEXT NOT NULL, -- e.g., 'ad_post', 'payout', 'purchase', 'transfer_in', 'transfer_out', 'cash_out', 'ad_click'
    amount REAL NOT NULL, -- positive or negative, number of tokens
    related_user_id INTEGER, -- nullable, for peer-to-peer transfers
    ad_id INTEGER, -- nullable, for ad-related transactions
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    description TEXT,
    user_deleted INTEGER DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES User(id),
    FOREIGN KEY (related_user_id) REFERENCES User(id),
    FOREIGN KEY (ad_id) REFERENCES Ad(id)
);

-- PayoutFund table (singleton row)
CREATE TABLE PayoutFund (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    balance REAL NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Archive tables for archived users and their data

-- ArchivedUser table - archived users
CREATE TABLE ArchivedUser (
    id INTEGER PRIMARY KEY,  -- Same ID as original user
    name TEXT NOT NULL,
    phone TEXT NOT NULL,
    token_balance REAL NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME,
    deletion_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_admin INTEGER NOT NULL DEFAULT 0
);

-- ArchivedAd table - archived ads
CREATE TABLE ArchivedAd (
    id INTEGER PRIMARY KEY,  -- Same ID as original ad
    description TEXT,
    price REAL,
    created_at DATETIME,
    subcategory_id INTEGER,
    user_id INTEGER NOT NULL,  -- Reference to ArchivedUser.id
    deletion_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (subcategory_id) REFERENCES PartSubCategory(id),
    FOREIGN KEY (user_id) REFERENCES ArchivedUser(id)
);

-- ArchivedAdCar table - archived ad-car relationships
CREATE TABLE ArchivedAdCar (
    ad_id INTEGER NOT NULL,
    car_id INTEGER NOT NULL,
    deletion_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (ad_id) REFERENCES ArchivedAd(id),
    FOREIGN KEY (car_id) REFERENCES Car(id),
    PRIMARY KEY (ad_id, car_id)
);

-- Add indexes for efficient querying of archived data
CREATE INDEX idx_archiveduser_deletion_date ON ArchivedUser(deletion_date);
CREATE INDEX idx_archivedad_user_id ON ArchivedAd(user_id);
CREATE INDEX idx_archivedad_deletion_date ON ArchivedAd(deletion_date);
CREATE INDEX idx_archivedadcar_ad_id ON ArchivedAdCar(ad_id);

-- Modify TokenTransaction to handle deleted users
CREATE INDEX idx_tokentransaction_user_deleted ON TokenTransaction(user_deleted);

-- Indexes for efficient queries and pagination
CREATE INDEX idx_car_make_year_model_engine ON Car(make_id, year_id, model_id, engine_id);
CREATE INDEX idx_ad_created_at_id ON Ad(created_at, id);
CREATE INDEX idx_adcar_ad_id ON AdCar(ad_id);
CREATE INDEX idx_adcar_car_id ON AdCar(car_id);
CREATE INDEX idx_partcategory_name ON PartCategory(name);
CREATE INDEX idx_partsubcategory_category_name ON PartSubCategory(category_id, name);

-- Create AdClick table for per-user ad click tracking
CREATE TABLE IF NOT EXISTS AdClick (
    ad_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    click_count INTEGER DEFAULT 0,
    PRIMARY KEY (ad_id, user_id)
); 