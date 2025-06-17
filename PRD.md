# Product Requirements Document (PRD)

## Project: Parts Pile

---

## 1. Overview

Parts Pile is a web-based platform for listing, searching, and managing automotive parts ads. It provides a structured, vehicle-centric approach to cataloging parts, allowing users to filter and search by make, year, model, engine, category, and subcategory. The system is backed by a normalized vehicle/parts database and supports CRUD operations for ads, as well as a rich vehicle data model. The platform serves both sellers (listing parts) and buyers (searching for parts).

---

## 2. Goals & Objectives

- Enable users to list and manage ads for automotive parts with detailed vehicle fitment data.
- Provide powerful, structured search and filtering by vehicle (make, year, model, engine) and part (category, subcategory).
- Support a modern, responsive UI for ad creation, editing, and browsing.
- Maintain a comprehensive, normalized database of vehicles and parts.
- Allow for future extensibility (e.g., more part attributes, user accounts, messaging).

---

## 3. Features

### 3.1 Ad Management
- Users can create, edit, and delete ads for automotive parts.
- Each ad includes: description, price, vehicle fitment (make, year(s), model(s), engine(s)), part category, and subcategory.
- Ads are timestamped.

### 3.2 Vehicle Data Integration
- The system maintains a comprehensive, normalized vehicle database (make, year, model, engine) to support accurate fitment and filtering.

### 3.3 Part Categorization
- Parts are organized by category and subcategory (e.g., "Electrical" > "Alternator").
- Categories and subcategories are stored in the database and can be extended.

### 3.4 Search & Filtering
- Users can search ads by free text, which is parsed by a Large Language Model (LLM) into a structured query (e.g., SQL or equivalent), enabling flexible, natural language search with accurate filtering by vehicle and part attributes.
- Search supports cursor-based infinite scroll/pagination.

### 3.5 API Endpoints
- RESTful endpoints for CRUD operations on ads and for fetching vehicle/part data for dynamic forms.

### 3.6 Modern UI/UX
- Modern, accessible web UI using Tailwind CSS and HTMX for dynamic updates.
- Form validation and user feedback for all actions.

---

## 4. Technology Stack

The platform is built with the following technologies:

- **Backend:** Written in Go, providing performance, reliability, and maintainability.
- **HTML Generation:** Uses Gomponents for type-safe, composable UI components in Go.
- **Frontend:** Tailwind CSS and HTMX for responsive, dynamic user interfaces.

---

## 5. User Stories

- As a seller, I want to create a new ad for a part, specifying the exact vehicles it fits, so buyers can find it easily.
- As a buyer, I want to search for parts by my car's make, year, model, and engine, so I only see relevant ads.
- As a user, I want to browse categories and subcategories to discover available parts.
- As a seller, I want to edit or delete my ads if details change or the part is sold.
- As a user, I want fast, accurate search results and a modern, easy-to-use interface.

---

## 6. Data Model (Simplified)

- **Make**: id, name
- **Year**: id, year
- **Model**: id, name
- **Engine**: id, name
- **Car**: id, make_id, year_id, model_id, engine_id
- **PartCategory**: id, name
- **PartSubCategory**: id, category_id, name
- **Ad**: id, description, price, created_at, subcategory_id
- **AdCar**: ad_id, car_id

See `schema.sql` for full schema and indexes.

---

## 7. API & Endpoints

- `GET /` — Home/search page
- `GET /new-ad` — New ad form
- `GET /edit-ad/{id}` — Edit ad form
- `GET /ad/{id}` — View ad details
- `GET /search` — Search ads (supports query and cursor for pagination)
- `GET /api/makes` — List all makes
- `GET /api/years?make=...` — List years for a make
- `GET /api/models?make=...&years=...` — List models for make/years
- `GET /api/engines?make=...&years=...&models=...` — List engines for make/years/models
- `POST /api/new-ad` — Create new ad
- `POST /api/update-ad` — Update ad
- `DELETE /delete-ad/{id}` — Delete ad

---

## 8. Non-Functional Requirements

- **Performance**: Fast search and filtering, efficient DB queries, indexed tables.
- **Scalability**: Designed for extensibility (e.g., user accounts, more part attributes).
- **Security**: Input validation, confirmation for destructive actions.
- **Reliability**: Transactional DB operations for ad CRUD.
- **Maintainability**: Modular Go codebase, clear separation of concerns.
- **Accessibility**: Responsive, accessible UI components.

---

## 9. Licensing

- BSD 3-Clause License (see LICENSE file)

---

## 10. Open Questions / Future Work

- User authentication and account management
- Messaging between buyers and sellers
- Image uploads for ads
- Advanced analytics and reporting
- Internationalization/localization
- Admin dashboard for moderation

--- 