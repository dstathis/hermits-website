package db

import (
	"database/sql"
	"time"
)

type Event struct {
	ID          string
	Title       string
	Format      string
	Description string
	Date        time.Time
	Location    string
	LocationURL string
	EntryFee    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func GetUpcomingEvents(d *sql.DB) ([]Event, error) {
	rows, err := d.Query(`
		SELECT id, title, format, description, date, location, location_url, entry_fee, created_at, updated_at
		FROM events
		WHERE date >= now()
		ORDER BY date ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func GetPastEvents(d *sql.DB) ([]Event, error) {
	rows, err := d.Query(`
		SELECT id, title, format, description, date, location, location_url, entry_fee, created_at, updated_at
		FROM events
		WHERE date < now()
		ORDER BY date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func GetAllEvents(d *sql.DB) ([]Event, error) {
	rows, err := d.Query(`
		SELECT id, title, format, description, date, location, location_url, entry_fee, created_at, updated_at
		FROM events
		ORDER BY date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func GetNextEvent(d *sql.DB) (*Event, error) {
	row := d.QueryRow(`
		SELECT id, title, format, description, date, location, location_url, entry_fee, created_at, updated_at
		FROM events
		WHERE date >= now()
		ORDER BY date ASC
		LIMIT 1
	`)
	e := &Event{}
	err := row.Scan(&e.ID, &e.Title, &e.Format, &e.Description, &e.Date, &e.Location, &e.LocationURL, &e.EntryFee, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

func GetEventByID(d *sql.DB, id string) (*Event, error) {
	row := d.QueryRow(`
		SELECT id, title, format, description, date, location, location_url, entry_fee, created_at, updated_at
		FROM events WHERE id = $1
	`, id)
	e := &Event{}
	err := row.Scan(&e.ID, &e.Title, &e.Format, &e.Description, &e.Date, &e.Location, &e.LocationURL, &e.EntryFee, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

func CreateEvent(d *sql.DB, e *Event) error {
	return d.QueryRow(`
		INSERT INTO events (title, format, description, date, location, location_url, entry_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, e.Title, e.Format, e.Description, e.Date, e.Location, e.LocationURL, e.EntryFee).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

func UpdateEvent(d *sql.DB, e *Event) error {
	_, err := d.Exec(`
		UPDATE events
		SET title = $2, format = $3, description = $4, date = $5, location = $6, location_url = $7, entry_fee = $8, updated_at = now()
		WHERE id = $1
	`, e.ID, e.Title, e.Format, e.Description, e.Date, e.Location, e.LocationURL, e.EntryFee)
	return err
}

func DeleteEvent(d *sql.DB, id string) error {
	_, err := d.Exec(`DELETE FROM events WHERE id = $1`, id)
	return err
}

func scanEvents(rows *sql.Rows) ([]Event, error) {
	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Title, &e.Format, &e.Description, &e.Date, &e.Location, &e.LocationURL, &e.EntryFee, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
