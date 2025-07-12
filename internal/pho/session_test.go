package pho

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionMetadata_String(t *testing.T) {
	session := &SessionMetadata{
		Created: time.Date(2025, 1, 11, 14, 30, 0, 0, time.UTC),
		QueryParams: QueryParameters{
			Database:   "testdb",
			Collection: "users",
			Query:      `{"active": true}`,
		},
	}

	result := session.String()
	expected := "Session: testdb.users, Query: {\"active\": true}, Created: 2025-01-11 14:30:00"
	assert.Equal(t, expected, result)
}

func TestSessionMetadata_Age(t *testing.T) {
	now := time.Now()
	session := &SessionMetadata{
		Created: now.Add(-1 * time.Hour),
	}

	age := session.Age()
	assert.Greater(t, age, 59*time.Minute)
	assert.Less(t, age, 61*time.Minute)
}

func TestApp_SaveAndLoadSession(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	app := NewApp()
	ctx := context.Background()

	queryParams := QueryParameters{
		URI:        "mongodb://localhost:27017",
		Database:   "testdb",
		Collection: "users",
		Query:      `{"active": true}`,
		Limit:      1000,
		Sort:       `{"created": -1}`,
		Projection: `{"name": 1}`,
	}

	// Test saving session
	err := app.SaveSession(ctx, queryParams)
	require.NoError(t, err)

	// Verify session file exists
	sessionPath := filepath.Join(phoDir, phoSessionFile)
	assert.FileExists(t, sessionPath)

	// Test loading session
	session, err := app.LoadSession(ctx)
	require.NoError(t, err)
	require.NotNil(t, session)

	assert.Equal(t, SessionStatusActive, session.Status)
	assert.Equal(t, queryParams.Database, session.QueryParams.Database)
	assert.Equal(t, queryParams.Collection, session.QueryParams.Collection)
	assert.Equal(t, queryParams.Query, session.QueryParams.Query)
	assert.Equal(t, queryParams.Limit, session.QueryParams.Limit)
	assert.Equal(t, queryParams.Sort, session.QueryParams.Sort)
	assert.Equal(t, queryParams.Projection, session.QueryParams.Projection)
	assert.WithinDuration(t, time.Now(), session.Created, 5*time.Second)
}

func TestApp_LoadSession_NoSession(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	app := NewApp()
	ctx := context.Background()

	session, err := app.LoadSession(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no session found")
	assert.Nil(t, session)
}

func TestApp_ClearSession(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	app := NewApp()
	ctx := context.Background()

	// Create a session first
	queryParams := QueryParameters{
		Database:   "testdb",
		Collection: "users",
		Query:      "{}",
		Limit:      100,
	}
	err := app.SaveSession(ctx, queryParams)
	require.NoError(t, err)

	// Verify session exists
	sessionPath := filepath.Join(phoDir, phoSessionFile)
	assert.FileExists(t, sessionPath)

	// Clear session
	err = app.ClearSession(ctx)
	require.NoError(t, err)

	// Verify session file is removed
	assert.NoFileExists(t, sessionPath)
}

func TestApp_HasActiveSession(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	app := NewApp()
	ctx := context.Background()

	// Test no session
	hasSession, session, err := app.HasActiveSession(ctx)
	require.NoError(t, err)
	assert.False(t, hasSession)
	assert.Nil(t, session)

	// Create session
	queryParams := QueryParameters{
		Database:   "testdb",
		Collection: "users",
		Query:      "{}",
		Limit:      100,
	}
	err = app.SaveSession(ctx, queryParams)
	require.NoError(t, err)

	// Test with session
	hasSession, session, err = app.HasActiveSession(ctx)
	require.NoError(t, err)
	assert.True(t, hasSession)
	require.NotNil(t, session)
	assert.Equal(t, "testdb", session.QueryParams.Database)
}

func TestApp_UpdateSessionStatus(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	app := NewApp()
	ctx := context.Background()

	// Test updating status with no session
	err := app.UpdateSessionStatus(ctx, SessionStatusModified)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active session found")

	// Create session
	queryParams := QueryParameters{
		Database:   "testdb",
		Collection: "users",
		Query:      "{}",
		Limit:      100,
	}
	err = app.SaveSession(ctx, queryParams)
	require.NoError(t, err)

	// Update status
	err = app.UpdateSessionStatus(ctx, SessionStatusModified)
	require.NoError(t, err)

	// Verify status was updated
	session, err := app.LoadSession(ctx)
	require.NoError(t, err)
	assert.Equal(t, SessionStatusModified, session.Status)
}

func TestApp_ValidateSession(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	app := NewApp()
	ctx := context.Background()

	// Test with nil session
	err := app.ValidateSession(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session is nil")

	// Create pho directory and files
	require.NoError(t, os.Mkdir(phoDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(phoDir, "_dump.json"), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(phoDir, "_meta"), []byte("test"), 0644))

	session := &SessionMetadata{
		DumpFile: "_dump.json",
		MetaFile: "_meta",
	}

	// Test with valid session
	err = app.ValidateSession(ctx, session)
	assert.NoError(t, err)

	// Test with missing dump file
	require.NoError(t, os.Remove(filepath.Join(phoDir, "_dump.json")))
	err = app.ValidateSession(ctx, session)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session dump file missing")
}
