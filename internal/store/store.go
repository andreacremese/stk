// Package store provides a SQLite-backed implementation of the stack.Storer
// interface. No interface is defined here; per Go convention the interface
// lives at the point of use (internal/stack).
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // register "sqlite" driver

	"github.com/andreacremese/stk/internal/stack"
)

const schema = `
CREATE TABLE IF NOT EXISTS stacks (
	id         TEXT NOT NULL PRIMARY KEY,
	repo       TEXT NOT NULL,
	branch     TEXT NOT NULL,
	note       TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_stacks_repo_branch_time
	ON stacks (repo, branch, created_at ASC, id ASC);
`

// timeLayout is used for all round-trips to/from SQLite.
// Fixed 9-digit fractional seconds ensure the formatted string sorts correctly
// as plain text in SQLite. time.RFC3339Nano omits trailing zeros, which breaks
// text sort (e.g. ".1Z" > ".101Z" because 'Z' > '0').
const timeLayout = "2006-01-02T15:04:05.000000000Z07:00"

// errNotFound is a private sentinel returned by removeAtOffset when no row
// exists at the requested offset. Callers map it to the appropriate public error.
var errNotFound = errors.New("no row at offset")

// Store is a SQLite-backed Storer. Obtain one via Open.
type Store struct {
	db *sql.DB
}

// DefaultDBPath returns the path where the database should live.
// It respects AGENT_STACK_DB_PATH; if unset it uses
// os.UserConfigDir()/agent-stack/stacks.db.
func DefaultDBPath() (string, error) {
	if p := os.Getenv("AGENT_STACK_DB_PATH"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolving config dir: %w", err)
	}
	return filepath.Join(dir, "agent-stack", "stacks.db"), nil
}

// Open opens (or creates) the SQLite database at path and applies the schema.
// The caller is responsible for calling Close when done.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close() // Best effort cleanup
		return nil, fmt.Errorf("pinging db: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close() // Best effort cleanup
		return nil, fmt.Errorf("applying schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// PushItem inserts a fully-constructed item into the database.
func (s *Store) PushItem(item stack.Item) error {
	_, err := s.db.Exec(
		`INSERT INTO stacks (id, repo, branch, note, created_at) VALUES (?, ?, ?, ?, ?)`,
		item.ID,
		item.Repo,
		item.Branch,
		item.Note,
		item.CreatedAt.UTC().Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

// List returns all items for the given repo+branch ordered by created_at DESC,
// id DESC (index 0 is the newest / top of stack).
func (s *Store) List(repo, branch string) ([]stack.Item, error) {
	rows, err := s.db.Query(
		`SELECT id, repo, branch, note, created_at
		 FROM stacks
		 WHERE repo = ? AND branch = ?
		 ORDER BY created_at DESC, id DESC`,
		repo, branch,
	)
	if err != nil {
		return nil, fmt.Errorf("list query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []stack.Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list rows: %w", err)
	}
	return items, nil
}

// RemoveTop removes and returns the item at index 0 (newest created_at).
// Returns stack.ErrEmptyStack if there are no items.
func (s *Store) RemoveTop(repo, branch string) (*stack.Item, error) {
	item, err := s.removeAtOffset(repo, branch, 0)
	if errors.Is(err, errNotFound) {
		return nil, stack.ErrEmptyStack
	}
	return item, err
}

// RemoveAt removes and returns the item at the given 0-based index.
// Returns stack.ErrIndexOutOfRange if the index is out of bounds.
func (s *Store) RemoveAt(repo, branch string, index int) (*stack.Item, error) {
	item, err := s.removeAtOffset(repo, branch, index)
	if errors.Is(err, errNotFound) {
		return nil, stack.ErrIndexOutOfRange
	}
	return item, err
}

// DeleteAll removes all items for the given repo+branch.
func (s *Store) DeleteAll(repo, branch string) error {
	_, err := s.db.Exec(
		`DELETE FROM stacks WHERE repo = ? AND branch = ?`,
		repo, branch,
	)
	if err != nil {
		return fmt.Errorf("delete all: %w", err)
	}
	return nil
}

// removeAtOffset selects the item at the given 0-based OFFSET within a
// repo+branch stack, deletes it, and returns it.
// Returns errNotFound when no row exists at that offset.
func (s *Store) removeAtOffset(repo, branch string, offset int) (*stack.Item, error) {
	row := s.db.QueryRow(
		`SELECT id, repo, branch, note, created_at
		 FROM stacks
		 WHERE repo = ? AND branch = ?
		 ORDER BY created_at DESC, id DESC
		 LIMIT 1 OFFSET ?`,
		repo, branch, offset,
	)
	it, err := scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, fmt.Errorf("query at offset %d: %w", offset, err)
	}
	if _, err := s.db.Exec(`DELETE FROM stacks WHERE id = ?`, it.ID); err != nil {
		return nil, fmt.Errorf("delete id %s: %w", it.ID, err)
	}
	return it, nil
}

// scanRow scans a single *sql.Row into a stack.Item.
func scanRow(row *sql.Row) (*stack.Item, error) {
	var it stack.Item
	var ts string
	if err := row.Scan(&it.ID, &it.Repo, &it.Branch, &it.Note, &ts); err != nil {
		return nil, err
	}
	return parseTime(&it, ts)
}

// scanItem scans the current position of an open *sql.Rows cursor into a stack.Item.
func scanItem(rows *sql.Rows) (stack.Item, error) {
	var it stack.Item
	var ts string
	if err := rows.Scan(&it.ID, &it.Repo, &it.Branch, &it.Note, &ts); err != nil {
		return it, fmt.Errorf("scan: %w", err)
	}
	item, err := parseTime(&it, ts)
	if err != nil {
		return it, err
	}
	return *item, nil
}

// parseTime parses a RFC3339Nano timestamp string into item.CreatedAt.
func parseTime(it *stack.Item, ts string) (*stack.Item, error) {
	t, err := time.Parse(timeLayout, ts)
	if err != nil {
		return nil, fmt.Errorf("parsing timestamp %q: %w", ts, err)
	}
	it.CreatedAt = t
	return it, nil
}
