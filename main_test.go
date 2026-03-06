package main

import (
	"bytes"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var uuidPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

func testDB(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.db")
}

func TestShowOutputDoesNotContainUUID(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	require.NoError(t, run(io.Discard, db, []string{"push", "testrepo", "main", "first note"}))
	require.NoError(t, run(io.Discard, db, []string{"push", "testrepo", "main", "second note"}))

	var out bytes.Buffer
	require.NoError(t, run(&out, db, []string{"show", "testrepo", "main"}))
	output := out.String()

	assert.NotRegexp(t, uuidPattern, output, "show output should not contain UUID")
	assert.Contains(t, output, "first note", "output should contain first note")
	assert.Contains(t, output, "second note", "output should contain second note")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.True(t, strings.HasPrefix(lines[0], "0  "), "first line should start with '0  '")
	assert.True(t, strings.HasPrefix(lines[1], "1  "), "second line should start with '1  '")
}

func TestPushOutputDoesNotContainUUID(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	var out bytes.Buffer
	require.NoError(t, run(&out, db, []string{"push", "testrepo", "main", "test note"}))
	output := out.String()

	assert.NotRegexp(t, uuidPattern, output, "push output should not contain UUID")
	assert.Contains(t, output, "pushed", "output should contain 'pushed'")
	assert.Contains(t, output, "test note", "output should contain the note text")
	assert.Regexp(t, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`, output, "output should contain RFC3339 timestamp")
}

func TestPopOutputDoesNotContainUUID(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	require.NoError(t, run(io.Discard, db, []string{"push", "testrepo", "main", "item to pop"}))

	var out bytes.Buffer
	require.NoError(t, run(&out, db, []string{"pop", "testrepo", "main"}))
	output := out.String()

	assert.NotRegexp(t, uuidPattern, output, "pop output should not contain UUID")
	assert.Contains(t, output, "popped", "output should contain 'popped'")
	assert.Contains(t, output, "item to pop", "output should contain the note text")
}

func TestPluckOutputDoesNotContainUUID(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	require.NoError(t, run(io.Discard, db, []string{"push", "testrepo", "main", "first"}))
	require.NoError(t, run(io.Discard, db, []string{"push", "testrepo", "main", "second"}))
	require.NoError(t, run(io.Discard, db, []string{"push", "testrepo", "main", "third"}))

	var out bytes.Buffer
	require.NoError(t, run(&out, db, []string{"pluck", "testrepo", "main", "1"}))
	output := out.String()

	assert.NotRegexp(t, uuidPattern, output, "pluck output should not contain UUID")
	assert.Contains(t, output, "plucked", "output should contain 'plucked'")
	assert.Contains(t, output, "second", "output should contain the plucked note")
}

func TestShowOutputFormat(t *testing.T) {
	t.Parallel()
	db := testDB(t)

	require.NoError(t, run(io.Discard, db, []string{"push", "testrepo", "main", "format test note"}))

	var out bytes.Buffer
	require.NoError(t, run(&out, db, []string{"show", "testrepo", "main"}))

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 1, "should have one line of output")

	parts := strings.SplitN(lines[0], "  ", 3)
	require.Len(t, parts, 3, "output should have 3 parts separated by double spaces")

	assert.Equal(t, "0", parts[0], "first part should be index 0")
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`, parts[1], "second part should be RFC3339 timestamp")
	assert.Equal(t, "format test note", parts[2], "third part should be the note")
	assert.Equal(t, 3, len(parts), "output should have exactly 3 fields (index, timestamp, note)")
}
