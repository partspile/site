package user

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/parts-pile/site/config"
)

type User struct {
	ID           int
	Name         string
	Phone        string
	TokenBalance float64
	PasswordHash string
	CreatedAt    time.Time
	IsAdmin      bool
}

var db *sql.DB

// session represents a single user session.
type session struct {
	userID    int
	expiresAt time.Time
}

// sessionStore holds our active sessions and a mutex for safe access.
type sessionStore struct {
	sync.RWMutex
	sessions map[string]session
}

// globalSessionStore is our in-memory session store.
var globalSessionStore = &sessionStore{
	sessions: make(map[string]session),
}

func InitDB(database *sql.DB) {
	db = database
	go globalSessionStore.cleanupExpiredSessions()
}

// cleanupExpiredSessions periodically iterates through the session store and
// removes sessions that have expired.
func (s *sessionStore) cleanupExpiredSessions() {
	for {
		time.Sleep(config.SessionCleanupInterval)
		s.Lock()
		for token, sess := range s.sessions {
			if time.Now().After(sess.expiresAt) {
				delete(s.sessions, token)
			}
		}
		s.Unlock()
	}
}

func CreateSession(userID int) (string, error) {
	sessionToken := fmt.Sprintf("session_token_%d_%d", userID, time.Now().UnixNano())
	expiresAt := time.Now().Add(config.SessionDuration)

	globalSessionStore.Lock()
	defer globalSessionStore.Unlock()

	globalSessionStore.sessions[sessionToken] = session{
		userID:    userID,
		expiresAt: expiresAt,
	}

	return sessionToken, nil
}

func VerifySession(sessionToken string) (int, error) {
	globalSessionStore.RLock()
	defer globalSessionStore.RUnlock()

	sess, ok := globalSessionStore.sessions[sessionToken]
	if !ok {
		return 0, fmt.Errorf("invalid session token")
	}

	if time.Now().After(sess.expiresAt) {
		// The cleanup goroutine will eventually get this, but we can also
		// proactively remove it. To do that without a deferred RUnlock,
		// we'd need to manage the lock manually.
		return 0, fmt.Errorf("session expired")
	}

	return sess.userID, nil
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

// GetUserByID retrieves a user by ID
func GetUserByID(id int) (User, error) {
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

// UpdateTokenBalance updates a user's token balance
func UpdateTokenBalance(userID int, newBalance float64) error {
	_, err := db.Exec(`UPDATE User SET token_balance = ? WHERE id = ?`, newBalance, userID)
	return err
}

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

// DeleteUser archives a user and all their ads instead of deleting them
func DeleteUser(userID int) error {
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

	// Archive user
	_, err = tx.Exec(`INSERT INTO UserDead (id, name, phone, token_balance, password_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		user.ID, user.Name, user.Phone, user.TokenBalance,
		user.PasswordHash, user.CreatedAt)
	if err != nil {
		return err
	}

	// Archive all ads by this user
	_, err = tx.Exec(`INSERT INTO AdDead (id, description, price, created_at, subcategory_id, user_id)
		SELECT id, description, price, created_at, subcategory_id, user_id
		FROM Ad WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	// Archive all ad-car relationships for this user's ads
	_, err = tx.Exec(`INSERT INTO AdCarDead (ad_id, car_id)
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

// SetAdmin sets or removes admin privileges for a user
func SetAdmin(userID int, isAdmin bool) error {
	_, err := db.Exec(`UPDATE User SET is_admin = ? WHERE id = ?`,
		map[bool]int{false: 0, true: 1}[isAdmin], userID)
	return err
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
