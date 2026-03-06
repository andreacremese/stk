package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uuidPattern matches UUID v4 strings (e.g., "c1696012-49b8-475c-adce-1ba2efab359e")
var uuidPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// captureStdout runs a function and captures its stdout output.
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// setupTestDB creates a temporary test database and sets the env var.
// Returns the cleanup function.
func setupTestDB(t *testing.T) func() {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	os.Setenv("AGENT_STACK_DB_PATH", dbPath)
	return func() {
		os.Unsetenv("AGENT_STACK_DB_PATH")
	}
}

func TestShowOutputDoesNotContainUUID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Push some items first
	err := run([]string{"push", "testrepo", "main", "first note"})
	require.NoError(t, err)
	err = run([]string{"push", "testrepo", "main", "second note"})
	require.NoError(t, err)

	// Capture show output
	output := captureStdout(func() {
		err := run([]string{"show", "testrepo", "main"})
		require.NoError(t, err)
	})

	// Verify output doesn't contain UUID
	assert.NotRegexp(t, uuidPattern, output, "show output should not contain UUID")

	// Verify expected content is present
	assert.Contains(t, output, "first note", "output should contain first note")
	assert.Contains(t, output, "second note", "output should contain second note")

	// Verify output format: should start with index number
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.True(t, strings.HasPrefix(lines[0], "0  "), "first line should start with '0  '")
	assert.True(t, strings.HasPrefix(lines[1], "1  "), "second line should start with '1  '")
}

func TestPushOutputDoesNotContainUUID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	output := captureStdout(func() {
		err := run([]string{"push", "testrepo", "main", "test note"})
		require.NoError(t, err)
	})

	// Verify output doesn't contain UUID
	assert.NotRegexp(t, uuidPattern, output, "push output should not contain UUID")

	// Verify expected content is present
	assert.Contains(t, output, "pushed", "output should start with 'pushed'")
	assert.Contains(t, output, "test note", "output should contain the note text")

	// Verify output has the timestamp (RFC3339 format contains 'T' and 'Z')
	assert.Contains(t, output, "T", "output should contain RFC3339 timestamp with 'T'")
	assert.Contains(t, output, "Z", "output should contain RFC3339 timestamp with 'Z'")
}

func TestPopOutputDoesNotContainUUID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Push an item first
	err := run([]string{"push", "testrepo", "main", "item to pop"})
	require.NoError(t, err)

	output := captureStdout(func() {
		err := run([]string{"pop", "testrepo", "main"})
		require.NoError(t, err)
	})

	// Verify output doesn't contain UUID
	assert.NotRegexp(t, uuidPattern, output, "pop output should not contain UUID")

	// Verify expected content is present
	assert.Contains(t, output, "popped", "output should start with 'popped'")
	assert.Contains(t, output, "item to pop", "output should contain the note text")
}

func TestPluckOutputDoesNotContainUUID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Push some items
	err := run([]string{"push", "testrepo", "main", "first"})
	require.NoError(t, err)
	err = run([]string{"push", "testrepo", "main", "second"})
	require.NoError(t, err)
	err = run([]string{"push", "testrepo", "main", "third"})
	require.NoError(t, err)

	output := captureStdout(func() {
		err := run([]string{"pluck", "testrepo", "main", "1"})
		require.NoError(t, err)
	})

	// Verify output doesn't contain UUID
	assert.NotRegexp(t, uuidPattern, output, "pluck output should not contain UUID")

	// Verify expected content is present
	assert.Contains(t, output, "plucked", "output should start with 'plucked'")
	assert.Contains(t, output, "second", "output should contain the plucked note")
}

func TestShowOutputFormat(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Push a note with known content
	err := run([]string{"push", "testrepo", "main", "format test note"})
	require.NoError(t, err)

	output := captureStdout(func() {
		err := run([]string{"show", "testrepo", "main"})
		require.NoError(t, err)
	})

	// Expected format: "<index>  <timestamp>  <note>"
	// Example: "0  2026-03-03T21:33:41Z  format test note"
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, 1, "should have one line of output")

	parts := strings.SplitN(lines[0], "  ", 3)
	require.Len(t, parts, 3, "output should have 3 parts separated by double spaces")

	// Verify first part is index
	assert.Equal(t, "0", parts[0], "first part should be index 0")

	// Verify second part is RFC3339 timestamp
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`, parts[1], "second part should be RFC3339 timestamp")

	// Verify third part is the note
	assert.Equal(t, "format test note", parts[2], "third part should be the note")

	// Count fields - should be exactly 3
	fieldCount := len(parts)
	assert.Equal(t, 3, fieldCount, "output should have exactly 3 fields (index, timestamp, note)")
}
