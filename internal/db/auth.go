package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AdminUser struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func CreateAdminUser(db *sql.DB, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO admin_users (username, password_hash) VALUES ($1, $2)`, username, string(hash))
	return err
}

func Authenticate(db *sql.DB, username, password string) (*AdminUser, error) {
	u := &AdminUser{}
	err := db.QueryRow(`SELECT id, username, password_hash, created_at FROM admin_users WHERE username = $1`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, nil
	}
	return u, nil
}

func CreateSession(db *sql.DB, userID string) (*Session, error) {
	s := &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_, err := db.Exec(`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`, s.ID, s.UserID, s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func GetSession(db *sql.DB, sessionID string) (*Session, error) {
	s := &Session{}
	err := db.QueryRow(`
		SELECT id, user_id, expires_at, created_at FROM sessions
		WHERE id = $1 AND expires_at > now()
	`, sessionID).Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func DeleteSession(db *sql.DB, sessionID string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}

func CleanExpiredSessions(db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE expires_at < now()`)
	return err
}
