-- Schema for project.db

-- Vehicle tables
CREATE TABLE Make (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    parent_company_id INTEGER REFERENCES ParentCompany(id)
);

CREATE TABLE ParentCompany (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    country TEXT
);

CREATE TABLE Year (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL UNIQUE
);

CREATE TABLE Model (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
);

CREATE TABLE Engine (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
);

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
CREATE INDEX idx_car_make_year_model_engine ON Car(make_id, year_id, model_id, engine_id);

-- Part tables
CREATE TABLE PartCategory (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);
CREATE INDEX idx_partcategory_name ON PartCategory(name);

CREATE TABLE PartSubCategory (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (category_id) REFERENCES PartCategory(id),
    UNIQUE (category_id, name)
);
CREATE INDEX idx_partsubcategory_category_name ON PartSubCategory(category_id, name);

-- Location table
CREATE TABLE Location (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    raw_text TEXT UNIQUE NOT NULL,
    city TEXT,
    admin_area TEXT,
    country TEXT,
    latitude REAL,
    longitude REAL
);

-- Phone verification table
CREATE TABLE PhoneVerification (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phone TEXT NOT NULL,
    verification_code TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_phoneverification_phone ON PhoneVerification(phone);
CREATE INDEX idx_phoneverification_expires ON PhoneVerification(expires_at);

-- User tables
CREATE TABLE User (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    phone TEXT NOT NULL UNIQUE,
    token_balance REAL NOT NULL DEFAULT 0.0,
    password_hash TEXT NOT NULL,
    password_salt TEXT NOT NULL,
    password_algo TEXT NOT NULL DEFAULT 'argon2id',
    phone_verified INTEGER NOT NULL DEFAULT 0,
    verification_code TEXT,
    notification_method TEXT NOT NULL DEFAULT 'sms',
    email_address TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_admin INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE ArchivedUser (
    id INTEGER PRIMARY KEY,  -- Same ID as original user
    name TEXT NOT NULL,
    phone TEXT NOT NULL,
    token_balance REAL NOT NULL,
    password_hash TEXT NOT NULL,
    password_salt TEXT NOT NULL,
    password_algo TEXT NOT NULL DEFAULT 'argon2id',
    notification_method TEXT NOT NULL DEFAULT 'sms',
    email_address TEXT,
    created_at DATETIME,
    deletion_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_admin INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_archiveduser_deletion_date ON ArchivedUser(deletion_date);

-- Ad tables
CREATE TABLE Ad (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT,
    description TEXT,
    price REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    subcategory_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    image_order TEXT,
    location_id INTEGER REFERENCES Location(id),
    click_count INTEGER DEFAULT 0,
    last_clicked_at DATETIME,
    has_vector INTEGER DEFAULT 0,
    FOREIGN KEY (subcategory_id) REFERENCES PartSubCategory(id),
    FOREIGN KEY (user_id) REFERENCES User(id)
);
CREATE INDEX idx_ad_created_at_id ON Ad(created_at, id);
CREATE INDEX idx_ad_deleted_at ON Ad(deleted_at);

CREATE TABLE AdCar (
    ad_id INTEGER NOT NULL,
    car_id INTEGER NOT NULL,
    FOREIGN KEY (ad_id) REFERENCES Ad(id),
    FOREIGN KEY (car_id) REFERENCES Car(id),
    PRIMARY KEY (ad_id, car_id)
);
CREATE INDEX idx_adcar_car_id ON AdCar(car_id);
CREATE INDEX idx_adcar_ad_id ON AdCar(ad_id);

CREATE TABLE BookmarkedAd (
    user_id INTEGER NOT NULL,
    ad_id INTEGER NOT NULL,
    bookmarked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, ad_id),
    FOREIGN KEY (user_id) REFERENCES User(id),
    FOREIGN KEY (ad_id) REFERENCES Ad(id)
);
CREATE INDEX idx_bookmarkedad_user_id ON BookmarkedAd(user_id);
CREATE INDEX idx_bookmarkedad_ad_id ON BookmarkedAd(ad_id);

CREATE TABLE UserAdClick (
    ad_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    click_count INTEGER DEFAULT 0,
    last_clicked_at DATETIME,
    PRIMARY KEY (ad_id, user_id),
    FOREIGN KEY (ad_id) REFERENCES Ad(id),
    FOREIGN KEY (user_id) REFERENCES User(id)
);

CREATE TABLE UserSearch (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    query_string TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES User(id)
);
CREATE INDEX idx_usersearch_user_id ON UserSearch(user_id);
CREATE INDEX idx_usersearch_created_at ON UserSearch(created_at);

-- Token/transaction tables
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
CREATE INDEX idx_tokentransaction_user_deleted ON TokenTransaction(user_deleted);

CREATE TABLE PayoutFund (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    balance REAL NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User embedding table for personalized vector search
CREATE TABLE UserEmbedding (
    user_id INTEGER PRIMARY KEY,
    embedding BLOB NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES User(id)
);

-- Messaging system tables
CREATE TABLE Conversation (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user1_id INTEGER NOT NULL,
    user2_id INTEGER NOT NULL,
    ad_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    user1_read BOOLEAN DEFAULT FALSE,
    user2_read BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (user1_id) REFERENCES User(id),
    FOREIGN KEY (user2_id) REFERENCES User(id),
    FOREIGN KEY (ad_id) REFERENCES Ad(id),
    UNIQUE (user1_id, user2_id, ad_id)
);
CREATE INDEX idx_conversation_user1_id ON Conversation(user1_id);
CREATE INDEX idx_conversation_user2_id ON Conversation(user2_id);
CREATE INDEX idx_conversation_ad_id ON Conversation(ad_id);
CREATE INDEX idx_conversation_updated_at ON Conversation(updated_at);

CREATE TABLE Message (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER NOT NULL,
    sender_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    read_at DATETIME,
    FOREIGN KEY (conversation_id) REFERENCES Conversation(id),
    FOREIGN KEY (sender_id) REFERENCES User(id)
);
CREATE INDEX idx_message_conversation_id ON Message(conversation_id);
CREATE INDEX idx_message_sender_id ON Message(sender_id);
CREATE INDEX idx_message_created_at ON Message(created_at);
CREATE INDEX idx_message_read_at ON Message(read_at);