package user

import (
	"database/sql"
	"time"
)

type User struct {
	ID           int
	Name         string
	Phone        string
	TokenBalance float64
	PasswordHash string
	CreatedAt    time.Time
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

// GetUserByPhone retrieves a user by phone number
func GetUserByPhone(phone string) (User, error) {
	row := db.QueryRow(`SELECT id, name, phone, token_balance, password_hash, created_at FROM User WHERE phone = ?`, phone)
	var u User
	var createdAt string
	err := row.Scan(&u.ID, &u.Name, &u.Phone, &u.TokenBalance, &u.PasswordHash, &createdAt)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return u, nil
}

// GetUserByID retrieves a user by ID
func GetUserByID(id int) (User, error) {
	row := db.QueryRow(`SELECT id, name, phone, token_balance, password_hash, created_at FROM User WHERE id = ?`, id)
	var u User
	var createdAt string
	err := row.Scan(&u.ID, &u.Name, &u.Phone, &u.TokenBalance, &u.PasswordHash, &createdAt)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return u, nil
}

// UpdateTokenBalance updates a user's token balance
func UpdateTokenBalance(userID int, newBalance float64) error {
	_, err := db.Exec(`UPDATE User SET token_balance = ? WHERE id = ?`, newBalance, userID)
	return err
}
