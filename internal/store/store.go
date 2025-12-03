package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/notifyd-eng/notifyd/internal/config"
)

type Store struct {
	db *sql.DB
}

type Notification struct {
	ID        string    `json:"id"`
	Channel   string    `json:"channel"`
	Recipient string    `json:"recipient"`
	Subject   string    `json:"subject,omitempty"`
	Body      string    `json:"body"`
	Priority  int       `json:"priority"`
	Status    string    `json:"status"`
	Attempts  int       `json:"attempts"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
}

type ListFilter struct {
	Channel string
	Status  string
	Limit   int
	Offset  int
}

func Open(cfg config.StoreConfig) (*Store, error) {
	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate() error {
	_, err := s.db.Exec(schema)
	return err
}

const schema = `
CREATE TABLE IF NOT EXISTS notifications (
    id         TEXT PRIMARY KEY,
    channel    TEXT NOT NULL,
    recipient  TEXT NOT NULL,
    subject    TEXT DEFAULT '',
    body       TEXT NOT NULL,
    priority   INTEGER DEFAULT 0,
    status     TEXT DEFAULT 'pending',
    attempts   INTEGER DEFAULT 0,
    metadata   TEXT DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    sent_at    DATETIME
);

CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_channel ON notifications(channel);
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at);

CREATE TABLE IF NOT EXISTS delivery_log (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    notification_id TEXT NOT NULL REFERENCES notifications(id),
    attempt         INTEGER NOT NULL,
    status          TEXT NOT NULL,
    error           TEXT DEFAULT '',
    duration_ms     INTEGER,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_delivery_log_nid ON delivery_log(notification_id);
`

func (s *Store) Insert(n *Notification) error {
	_, err := s.db.Exec(
		`INSERT INTO notifications (id, channel, recipient, subject, body, priority, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.Channel, n.Recipient, n.Subject, n.Body, n.Priority, "pending",
	)
	return err
}

func (s *Store) Get(id string) (*Notification, error) {
	n := &Notification{}
	err := s.db.QueryRow(
		`SELECT id, channel, recipient, subject, body, priority, status, attempts, created_at, updated_at, sent_at
		 FROM notifications WHERE id = ?`, id,
	).Scan(&n.ID, &n.Channel, &n.Recipient, &n.Subject, &n.Body, &n.Priority,
		&n.Status, &n.Attempts, &n.CreatedAt, &n.UpdatedAt, &n.SentAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return n, err
}

func (s *Store) List(f ListFilter) ([]*Notification, error) {
	query := `SELECT id, channel, recipient, subject, body, priority, status, attempts, created_at, updated_at, sent_at
	           FROM notifications WHERE 1=1`
	var args []interface{}

	if f.Channel != "" {
		query += " AND channel = ?"
		args = append(args, f.Channel)
	}
	if f.Status != "" {
		query += " AND status = ?"
		args = append(args, f.Status)
	}

	query += " ORDER BY created_at DESC"

	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}
	if f.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, f.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*Notification
	for rows.Next() {
		n := &Notification{}
		if err := rows.Scan(&n.ID, &n.Channel, &n.Recipient, &n.Subject, &n.Body,
			&n.Priority, &n.Status, &n.Attempts, &n.CreatedAt, &n.UpdatedAt, &n.SentAt); err != nil {
			return nil, err
		}
		results = append(results, n)
	}
	return results, rows.Err()
}

func (s *Store) UpdateStatus(id, status string) error {
	_, err := s.db.Exec(
		`UPDATE notifications SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, id,
	)
	return err
}

func (s *Store) MarkSent(id string) error {
	_, err := s.db.Exec(
		`UPDATE notifications SET status = 'sent', sent_at = CURRENT_TIMESTAMP,
		 attempts = attempts + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id,
	)
	return err
}

func (s *Store) MarkFailed(id string, errMsg string) error {
	_, err := s.db.Exec(
		`UPDATE notifications SET status = 'failed', attempts = attempts + 1,
		 updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id,
	)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`INSERT INTO delivery_log (notification_id, attempt, status, error)
		 SELECT id, attempts, 'failed', ? FROM notifications WHERE id = ?`,
		errMsg, id,
	)
	return err
}

func (s *Store) PendingBatch(limit int) ([]*Notification, error) {
	return s.List(ListFilter{Status: "pending", Limit: limit})
}

func (s *Store) Stats() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT status, COUNT(*) FROM notifications GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}
	return stats, rows.Err()
}
