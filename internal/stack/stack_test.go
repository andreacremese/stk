package stack_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreacremese/stk/internal/stack"
)

// mockStore is an in-memory Storer used exclusively for unit tests.
type mockStore struct {
	items []stack.Item
}

func (m *mockStore) PushItem(item stack.Item) error {
	m.items = append(m.items, item)
	return nil
}

func (m *mockStore) List(repo, branch string) ([]stack.Item, error) {
	var out []stack.Item
	for _, it := range m.items {
		if it.Repo == repo && it.Branch == branch {
			out = append(out, it)
		}
	}
	return out, nil
}

func (m *mockStore) RemoveTop(repo, branch string) (*stack.Item, error) {
	for i, it := range m.items {
		if it.Repo == repo && it.Branch == branch {
			removed := m.items[i]
			m.items = append(m.items[:i], m.items[i+1:]...)
			return &removed, nil
		}
	}
	return nil, stack.ErrEmptyStack
}

func (m *mockStore) RemoveAt(repo, branch string, index int) (*stack.Item, error) {
	var matches []int
	for i, it := range m.items {
		if it.Repo == repo && it.Branch == branch {
			matches = append(matches, i)
		}
	}
	if index >= len(matches) {
		return nil, stack.ErrIndexOutOfRange
	}
	globalIdx := matches[index]
	removed := m.items[globalIdx]
	m.items = append(m.items[:globalIdx], m.items[globalIdx+1:]...)
	return &removed, nil
}

func (m *mockStore) DeleteAll(repo, branch string) error {
	var remaining []stack.Item
	for _, it := range m.items {
		if it.Repo != repo || it.Branch != branch {
			remaining = append(remaining, it)
		}
	}
	m.items = remaining
	return nil
}

func newService() (*stack.Stack, *mockStore) {
	ms := &mockStore{}
	return stack.New(ms), ms
}

func TestPush(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		svc, ms := newService()
		before := time.Now().UTC()
		item, err := svc.Push("myrepo", "main", "investigate the flaky test")
		require.NoError(t, err)
		assert.NotEmpty(t, item.ID)
		assert.Equal(t, "myrepo", item.Repo)
		assert.Equal(t, "main", item.Branch)
		assert.Equal(t, "investigate the flaky test", item.Note)
		assert.False(t, item.CreatedAt.Before(before), "CreatedAt is before push time")
		assert.Len(t, ms.items, 1)
	})

	t.Run("note too long", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		_, err := svc.Push("myrepo", "main", strings.Repeat("x", stack.MaxNoteLen+1))
		assert.ErrorIs(t, err, stack.ErrNoteTooLong)
	})

	t.Run("note at max length is accepted", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		_, err := svc.Push("myrepo", "main", strings.Repeat("x", stack.MaxNoteLen))
		assert.NoError(t, err)
	})

	t.Run("ids are unique across pushes", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		a, _ := svc.Push("r", "b", "first")
		b, _ := svc.Push("r", "b", "second")
		assert.NotEqual(t, a.ID, b.ID)
	})
}

func TestShow(t *testing.T) {
	t.Parallel()

	t.Run("returns items in insertion order", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		svc.Push("repo", "main", "first")  //nolint:errcheck
		svc.Push("repo", "main", "second") //nolint:errcheck
		svc.Push("repo", "main", "third")  //nolint:errcheck
		items, err := svc.Show("repo", "main")
		require.NoError(t, err)
		require.Len(t, items, 3)
		assert.Equal(t, "first", items[0].Note)
		assert.Equal(t, "second", items[1].Note)
		assert.Equal(t, "third", items[2].Note)
	})

	t.Run("empty stack returns empty slice", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		items, err := svc.Show("repo", "main")
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("isolated by repo and branch", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		svc.Push("repo", "main", "on main")    //nolint:errcheck
		svc.Push("repo", "feature", "on feat") //nolint:errcheck
		items, _ := svc.Show("repo", "main")
		require.Len(t, items, 1)
		assert.Equal(t, "on main", items[0].Note)
	})
}

func TestPop(t *testing.T) {
	t.Parallel()

	t.Run("removes and returns oldest item", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		svc.Push("repo", "main", "first")  //nolint:errcheck
		svc.Push("repo", "main", "second") //nolint:errcheck
		item, err := svc.Pop("repo", "main")
		require.NoError(t, err)
		assert.Equal(t, "first", item.Note)
		remaining, _ := svc.Show("repo", "main")
		assert.Len(t, remaining, 1)
	})

	t.Run("empty stack returns ErrEmptyStack", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		_, err := svc.Pop("repo", "main")
		assert.ErrorIs(t, err, stack.ErrEmptyStack)
	})
}

func TestClear(t *testing.T) {
	t.Parallel()

	t.Run("removes all items", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		svc.Push("repo", "main", "a") //nolint:errcheck
		svc.Push("repo", "main", "b") //nolint:errcheck
		require.NoError(t, svc.Clear("repo", "main"))
		items, _ := svc.Show("repo", "main")
		assert.Empty(t, items)
	})

	t.Run("does not affect other stacks", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		svc.Push("repo", "main", "keep me") //nolint:errcheck
		svc.Push("repo", "feat", "remove")  //nolint:errcheck
		svc.Clear("repo", "feat")           //nolint:errcheck
		items, _ := svc.Show("repo", "main")
		assert.Len(t, items, 1)
	})
}

func TestPluck(t *testing.T) {
	t.Parallel()

	t.Run("removes item at given index", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		svc.Push("repo", "main", "zero") //nolint:errcheck
		svc.Push("repo", "main", "one")  //nolint:errcheck
		svc.Push("repo", "main", "two")  //nolint:errcheck
		item, err := svc.Pluck("repo", "main", 1)
		require.NoError(t, err)
		assert.Equal(t, "one", item.Note)
		remaining, _ := svc.Show("repo", "main")
		require.Len(t, remaining, 2)
		assert.Equal(t, "zero", remaining[0].Note)
		assert.Equal(t, "two", remaining[1].Note)
	})

	t.Run("negative index returns ErrIndexOutOfRange", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		svc.Push("repo", "main", "note") //nolint:errcheck
		_, err := svc.Pluck("repo", "main", -1)
		assert.ErrorIs(t, err, stack.ErrIndexOutOfRange)
	})

	t.Run("out of range index returns ErrIndexOutOfRange", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		svc.Push("repo", "main", "only item") //nolint:errcheck
		_, err := svc.Pluck("repo", "main", 5)
		assert.ErrorIs(t, err, stack.ErrIndexOutOfRange)
	})

	t.Run("empty stack returns ErrIndexOutOfRange", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService()
		_, err := svc.Pluck("repo", "main", 0)
		assert.ErrorIs(t, err, stack.ErrIndexOutOfRange)
	})
}
