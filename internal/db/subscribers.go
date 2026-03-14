package db

import (
	"database/sql"
	"time"
)

type Subscriber struct {
	ID        string
	Email     string
	Name      string
	Token     string
	Confirmed bool
	CreatedAt time.Time
}

func CreateSubscriber(db *sql.DB, email, name string) (*Subscriber, error) {
	s := &Subscriber{}
	err := db.QueryRow(`
		INSERT INTO subscribers (email, name)
		VALUES ($1, $2)
		ON CONFLICT (email) DO NOTHING
		RETURNING id, email, name, token, confirmed, created_at
	`, email, name).Scan(&s.ID, &s.Email, &s.Name, &s.Token, &s.Confirmed, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // already exists
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func GetAllSubscribers(db *sql.DB) ([]Subscriber, error) {
	rows, err := db.Query(`
		SELECT id, email, name, token, confirmed, created_at
		FROM subscribers
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscriber
	for rows.Next() {
		var s Subscriber
		if err := rows.Scan(&s.ID, &s.Email, &s.Name, &s.Token, &s.Confirmed, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func DeleteSubscriber(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM subscribers WHERE id = $1`, id)
	return err
}

func UnsubscribeByToken(db *sql.DB, token string) error {
	result, err := db.Exec(`UPDATE subscribers SET confirmed = false WHERE token = $1`, token)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func GetConfirmedSubscriberEmails(db *sql.DB) ([]Subscriber, error) {
	rows, err := db.Query(`SELECT id, email, name, token, confirmed, created_at FROM subscribers WHERE confirmed = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscriber
	for rows.Next() {
		var s Subscriber
		if err := rows.Scan(&s.ID, &s.Email, &s.Name, &s.Token, &s.Confirmed, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func ConfirmSubscriber(db *sql.DB, token string) error {
	result, err := db.Exec(`UPDATE subscribers SET confirmed = true WHERE token = $1 AND confirmed = false`, token)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
