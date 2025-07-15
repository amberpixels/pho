package pho_test

import (
	"context"
	"os"
	"path/filepath"
	"pho/internal/pho"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionMetadata_String(t *testing.T) {
	session := &pho.SessionMetadata{
		Created: time.Date(2025, 1, 11, 14, 30, 0, 0, time.UTC),
		QueryParams: pho.QueryParameters{
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
	session := &pho.SessionMetadata{
		Created: now.Add(-1 * time.Hour),
	}

	age := session.Age()
	assert.Greater(t, age, 59*time.Minute)
	assert.Less(t, age, 61*time.Minute)
}

func TestApp_SaveAndLoadSession(t *testing.T) {
	tempDir := t.TempDir()

	// Set up isolated environment
	t.Setenv("PHO_DATA_DIR", tempDir+"/data")
	t.Setenv("PHO_CONFIG_DIR", tempDir+"/config")

	app := pho.NewApp()
	ctx := context.Background()

	queryParams := pho.QueryParameters{
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

	// Verify session.conf file exists
	dataDir := tempDir + "/data"
	sessionPath := filepath.Join(dataDir, pho.GetPhoSessionConf())
	assert.FileExists(t, sessionPath)

	// Create a mock dump file (normally created during query execution)
	dumpPath := filepath.Join(dataDir, "_dump.jsonl")
	mockData := `{"_id": {"$oid": "507f1f77bcf86cd799439011"}, "name": "test"}`
	require.NoError(t, os.WriteFile(dumpPath, []byte(mockData), 0644))

	// Test loading session
	session, err := app.LoadSession(ctx)
	require.NoError(t, err)
	require.NotNil(t, session)

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

	// Set up isolated environment
	t.Setenv("PHO_DATA_DIR", tempDir+"/data")
	t.Setenv("PHO_CONFIG_DIR", tempDir+"/config")

	app := pho.NewApp()
	ctx := context.Background()

	session, err := app.LoadSession(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no session found")
	assert.Nil(t, session)
}

func TestApp_ClearSession(t *testing.T) {
	tempDir := t.TempDir()

	// Set up isolated environment
	t.Setenv("PHO_DATA_DIR", tempDir+"/data")
	t.Setenv("PHO_CONFIG_DIR", tempDir+"/config")

	app := pho.NewApp()
	ctx := context.Background()

	// Create a session first
	queryParams := pho.QueryParameters{
		Database:   "testdb",
		Collection: "users",
		Query:      "{}",
		Limit:      100,
	}
	err := app.SaveSession(ctx, queryParams)
	require.NoError(t, err)

	// Create a mock dump file (normally created during query execution)
	dataDir := tempDir + "/data"
	dumpPath := filepath.Join(dataDir, "_dump.jsonl")
	mockData := `{"_id": {"$oid": "507f1f77bcf86cd799439011"}, "name": "test"}`
	require.NoError(t, os.WriteFile(dumpPath, []byte(mockData), 0644))

	// Verify session exists
	sessionPath := filepath.Join(dataDir, pho.GetPhoSessionConf())
	assert.FileExists(t, sessionPath)

	// Clear session
	err = app.ClearSession(ctx)
	require.NoError(t, err)

	// Verify session file is removed
	assert.NoFileExists(t, sessionPath)
}

func TestApp_HasActiveSession(t *testing.T) {
	tempDir := t.TempDir()

	// Set up isolated environment
	t.Setenv("PHO_DATA_DIR", tempDir+"/data")
	t.Setenv("PHO_CONFIG_DIR", tempDir+"/config")

	app := pho.NewApp()
	ctx := context.Background()

	// Test no session
	hasSession, session, err := app.HasActiveSession(ctx)
	require.NoError(t, err)
	assert.False(t, hasSession)
	assert.Nil(t, session)

	// Create session
	queryParams := pho.QueryParameters{
		Database:   "testdb",
		Collection: "users",
		Query:      "{}",
		Limit:      100,
	}
	err = app.SaveSession(ctx, queryParams)
	require.NoError(t, err)

	// Create a mock dump file (normally created during query execution)
	dataDir := tempDir + "/data"
	dumpPath := filepath.Join(dataDir, "_dump.jsonl")
	mockData := `{"_id": {"$oid": "507f1f77bcf86cd799439011"}, "name": "test"}`
	require.NoError(t, os.WriteFile(dumpPath, []byte(mockData), 0644))

	// Test with session
	hasSession, session, err = app.HasActiveSession(ctx)
	require.NoError(t, err)
	assert.True(t, hasSession)
	require.NotNil(t, session)
	assert.Equal(t, "testdb", session.QueryParams.Database)
}

func TestApp_ValidateSession(t *testing.T) {
	tempDir := t.TempDir()

	// Set up isolated environment
	t.Setenv("PHO_DATA_DIR", tempDir+"/data")
	t.Setenv("PHO_CONFIG_DIR", tempDir+"/config")

	app := pho.NewApp()
	ctx := context.Background()

	// Test with nil session
	err := app.ValidateSession(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session is nil")

	// Get the actual data directory for this test
	dataDir := tempDir + "/data"
	require.NoError(t, os.MkdirAll(dataDir, 0755))

	// Create dump file and session config
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "_dump.json"), []byte("{}"), 0644))
	sessionConfig := "Created: 2025-01-11T14:30:00Z\nURI: mongodb://localhost:27017\nDatabase: testdb\nCollection: users\nDumpFile: _dump.json\nDocumentCount: 1\n"
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "session.conf"), []byte(sessionConfig), 0644))

	session := &pho.SessionMetadata{
		DumpFile: "_dump.json",
	}

	// Test with valid session
	err = app.ValidateSession(ctx, session)
	assert.NoError(t, err)

	// Test with missing dump file
	require.NoError(t, os.Remove(filepath.Join(dataDir, "_dump.json")))
	err = app.ValidateSession(ctx, session)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session dump file missing")
}
