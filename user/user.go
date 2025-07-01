package user

import (
	"database/sql"
	"time"
)

// Table name constants
const (
	TableUser         = "User"
	TableArchivedUser = "ArchivedUser"
)

// UserStatus represents the status of a user
type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusArchived UserStatus = "archived"
)

type User struct {
	ID           int
	Name         string
	Phone        string
	TokenBalance float64
	PasswordHash string
	CreatedAt    time.Time
	IsAdmin      bool
	DeletionDate *time.Time `json:"deletion_date,omitempty"`
}

// IsArchived returns true if the user has been archived
func (u User) IsArchived() bool {
	return u.DeletionDate != nil
}

var db *sql.DB

func InitDB(database *sql.DB) {
	db = database
}

// CreateUser inserts a new user into the database
func CreateUser(name, phone, passwordHash string) (int, error) {
	res, err := db.Exec(`INSERT INTO User (name, phone, password_hash) VALUES (?, ?, ?)`, name, phone, passwordHash)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// GetUserByID retrieves a user by ID from either active or archived tables
// Returns the user, its status, and whether it was found
func GetUserByID(id int) (User, UserStatus, bool) {
	// Try active users first
	user, err := GetUser(id)
	if err == nil {
		return user, StatusActive, true
	}

	// Try archived users
	archivedUser, ok := GetArchivedUser(id)
	if ok {
		return archivedUser, StatusArchived, true
	}

	return User{}, StatusActive, false
}

// GetUserByPhone retrieves a user by phone number
func GetUserByPhone(phone string) (User, error) {
	row := db.QueryRow(`SELECT id, name, phone, token_balance, password_hash, created_at, is_admin FROM User WHERE phone = ?`, phone)
	var u User
	var createdAt string
	var isAdmin int
	err := row.Scan(&u.ID, &u.Name, &u.Phone, &u.TokenBalance, &u.PasswordHash, &createdAt, &isAdmin)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	u.IsAdmin = isAdmin == 1
	return u, nil
}

// GetUser retrieves a user by ID (active users only)
func GetUser(id int) (User, error) {
	row := db.QueryRow(`SELECT id, name, phone, token_balance, password_hash, created_at, is_admin FROM User WHERE id = ?`, id)
	var u User
	var createdAt string
	var isAdmin int
	err := row.Scan(&u.ID, &u.Name, &u.Phone, &u.TokenBalance, &u.PasswordHash, &createdAt, &isAdmin)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	u.IsAdmin = isAdmin == 1
	return u, nil
}

// GetArchivedUser retrieves an archived user by ID
func GetArchivedUser(id int) (User, bool) {
	row := db.QueryRow(`SELECT id, name, phone, token_balance, password_hash, created_at, is_admin, deletion_date FROM ArchivedUser WHERE id = ?`, id)
	var u User
	var createdAt, deletionDate string
	var isAdmin int
	err := row.Scan(&u.ID, &u.Name, &u.Phone, &u.TokenBalance, &u.PasswordHash, &createdAt, &isAdmin, &deletionDate)
	if err != nil {
		return User{}, false
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	u.IsAdmin = isAdmin == 1

	// Parse deletion date
	if parsedTime, err := time.Parse(time.RFC3339Nano, deletionDate); err == nil {
		u.DeletionDate = &parsedTime
	}

	return u, true
}

// UpdateTokenBalance updates a user's token balance

// GetUserByName retrieves a user by name (username)
func GetUserByName(name string) (User, error) {
	row := db.QueryRow(`SELECT id, name, phone, token_balance, password_hash, created_at, is_admin FROM User WHERE name = ?`, name)
	var u User
	var createdAt string
	var isAdmin int
	err := row.Scan(&u.ID, &u.Name, &u.Phone, &u.TokenBalance, &u.PasswordHash, &createdAt, &isAdmin)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	u.IsAdmin = isAdmin == 1
	return u, nil
}

// UpdateUserPassword updates a user's password hash
func UpdateUserPassword(userID int, newHash string) (int, error) {
	res, err := db.Exec(`UPDATE User SET password_hash = ? WHERE id = ?`, newHash, userID)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

// ArchiveUser archives a user and all their ads instead of deleting them
func ArchiveUser(userID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get user data before archiving
	var user User
	err = tx.QueryRow(`SELECT id, name, phone, token_balance, password_hash, created_at 
		FROM User WHERE id = ?`, userID).Scan(
		&user.ID, &user.Name, &user.Phone, &user.TokenBalance,
		&user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return err
	}

	// Archive user with deletion_date
	deletionDate := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = tx.Exec(`INSERT INTO ArchivedUser (id, name, phone, token_balance, password_hash, created_at, deletion_date)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Name, user.Phone, user.TokenBalance,
		user.PasswordHash, user.CreatedAt, deletionDate)
	if err != nil {
		return err
	}

	// Archive all ads by this user
	_, err = tx.Exec(`INSERT INTO ArchivedAd (id, title, description, price, created_at, subcategory_id, user_id, deletion_date)
		SELECT id, title, description, price, created_at, subcategory_id, user_id, ?
		FROM Ad WHERE user_id = ?`, deletionDate, userID)
	if err != nil {
		return err
	}

	// Archive all ad-car relationships for this user's ads
	_, err = tx.Exec(`INSERT INTO ArchivedAdCar (ad_id, car_id)
		SELECT ac.ad_id, ac.car_id
		FROM AdCar ac
		JOIN Ad a ON a.id = ac.ad_id
		WHERE a.user_id = ?`, userID)
	if err != nil {
		return err
	}

	// Mark TokenTransaction records as deleted
	_, err = tx.Exec(`UPDATE TokenTransaction 
		SET user_deleted = 1 
		WHERE user_id = ? OR related_user_id = ?`, userID, userID)
	if err != nil {
		return err
	}

	// Delete original ad-car relationships
	_, err = tx.Exec(`DELETE FROM AdCar WHERE ad_id IN (
		SELECT id FROM Ad WHERE user_id = ?)`, userID)
	if err != nil {
		return err
	}

	// Delete original ads
	_, err = tx.Exec(`DELETE FROM Ad WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	// Delete original user
	_, err = tx.Exec(`DELETE FROM User WHERE id = ?`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RestoreUser moves a user from the ArchivedUser table back to the active User table
func RestoreUser(userID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get user data from archive
	var user User
	err = tx.QueryRow(`SELECT id, name, phone, token_balance, password_hash, created_at, is_admin 
		FROM ArchivedUser WHERE id = ?`, userID).Scan(
		&user.ID, &user.Name, &user.Phone, &user.TokenBalance,
		&user.PasswordHash, &user.CreatedAt, &user.IsAdmin)
	if err != nil {
		return err
	}

	// Restore user
	_, err = tx.Exec(`INSERT INTO User (id, name, phone, token_balance, password_hash, created_at, is_admin)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Name, user.Phone, user.TokenBalance,
		user.PasswordHash, user.CreatedAt, user.IsAdmin)
	if err != nil {
		return err
	}

	// Restore all ads by this user
	_, err = tx.Exec(`INSERT INTO Ad (id, description, price, created_at, subcategory_id, user_id)
		SELECT id, description, price, created_at, subcategory_id, user_id
		FROM ArchivedAd WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	// Restore all ad-car relationships for this user's ads
	_, err = tx.Exec(`INSERT INTO AdCar (ad_id, car_id)
		SELECT aac.ad_id, aac.car_id
		FROM ArchivedAdCar aac
		JOIN ArchivedAd ad ON ad.id = aac.ad_id
		WHERE ad.user_id = ?`, userID)
	if err != nil {
		return err
	}

	// Un-mark TokenTransaction records
	_, err = tx.Exec(`UPDATE TokenTransaction 
		SET user_deleted = 0 
		WHERE user_id = ? OR related_user_id = ?`, userID, userID)
	if err != nil {
		return err
	}

	// Delete from archive tables
	_, err = tx.Exec(`DELETE FROM ArchivedAdCar WHERE ad_id IN (
		SELECT id FROM ArchivedAd WHERE user_id = ?)`, userID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM ArchivedAd WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetAllUsers returns all users in the system
func GetAllUsers() ([]User, error) {
	rows, err := db.Query(`SELECT id, name, phone, token_balance, password_hash, created_at, is_admin FROM User`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var createdAt string
		var isAdmin int
		err := rows.Scan(&u.ID, &u.Name, &u.Phone, &u.TokenBalance, &u.PasswordHash, &createdAt, &isAdmin)
		if err != nil {
			return nil, err
		}
		u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		u.IsAdmin = isAdmin == 1
		users = append(users, u)
	}
	return users, nil
}

// GetAllArchivedUsers returns all archived users
func GetAllArchivedUsers() ([]User, error) {
	rows, err := db.Query(`SELECT id, name, phone, token_balance, password_hash, created_at, is_admin, deletion_date FROM ArchivedUser`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var createdAt, deletionDate string
		var isAdmin int
		err := rows.Scan(&u.ID, &u.Name, &u.Phone, &u.TokenBalance, &u.PasswordHash, &createdAt, &isAdmin, &deletionDate)
		if err != nil {
			return nil, err
		}
		u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		u.IsAdmin = isAdmin == 1
		if parsedTime, err := time.Parse(time.RFC3339Nano, deletionDate); err == nil {
			u.DeletionDate = &parsedTime
		}
		users = append(users, u)
	}
	return users, nil
}

// Transaction represents a token transaction
type Transaction struct {
	ID        int
	UserID    int
	Amount    float64
	Type      string
	CreatedAt time.Time
}

// GetAllTransactions returns all token transactions
func GetAllTransactions() ([]Transaction, error) {
	rows, err := db.Query(`SELECT id, user_id, amount, type, created_at FROM TokenTransaction`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		var createdAt string
		err := rows.Scan(&t.ID, &t.UserID, &t.Amount, &t.Type, &createdAt)
		if err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		transactions = append(transactions, t)
	}
	return transactions, nil
}

// SetAdmin sets or removes admin privileges for a user
func SetAdmin(userID int, isAdmin bool) error {
	_, err := db.Exec(`UPDATE User SET is_admin = ? WHERE id = ?`,
		map[bool]int{false: 0, true: 1}[isAdmin], userID)
	return err
}
