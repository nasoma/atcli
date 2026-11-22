package storage

import (
	"context"
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
)

type Record struct {
	ID          string `json:"local_operation_id"`
	RequestID   string `json:"provider_request_id"`
	Account     string `json:"account"`
	Recipient   string `json:"recipient"`
	Amount      int64  `json:"amount_minor,omitempty"`
	Currency    string `json:"currency,omitempty"`
	Environment string `json:"environment"`
	Status      string `json:"status"`
	Submission  string `json:"submission"`
	Message     string `json:"provider_message"`
	Created     string `json:"created_at"`
	Updated     string `json:"updated_at"`
	Checked     string `json:"last_checked_at"`
}
type Store struct{ DB *sql.DB }

func Open(path string) (*Store, error) {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	f.Close()
	if e = os.Chmod(path, 0600); e != nil {
		return nil, e
	}
	db, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	s := &Store{db}
	_, e = db.Exec(`PRAGMA busy_timeout=5000; PRAGMA journal_mode=DELETE; PRAGMA synchronous=FULL;
 CREATE TABLE IF NOT EXISTS schema_version(version INTEGER PRIMARY KEY);
 INSERT OR IGNORE INTO schema_version VALUES(1);
 CREATE TABLE IF NOT EXISTS transactions(id TEXT PRIMARY KEY,request_id TEXT NOT NULL DEFAULT '',account TEXT NOT NULL,recipient TEXT NOT NULL,amount INTEGER NOT NULL,currency TEXT NOT NULL,environment TEXT NOT NULL,status TEXT NOT NULL,submission TEXT NOT NULL,message TEXT NOT NULL DEFAULT '',created TEXT NOT NULL,updated TEXT NOT NULL,checked TEXT NOT NULL DEFAULT '');
 CREATE UNIQUE INDEX IF NOT EXISTS provider_id ON transactions(environment,account,request_id) WHERE request_id!='';
 CREATE INDEX IF NOT EXISTS history ON transactions(environment,created);`)
	if e != nil {
		db.Close()
		return nil, e
	}
	var version int
	if e = db.QueryRow("SELECT max(version) FROM schema_version").Scan(&version); e != nil || version != 1 {
		db.Close()
		return nil, fmt.Errorf("unsupported history schema version %d", version)
	}
	return s, nil
}
func Now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// Reserve is one atomic write: concurrent processes cannot bypass deduplication or daily limits.
func (s *Store) Reserve(ctx context.Context, r Record, daily int64) error {
	conn, e := s.DB.Conn(ctx)
	if e != nil {
		return e
	}
	defer conn.Close()
	if _, e = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); e != nil {
		return e
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	var n int
	e = conn.QueryRowContext(ctx, `SELECT count(*) FROM transactions WHERE environment=? AND account=? AND recipient=? AND currency=? AND amount=? AND (status IN ('unknown','pending') OR (status='delivered' AND created>?))`, r.Environment, r.Account, r.Recipient, r.Currency, r.Amount, time.Now().UTC().Add(-2*time.Minute).Format(time.RFC3339Nano)).Scan(&n)
	if e != nil {
		return e
	}
	if n > 0 {
		return fmt.Errorf("duplicate or unresolved purchase exists; reconcile its status before sending again")
	}
	if daily > 0 {
		var total int64
		e = conn.QueryRowContext(ctx, `SELECT coalesce(sum(amount),0) FROM transactions WHERE environment=? AND account=? AND currency=? AND status!='failed' AND created>=?`, r.Environment, r.Account, r.Currency, time.Now().UTC().Format("2006-01-02")+"T00:00:00").Scan(&total)
		if e != nil {
			return e
		}
		if total > daily-r.Amount {
			return fmt.Errorf("local daily spending limit exceeded (UTC day)")
		}
	}
	_, e = conn.ExecContext(ctx, `INSERT INTO transactions(id,account,recipient,amount,currency,environment,status,submission,created,updated) VALUES(?,?,?,?,?,?,'unknown','intent',?,?)`, r.ID, r.Account, r.Recipient, r.Amount, r.Currency, r.Environment, r.Created, r.Created)
	if e != nil {
		return e
	}
	_, e = conn.ExecContext(ctx, "COMMIT")
	return e
}
func (s *Store) Update(ctx context.Context, r Record) error {
	_, e := s.DB.ExecContext(ctx, `UPDATE transactions SET request_id=?,status=?,submission=?,message=?,updated=?,checked=? WHERE id=?`, r.RequestID, r.Status, r.Submission, r.Message, Now(), r.Checked, r.ID)
	return e
}
func (s *Store) List(ctx context.Context, env, status string, limit int) ([]Record, error) {
	rows, e := s.DB.QueryContext(ctx, `SELECT id,request_id,account,recipient,amount,currency,environment,status,submission,message,created,updated,checked FROM transactions WHERE (?='' OR environment=?) AND (?='' OR status=?) ORDER BY created DESC LIMIT ?`, env, env, status, status, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		var r Record
		if e = rows.Scan(&r.ID, &r.RequestID, &r.Account, &r.Recipient, &r.Amount, &r.Currency, &r.Environment, &r.Status, &r.Submission, &r.Message, &r.Created, &r.Updated, &r.Checked); e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (s *Store) Find(ctx context.Context, env, account, id string) (Record, error) {
	var r Record
	e := s.DB.QueryRowContext(ctx, `SELECT id,request_id,account,recipient,amount,currency,environment,status,submission,message,created,updated,checked FROM transactions WHERE environment=? AND account=? AND (id=? OR request_id=?)`, env, account, id, id).Scan(&r.ID, &r.RequestID, &r.Account, &r.Recipient, &r.Amount, &r.Currency, &r.Environment, &r.Status, &r.Submission, &r.Message, &r.Created, &r.Updated, &r.Checked)
	return r, e
}

// LocalTime formats a persisted UTC timestamp for human display; JSON keeps UTC.
func LocalTime(s string) string {
	if s == "" {
		return "Not checked"
	}
	t, e := time.Parse(time.RFC3339Nano, s)
	if e != nil {
		return s
	}
	return t.Local().Format("2006-01-02 15:04:05 MST")
}
