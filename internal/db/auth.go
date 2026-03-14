package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AdminUser struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	InviteToken  string
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

func InviteAdminUser(db *sql.DB, username, email string) (string, error) {
	token := uuid.New().String()
	_, err := db.Exec(
		`INSERT INTO admin_users (username, email, invite_token) VALUES ($1, $2, $3)`,
		username, email, token,
	)
	if err != nil {
		return "", err
	}
	return token, nil
}

func GetAdminUserByInviteToken(db *sql.DB, token string) (*AdminUser, error) {
	u := &AdminUser{}
	err := db.QueryRow(
		`SELECT id, username, email, password_hash, invite_token, created_at FROM admin_users WHERE invite_token = $1`,
		token,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.InviteToken, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func AcceptInvite(db *sql.DB, token, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	result, err := db.Exec(
		`UPDATE admin_users SET password_hash = $1, invite_token = '' WHERE invite_token = $2`,
		string(hash), token,
	)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func GetAllAdminUsers(db *sql.DB) ([]AdminUser, error) {
	rows, err := db.Query(`SELECT id, username, email, password_hash, invite_token, created_at FROM admin_users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []AdminUser
	for rows.Next() {
		var u AdminUser
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.InviteToken, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func DeleteAdminUser(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM admin_users WHERE id = $1`, id)
	return err
}

func ChangePassword(db *sql.DB, userID, oldPassword, newPassword string) error {
	var hash string
	err := db.QueryRow(`SELECT password_hash FROM admin_users WHERE id = $1`, userID).Scan(&hash)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("incorrect current password")
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE admin_users SET password_hash = $1 WHERE id = $2`, string(newHash), userID)
	return err
}

func Authenticate(db *sql.DB, username, password string) (*AdminUser, error) {
	u := &AdminUser{}
	err := db.QueryRow(`SELECT id, username, email, password_hash, invite_token, created_at FROM admin_users WHERE username = $1`, username).
		Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.InviteToken, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Pending invites (no password set) can't log in
	if u.PasswordHash == "" {
		return nil, nil
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
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
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
