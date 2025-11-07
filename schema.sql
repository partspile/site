-- Schema for project.db

-- Ad Category table
CREATE TABLE AdCategory (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE CHECK (length(name) > 0)
);

-- Specification field metadata tables
CREATE TABLE SpecificationField (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE CHECK (length(name) > 0),
    display_name TEXT NOT NULL,
    field_type TEXT NOT NULL CHECK (field_type IN ('string', 'int', 'float64')),
    table_name TEXT NOT NULL,
    table_field_name TEXT NOT NULL,
    qdrant_type TEXT NOT NULL CHECK (qdrant_type IN ('keyword', 'keyword_array', 'integer', 'float', 'geo'))
);

CREATE TABLE AdCategoryField (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    specification_field_id INTEGER NOT NULL REFERENCES SpecificationField(id),
    priority_order INTEGER NOT NULL,
    next_in_chain INTEGER NOT NULL DEFAULT 0,
    is_required INTEGER NOT NULL DEFAULT 1,
    UNIQUE (ad_category_id, specification_field_id)
);
CREATE INDEX idx_adcategoryfield_category ON AdCategoryField(ad_category_id);
CREATE INDEX idx_adcategoryfield_order ON AdCategoryField(ad_category_id, priority_order);

-- Specification tables
CREATE TABLE Make (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    name TEXT NOT NULL CHECK (length(name) > 0),
    parent_company_id INTEGER REFERENCES ParentCompany(id)
);

CREATE TABLE ParentCompany (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE CHECK (length(name) > 0),
    country TEXT
);

CREATE TABLE Year (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    year INTEGER NOT NULL
);

CREATE TABLE Model (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    name TEXT NOT NULL CHECK (length(name) > 0)
);

CREATE TABLE Engine (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    name TEXT NOT NULL CHECK (length(name) > 0)
);

CREATE TABLE Breed (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    name TEXT NOT NULL CHECK (length(name) > 0)
);

CREATE TABLE PartCategory (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    name TEXT NOT NULL CHECK (length(name) > 0)
);

CREATE TABLE PartSubCategory (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    name TEXT NOT NULL CHECK (length(name) > 0)
);

CREATE TABLE Specification (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    make_id INTEGER REFERENCES Make(id),
    year_id INTEGER REFERENCES Year(id),
    model_id INTEGER REFERENCES Model(id),
    engine_id INTEGER REFERENCES Engine(id),
    breed_id INTEGER REFERENCES Breed(id),
    part_category_id INTEGER REFERENCES PartCategory(id),
    part_subcategory_id INTEGER REFERENCES PartSubCategory(id),
    UNIQUE (ad_category_id, make_id, year_id, model_id, engine_id, part_category_id, part_subcategory_id, breed_id)
);

-- Location table
CREATE TABLE Location (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    raw_text TEXT UNIQUE NOT NULL CHECK (length(raw_text) > 0),
    city TEXT NOT NULL,
    admin_area TEXT NOT NULL,
    country TEXT NOT NULL,
    latitude REAL NOT NULL,
    longitude REAL NOT NULL
);
CREATE INDEX idx_location_raw_text ON Location(raw_text);

