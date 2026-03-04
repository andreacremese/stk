package gitctx_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreacremese/stk/internal/gitctx"
)

// response holds the stubbed return values for a single git invocation.
type response struct {
	out string
	err error
}

// stubLookup returns a LookupFunc whose behaviour is driven by a map from
// space-joined git args (e.g. "rev-parse --show-toplevel") to a response.
// It calls t.Fatalf for any git invocation not present in the map.
func stubLookup(t *testing.T, calls map[string]response) gitctx.LookupFunc {
	t.Helper()
	return func(args ...string) (string, error) {
		key := strings.Join(args, " ")
		r, ok := calls[key]
		if !ok {
			t.Fatalf("unexpected git call: git %s", key)
		}
		return r.out, r.err
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("explicit repo and branch used as-is without git", func(t *testing.T) {
		t.Parallel()
		lookup := func(args ...string) (string, error) {
			t.Fatalf("git must not be called; got args: %v", args)
			return "", nil
		}
		ctx, err := gitctx.Resolve("myrepo", "main", lookup)
		require.NoError(t, err)
		assert.Equal(t, "myrepo", ctx.Repo)
		assert.Equal(t, "main", ctx.Branch)
	})

	t.Run("underscore repo resolved to git toplevel basename", func(t *testing.T) {
		t.Parallel()
		lookup := stubLookup(t, map[string]response{
			"rev-parse --show-toplevel": {out: "/home/user/projects/my-repo"},
		})
		ctx, err := gitctx.Resolve("_", "main", lookup)
		require.NoError(t, err)
		assert.Equal(t, "my-repo", ctx.Repo)
		assert.Equal(t, "main", ctx.Branch)
	})

	t.Run("empty branch resolved from git", func(t *testing.T) {
		t.Parallel()
		lookup := stubLookup(t, map[string]response{
			"branch --show-current": {out: "feature/new-thing"},
		})
		ctx, err := gitctx.Resolve("myrepo", "", lookup)
		require.NoError(t, err)
		assert.Equal(t, "myrepo", ctx.Repo)
		assert.Equal(t, "feature/new-thing", ctx.Branch)
	})

	t.Run("both underscore resolved from git", func(t *testing.T) {
		t.Parallel()
		lookup := stubLookup(t, map[string]response{
			"rev-parse --show-toplevel": {out: "/projects/agent-stack"},
			"branch --show-current":     {out: "main"},
		})
		ctx, err := gitctx.Resolve("_", "_", lookup)
		require.NoError(t, err)
		assert.Equal(t, "agent-stack", ctx.Repo)
		assert.Equal(t, "main", ctx.Branch)
	})

	t.Run("detached HEAD falls back to short commit hash", func(t *testing.T) {
		t.Parallel()
		lookup := stubLookup(t, map[string]response{
			"rev-parse --show-toplevel": {out: "/projects/myrepo"},
			"branch --show-current":     {out: ""},
			"rev-parse --short HEAD":    {out: "abc1234"},
		})
		ctx, err := gitctx.Resolve("_", "_", lookup)
		require.NoError(t, err)
		assert.Equal(t, "myrepo", ctx.Repo)
		assert.Equal(t, "abc1234", ctx.Branch)
	})

	t.Run("ErrNotGitRepo propagated when repo needs resolution", func(t *testing.T) {
		t.Parallel()
		lookup := stubLookup(t, map[string]response{
			"rev-parse --show-toplevel": {err: gitctx.ErrNotGitRepo},
		})
		_, err := gitctx.Resolve("_", "main", lookup)
		assert.ErrorIs(t, err, gitctx.ErrNotGitRepo)
	})

	t.Run("ErrNotGitRepo propagated when branch needs resolution", func(t *testing.T) {
		t.Parallel()
		lookup := stubLookup(t, map[string]response{
			"branch --show-current": {err: gitctx.ErrNotGitRepo},
		})
		_, err := gitctx.Resolve("myrepo", "_", lookup)
		assert.ErrorIs(t, err, gitctx.ErrNotGitRepo)
	})

	t.Run("lookup error on toplevel is forwarded", func(t *testing.T) {
		t.Parallel()
		boom := fmt.Errorf("git exploded")
		lookup := stubLookup(t, map[string]response{
			"rev-parse --show-toplevel": {err: boom},
		})
		_, err := gitctx.Resolve("_", "main", lookup)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("lookup error on detached HEAD hash is forwarded", func(t *testing.T) {
		t.Parallel()
		boom := fmt.Errorf("no HEAD")
		lookup := stubLookup(t, map[string]response{
			"rev-parse --show-toplevel": {out: "/projects/repo"},
			"branch --show-current":     {out: ""},
			"rev-parse --short HEAD":    {err: boom},
		})
		_, err := gitctx.Resolve("_", "_", lookup)
		assert.ErrorIs(t, err, boom)
	})
}
