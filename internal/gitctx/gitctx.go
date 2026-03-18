// Package gitctx resolves the git repository name and current branch for the
// working directory. Resolution is done via a pluggable LookupFunc so that
// callers can inject stubs in tests instead of executing real git commands.
package gitctx

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Ctx holds the resolved repository name and branch.
type Ctx struct {
	Repo   string
	Branch string
}

// ErrNotGitRepo is returned when git context is required but the working
// directory is not inside a git repository.
var ErrNotGitRepo = errors.New("not inside a git repository")

// LookupFunc executes a git sub-command and returns the trimmed stdout.
// args are the arguments after "git" (e.g. "rev-parse", "--show-toplevel").
type LookupFunc func(args ...string) (string, error)

// Resolve returns the git Ctx for the given repo and branch values.
// A value of "" or "_" triggers resolution from git via lookup.
// Any other non-empty value is used as-is without calling git.
func Resolve(repo, branch string, lookup LookupFunc) (Ctx, error) {
	var ctx Ctx
	var err error

	if needsResolution(repo) {
		ctx.Repo, err = resolveRepo(lookup)
		if err != nil {
			return Ctx{}, err
		}
	} else {
		ctx.Repo = repo
	}

	if needsResolution(branch) {
		ctx.Branch, err = resolveBranch(lookup)
		if err != nil {
			return Ctx{}, err
		}
	} else {
		ctx.Branch = branch
	}

	return ctx, nil
}

// DefaultLookup runs git with the supplied args and returns the trimmed
// stdout. It maps a git exit-code 128 to ErrNotGitRepo.
func DefaultLookup(args ...string) (string, error) {
	// #nosec G204 -- This tool intentionally wraps git commands; args are not user-controlled in production use
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
			return "", ErrNotGitRepo
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// needsResolution reports whether v should be resolved via git.
func needsResolution(v string) bool {
	return v == "" || v == "_"
}

// resolveRepo returns the basename of the git top-level directory.
func resolveRepo(lookup LookupFunc) (string, error) {
	toplevel, err := lookup("rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolving repo: %w", err)
	}
	return filepath.Base(toplevel), nil
}

// resolveBranch returns the current branch name. When in detached-HEAD state
// (git branch --show-current returns empty), it falls back to the short
// commit hash from git rev-parse --short HEAD.
func resolveBranch(lookup LookupFunc) (string, error) {
	branch, err := lookup("branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("resolving branch: %w", err)
	}
	if branch != "" {
		return branch, nil
	}
	// Detached HEAD: --show-current returns an empty string.
	hash, err := lookup("rev-parse", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("resolving HEAD hash: %w", err)
	}
	return hash, nil
}
