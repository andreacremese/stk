// Package stack implements the business logic for the agent-stack CLI.
// It defines the Storer interface that the persistence layer must satisfy,
// and exposes a Stack service that all subcommands call into.
package stack

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

// MaxNoteLen is the maximum allowed length for a note.
const MaxNoteLen = 160

// Sentinel errors returned by Stack methods.
var (
	ErrNoteTooLong     = errors.New("note exceeds 160 characters")
	ErrEmptyStack      = errors.New("stack is empty")
	ErrIndexOutOfRange = errors.New("index out of range")
)

// Item represents a single entry in a stack.
type Item struct {
	ID        string
	Repo      string
	Branch    string
	Note      string
	CreatedAt time.Time
}

// Storer is the persistence interface consumed by this package.
// Implementations live in internal/store; the interface is defined here
// per Go convention (interfaces at the point of use).
type Storer interface {
	// PushItem persists a fully-constructed item.
	PushItem(item Item) error

	// List returns all items for the given repo+branch, ordered by
	// created_at DESC (index 0 is the newest / top of stack).
	List(repo, branch string) ([]Item, error)

	// RemoveTop removes and returns the item at index 0 (newest created_at).
	// Returns ErrEmptyStack if there are no items.
	RemoveTop(repo, branch string) (*Item, error)

	// RemoveAt removes and returns the item at the given 0-based index.
	// Returns ErrIndexOutOfRange if index is out of bounds.
	RemoveAt(repo, branch string, index int) (*Item, error)

	// DeleteAll removes all items for the given repo+branch.
	DeleteAll(repo, branch string) error
}

// Stack is the business-logic service. All CLI subcommands go through here.
type Stack struct {
	store Storer
}

// New returns a Stack backed by the provided Storer.
func New(store Storer) *Stack {
	return &Stack{store: store}
}

// Push validates the note, generates a new ID, and persists the item.
func (s *Stack) Push(repo, branch, note string) (*Item, error) {
	if len(note) > MaxNoteLen {
		return nil, ErrNoteTooLong
	}
	id, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("generating id: %w", err)
	}
	item := Item{
		ID:        id,
		Repo:      repo,
		Branch:    branch,
		Note:      note,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.PushItem(item); err != nil {
		return nil, fmt.Errorf("push: %w", err)
	}
	return &item, nil
}

// Show returns all items for the given repo+branch (index 0 = top/newest).
func (s *Stack) Show(repo, branch string) ([]Item, error) {
	items, err := s.store.List(repo, branch)
	if err != nil {
		return nil, fmt.Errorf("show: %w", err)
	}
	return items, nil
}

// Pop removes and returns the top item (index 0, the newest).
// Returns ErrEmptyStack if the stack is empty.
func (s *Stack) Pop(repo, branch string) (*Item, error) {
	item, err := s.store.RemoveTop(repo, branch)
	if err != nil {
		return nil, fmt.Errorf("pop: %w", err)
	}
	return item, nil
}

// Clear removes all items for the given repo+branch.
func (s *Stack) Clear(repo, branch string) error {
	if err := s.store.DeleteAll(repo, branch); err != nil {
		return fmt.Errorf("clear: %w", err)
	}
	return nil
}

// Pluck removes and returns the item at the given 0-based index.
// Returns ErrIndexOutOfRange if the index is invalid.
func (s *Stack) Pluck(repo, branch string, index int) (*Item, error) {
	if index < 0 {
		return nil, ErrIndexOutOfRange
	}
	item, err := s.store.RemoveAt(repo, branch, index)
	if err != nil {
		return nil, fmt.Errorf("pluck: %w", err)
	}
	return item, nil
}

// newUUID returns a random UUID v4 string.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// Set version 4 and variant bits per RFC 4122.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