-- Phone verification table
CREATE TABLE PhoneVerification (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phone TEXT NOT NULL,
    verification_code TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    registration_name TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_phoneverification_phone ON PhoneVerification(phone);
CREATE INDEX idx_phoneverification_expires ON PhoneVerification(expires_at);

-- User tables
CREATE TABLE User (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE CHECK (length(name) > 0),
    phone TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    password_salt TEXT NOT NULL,
    password_algo TEXT NOT NULL DEFAULT 'argon2id',
    phone_verified INTEGER NOT NULL DEFAULT 0,
    verification_code TEXT NOT NULL DEFAULT '',
    notification_method TEXT NOT NULL DEFAULT 'sms',
    email_address TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_admin INTEGER NOT NULL DEFAULT 0,
    sms_opted_out INTEGER NOT NULL DEFAULT 0,
    deleted_at DATETIME
);
CREATE INDEX idx_user_deleted_at ON User(deleted_at);

-- Ad tables
CREATE TABLE Ad (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    price REAL NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    user_id INTEGER NOT NULL REFERENCES User(id),
    image_count INTEGER DEFAULT 0,
    location_id INTEGER REFERENCES Location(id),
    click_count INTEGER DEFAULT 0,
    last_clicked_at DATETIME,
    has_vector INTEGER DEFAULT 0
);
CREATE INDEX idx_ad_created_at_id ON Ad(created_at, id);
CREATE INDEX idx_ad_deleted_at ON Ad(deleted_at);
CREATE INDEX idx_ad_ad_category_id ON Ad(ad_category_id);

CREATE TABLE AdSpecification (
    ad_id INTEGER NOT NULL REFERENCES Ad(id),
    specification_id INTEGER NOT NULL REFERENCES Specification(id),
    PRIMARY KEY (ad_id, specification_id)
);
CREATE INDEX idx_adspecification_specification_id ON AdSpecification(specification_id);
CREATE INDEX idx_adspecification_ad_id ON AdSpecification(ad_id);

CREATE TABLE BookmarkedAd (
    user_id INTEGER NOT NULL REFERENCES User(id),
    ad_id INTEGER NOT NULL REFERENCES Ad(id),
    bookmarked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, ad_id)
);
CREATE INDEX idx_bookmarkedad_user_id ON BookmarkedAd(user_id);
CREATE INDEX idx_bookmarkedad_ad_id ON BookmarkedAd(ad_id);

CREATE TABLE UserAdClick (
    ad_id INTEGER NOT NULL REFERENCES Ad(id),
    user_id INTEGER NOT NULL REFERENCES User(id),
    click_count INTEGER DEFAULT 0,
    last_clicked_at DATETIME,
    PRIMARY KEY (ad_id, user_id)
);

CREATE TABLE UserSearch (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES User(id),
    query_string TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_usersearch_user_id ON UserSearch(user_id);
CREATE INDEX idx_usersearch_created_at ON UserSearch(created_at);

-- Messaging system tables
CREATE TABLE Conversation (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user1_id INTEGER NOT NULL REFERENCES User(id),
    user2_id INTEGER NOT NULL REFERENCES User(id),
    ad_id INTEGER NOT NULL REFERENCES Ad(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    user1_read BOOLEAN DEFAULT FALSE,
    user2_read BOOLEAN DEFAULT FALSE,
    UNIQUE (user1_id, user2_id, ad_id)
);
CREATE INDEX idx_conversation_user1_id ON Conversation(user1_id);
CREATE INDEX idx_conversation_user2_id ON Conversation(user2_id);
CREATE INDEX idx_conversation_ad_id ON Conversation(ad_id);
CREATE INDEX idx_conversation_updated_at ON Conversation(updated_at);

CREATE TABLE Message (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER NOT NULL REFERENCES Conversation(id),
    sender_id INTEGER NOT NULL REFERENCES User(id),
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    read_at DATETIME
);
CREATE INDEX idx_message_conversation_id ON Message(conversation_id);
CREATE INDEX idx_message_sender_id ON Message(sender_id);
CREATE INDEX idx_message_created_at ON Message(created_at);
CREATE INDEX idx_message_read_at ON Message(read_at);

-- Rock system tables
CREATE TABLE UserRock (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES User(id),
    rock_count INTEGER NOT NULL DEFAULT 3,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id)
);
CREATE INDEX idx_userrock_user_id ON UserRock(user_id);

CREATE TABLE AdRock (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_id INTEGER NOT NULL REFERENCES Ad(id),
    thrower_id INTEGER NOT NULL REFERENCES User(id),
    conversation_id INTEGER NOT NULL REFERENCES Conversation(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    resolved_at DATETIME,
    resolved_by INTEGER REFERENCES User(id)
);
CREATE INDEX idx_adrock_ad_id ON AdRock(ad_id);
CREATE INDEX idx_adrock_thrower_id ON AdRock(thrower_id);
CREATE INDEX idx_adrock_conversation_id ON AdRock(conversation_id);
CREATE INDEX idx_adrock_resolved_at ON AdRock(resolved_at);