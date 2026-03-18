package store_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreacremese/stk/internal/stack"
	"github.com/andreacremese/stk/internal/store"
)

// newStore opens a real SQLite-backed Store in t's temp directory.
func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err, "opening store")
	t.Cleanup(func() { assert.NoError(t, s.Close()) })
	return s
}

// mkItem builds a stack.Item with explicit fields.
func mkItem(id, repo, branch, note string, createdAt time.Time) stack.Item {
	return stack.Item{
		ID:        id,
		Repo:      repo,
		Branch:    branch,
		Note:      note,
		CreatedAt: createdAt.UTC().Truncate(time.Millisecond),
	}
}

func TestOpen(t *testing.T) {
	t.Parallel()

	t.Run("creates nested directory if missing", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "a", "b", "c", "test.db")
		s, err := store.Open(path)
		require.NoError(t, err)
		require.NoError(t, s.Close())
	})

	t.Run("is idempotent on repeated opens", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "test.db")
		s1, err := store.Open(path)
		require.NoError(t, err)
		require.NoError(t, s1.Close())
		s2, err := store.Open(path)
		require.NoError(t, err)
		require.NoError(t, s2.Close())
	})
}

func TestDefaultDBPath(t *testing.T) {
	// Cannot use t.Parallel: subtests use t.Setenv, which panics if any
	// ancestor test is parallel.

	t.Run("respects AGENT_STACK_DB_PATH env var", func(t *testing.T) {
		t.Setenv("AGENT_STACK_DB_PATH", "/tmp/custom.db")
		path, err := store.DefaultDBPath()
		require.NoError(t, err)
		assert.Equal(t, "/tmp/custom.db", path)
	})

	t.Run("returns non-empty platform default", func(t *testing.T) {
		t.Setenv("AGENT_STACK_DB_PATH", "")
		path, err := store.DefaultDBPath()
		require.NoError(t, err)
		assert.NotEmpty(t, path)
		assert.Contains(t, path, "agent-stack")
		assert.Contains(t, path, "stacks.db")
	})
}

func TestPushItem(t *testing.T) {
	t.Parallel()

	t.Run("persists item retrievable via List", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		ts := time.Now().UTC().Truncate(time.Millisecond)
		it := mkItem("id-1", "repo", "main", "fix the thing", ts)
		require.NoError(t, s.PushItem(it))

		items, err := s.List("repo", "main")
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, it.ID, items[0].ID)
		assert.Equal(t, it.Note, items[0].Note)
		assert.Equal(t, it.Repo, items[0].Repo)
		assert.Equal(t, it.Branch, items[0].Branch)
		assert.WithinDuration(t, ts, items[0].CreatedAt, time.Millisecond)
	})

	t.Run("duplicate id returns error", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		it := mkItem("dup-id", "repo", "main", "note", time.Now())
		require.NoError(t, s.PushItem(it))
		assert.Error(t, s.PushItem(it))
	})
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("returns items in created_at DESC order", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		base := time.Now().UTC()
		require.NoError(t, s.PushItem(mkItem("a", "repo", "main", "first", base)))
		require.NoError(t, s.PushItem(mkItem("b", "repo", "main", "second", base.Add(time.Millisecond))))
		require.NoError(t, s.PushItem(mkItem("c", "repo", "main", "third", base.Add(2*time.Millisecond))))

		items, err := s.List("repo", "main")
		require.NoError(t, err)
		require.Len(t, items, 3)
		assert.Equal(t, "third", items[0].Note)
		assert.Equal(t, "second", items[1].Note)
		assert.Equal(t, "first", items[2].Note)
	})

	t.Run("empty stack returns empty slice", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		items, err := s.List("repo", "main")
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("isolated by repo and branch", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		base := time.Now().UTC()
		require.NoError(t, s.PushItem(mkItem("a", "repo", "main", "on main", base)))
		require.NoError(t, s.PushItem(mkItem("b", "repo", "feature", "on feat", base.Add(time.Millisecond))))

		items, err := s.List("repo", "main")
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "on main", items[0].Note)
	})
}

func TestRemoveTop(t *testing.T) {
	t.Parallel()

	t.Run("removes and returns newest item", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		base := time.Now().UTC()
		require.NoError(t, s.PushItem(mkItem("a", "repo", "main", "first", base)))
		require.NoError(t, s.PushItem(mkItem("b", "repo", "main", "second", base.Add(time.Millisecond))))

		got, err := s.RemoveTop("repo", "main")
		require.NoError(t, err)
		assert.Equal(t, "second", got.Note)

		remaining, err := s.List("repo", "main")
		require.NoError(t, err)
		require.Len(t, remaining, 1)
		assert.Equal(t, "first", remaining[0].Note)
	})

	t.Run("empty stack returns ErrEmptyStack", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		_, err := s.RemoveTop("repo", "main")
		assert.ErrorIs(t, err, stack.ErrEmptyStack)
	})
}

func TestRemoveAt(t *testing.T) {
	t.Parallel()

	t.Run("removes item at given index", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		base := time.Now().UTC()
		require.NoError(t, s.PushItem(mkItem("a", "repo", "main", "zero", base)))
		require.NoError(t, s.PushItem(mkItem("b", "repo", "main", "one", base.Add(time.Millisecond))))
		require.NoError(t, s.PushItem(mkItem("c", "repo", "main", "two", base.Add(2*time.Millisecond))))

		got, err := s.RemoveAt("repo", "main", 1)
		require.NoError(t, err)
		assert.Equal(t, "one", got.Note)

		remaining, err := s.List("repo", "main")
		require.NoError(t, err)
		require.Len(t, remaining, 2)
		assert.Equal(t, "two", remaining[0].Note)
		assert.Equal(t, "zero", remaining[1].Note)
	})

	t.Run("out of range index returns ErrIndexOutOfRange", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		require.NoError(t, s.PushItem(mkItem("a", "repo", "main", "only", time.Now())))
		_, err := s.RemoveAt("repo", "main", 5)
		assert.ErrorIs(t, err, stack.ErrIndexOutOfRange)
	})

	t.Run("empty stack returns ErrIndexOutOfRange", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		_, err := s.RemoveAt("repo", "main", 0)
		assert.ErrorIs(t, err, stack.ErrIndexOutOfRange)
	})
}

func TestDeleteAll(t *testing.T) {
	t.Parallel()

	t.Run("removes all items", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		base := time.Now().UTC()
		require.NoError(t, s.PushItem(mkItem("a", "repo", "main", "first", base)))
		require.NoError(t, s.PushItem(mkItem("b", "repo", "main", "second", base.Add(time.Millisecond))))

		require.NoError(t, s.DeleteAll("repo", "main"))

		items, err := s.List("repo", "main")
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("does not affect other stacks", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		base := time.Now().UTC()
		require.NoError(t, s.PushItem(mkItem("a", "repo", "main", "keep me", base)))
		require.NoError(t, s.PushItem(mkItem("b", "repo", "feat", "remove me", base.Add(time.Millisecond))))

		require.NoError(t, s.DeleteAll("repo", "feat"))

		items, err := s.List("repo", "main")
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "keep me", items[0].Note)
	})

	t.Run("no-op on empty stack", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		assert.NoError(t, s.DeleteAll("repo", "main"))
	})
}
