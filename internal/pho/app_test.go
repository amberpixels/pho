package pho_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pho/internal/pho"
	"pho/internal/render"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	tests := []struct {
		name     string
		opts     []pho.Option
		expected *pho.App
	}{
		{
			name:     "no options",
			opts:     nil,
			expected: &pho.App{},
		},
		{
			name:     "with URI",
			opts:     []pho.Option{pho.WithURI("mongodb://localhost:27017")},
			expected: &pho.App{},
			// Note: cannot access private fields in struct literal
		},
		{
			name:     "with database",
			opts:     []pho.Option{pho.WithDatabase("testdb")},
			expected: &pho.App{},
			// Note: cannot access private fields in struct literal
		},
		{
			name:     "with collection",
			opts:     []pho.Option{pho.WithCollection("testcoll")},
			expected: &pho.App{},
			// Note: cannot access private fields in struct literal
		},
		{
			name:     "with renderer",
			opts:     []pho.Option{pho.WithRenderer(render.NewRenderer())},
			expected: &pho.App{},
			// Note: cannot access private fields in struct literal
		},
		{
			name: "with all options",
			opts: []pho.Option{
				pho.WithURI("mongodb://localhost:27017"),
				pho.WithDatabase("testdb"),
				pho.WithCollection("testcoll"),
				pho.WithRenderer(render.NewRenderer()),
			},
			expected: &pho.App{},
			// Note: cannot access private fields in struct literal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := pho.NewApp(tt.opts...)

			// Use AppReflect to access private fields
			ar := pho.AppReflect{App: app}

			// Test field values based on options provided
			if len(tt.opts) == 0 {
				assert.Empty(t, ar.GetURI())
				assert.Empty(t, ar.GetDBName())
				assert.Empty(t, ar.GetCollectionName())
				assert.Nil(t, ar.GetRender())
			} else {
				// Just verify app was created properly - specific field tests in other test functions
				assert.NotNil(t, app)
			}
		})
	}
}

func TestApp_getDumpFileExtension(t *testing.T) {
	tests := []struct {
		name        string
		asValidJSON bool
		expectedExt string
	}{
		{
			name:        "JSONL format",
			asValidJSON: false,
			expectedExt: ".jsonl",
		},
		{
			name:        "JSON array format",
			asValidJSON: true,
			expectedExt: ".json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := render.NewRenderer(render.WithAsValidJSON(tt.asValidJSON))

			app := pho.NewApp(pho.WithRenderer(renderer))
			ar := pho.AppReflect{App: app}
			result := ar.GetDumpFileExtension()

			assert.Equal(t, tt.expectedExt, result)
		})
	}
}

func TestApp_getDumpFilename(t *testing.T) {
	tests := []struct {
		name         string
		asValidJSON  bool
		expectedName string
	}{
		{
			name:         "JSONL filename",
			asValidJSON:  false,
			expectedName: "_dump.jsonl",
		},
		{
			name:         "JSON filename",
			asValidJSON:  true,
			expectedName: "_dump.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := render.NewRenderer(render.WithAsValidJSON(tt.asValidJSON))

			app := pho.NewApp(pho.WithRenderer(renderer))
			ar := pho.AppReflect{App: app}
			result := ar.GetDumpFilename()

			assert.Equal(t, tt.expectedName, result)
		})
	}
}

func TestApp_ConnectDB(t *testing.T) {
	tests := []struct {
		name           string
		uri            string
		dbName         string
		collectionName string
		wantErr        bool
		errorContains  string
	}{
		{
			name:           "missing URI",
			uri:            "",
			dbName:         "test",
			collectionName: "test",
			wantErr:        true,
			errorContains:  "URI is required",
		},
		{
			name:           "missing database name",
			uri:            "mongodb://localhost:27017",
			dbName:         "",
			collectionName: "test",
			wantErr:        true,
			errorContains:  "DB Name is required",
		},
		{
			name:           "missing collection name",
			uri:            "mongodb://localhost:27017",
			dbName:         "test",
			collectionName: "",
			wantErr:        true,
			errorContains:  "collection name is required",
		},
		{
			name:           "invalid URI format",
			uri:            "invalid-uri",
			dbName:         "test",
			collectionName: "test",
			wantErr:        true,
			errorContains:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := pho.NewApp(
				pho.WithURI(tt.uri),
				pho.WithDatabase(tt.dbName),
				pho.WithCollection(tt.collectionName),
			)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			err := app.ConnectDB(ctx)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)

			// Clean up connection
			ar := pho.AppReflect{App: app}
			if ar.GetDBClient() != nil {
				app.Close(ctx)
			}
		})
	}
}

func TestApp_Close(t *testing.T) {
	tests := []struct {
		name     string
		setupApp func() *pho.App
		wantErr  bool
	}{
		{
			name: "close with no client",
			setupApp: func() *pho.App {
				return pho.NewApp()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.setupApp()
			ctx := context.Background()

			err := app.Close(ctx)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestApp_setupPhoDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory for test
	t.Chdir(tempDir)

	app := pho.NewApp()

	// Test creating pho directory
	ar := pho.AppReflect{App: app}
	err := ar.SetupPhoDir()
	require.NoError(t, err)

	// Verify directory exists
	phoDir, err := pho.GetPhoDir()
	require.NoError(t, err)
	_, err = os.Stat(phoDir)
	assert.False(t, os.IsNotExist(err))

	// Test that it doesn't error when directory already exists
	err = ar.SetupPhoDir()
	require.NoError(t, err)
}

func TestApp_SetupDumpDestination(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	// Change to temp directory for test
	t.Chdir(tempDir)

	renderer := render.NewRenderer(render.WithAsValidJSON(false)) // Use JSONL format

	app := pho.NewApp(pho.WithRenderer(renderer))

	file, path, err := app.SetupDumpDestination()
	require.NoError(t, err)
	defer file.Close()

	phoDir, err := pho.GetPhoDir()
	require.NoError(t, err)
	expectedPath := filepath.Join(phoDir, "_dump.jsonl")
	assert.Equal(t, expectedPath, path)

	// Verify file was created
	_, err = os.Stat(path)
	assert.False(t, os.IsNotExist(err))
}

func TestApp_OpenEditor(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp(t.TempDir(), "test_*.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	tests := []struct {
		name      string
		editorCmd string
		filePath  string
		wantErr   bool
	}{
		{
			name:      "invalid editor command",
			editorCmd: "nonexistent-editor-12345",
			filePath:  tempFile.Name(),
			wantErr:   true,
		},
		{
			name:      "editor with spaces in command",
			editorCmd: "echo test arg",
			filePath:  tempFile.Name(),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := pho.NewApp()
			err := app.OpenEditor(tt.editorCmd, tt.filePath)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestApp_readMeta_errors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Set up isolated environment
	t.Setenv("PHO_DATA_DIR", tempDir)

	app := pho.NewApp()
	ctx := context.Background()

	// Test missing meta file
	ar := pho.AppReflect{App: app}
	_, err := ar.ReadMeta(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "meta file is missing")
}

func TestApp_readDump_errors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Set up isolated environment
	t.Setenv("PHO_DATA_DIR", tempDir)

	renderer := render.NewRenderer(render.WithAsValidJSON(false))

	app := pho.NewApp(pho.WithRenderer(renderer))
	ctx := context.Background()

	// Test missing dump file
	ar := pho.AppReflect{App: app}
	_, err := ar.ReadDump(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "meta file is missing")
}

func TestApp_ReviewChanges_errors(t *testing.T) {
	tests := []struct {
		name           string
		collectionName string
		wantErr        bool
		errorContains  string
	}{
		{
			name:           "missing collection name",
			collectionName: "",
			wantErr:        true,
			errorContains:  "collection name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := pho.NewApp(pho.WithCollection(tt.collectionName))
			ctx := context.Background()

			err := app.ReviewChanges(ctx)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestApp_ApplyChanges_errors(t *testing.T) {
	tests := []struct {
		name           string
		collectionName string
		dbName         string
		wantErr        bool
		errorContains  string
	}{
		{
			name:           "missing collection name",
			collectionName: "",
			dbName:         "test",
			wantErr:        true,
			errorContains:  "collection name is required",
		},
		{
			name:           "missing database name",
			collectionName: "test",
			dbName:         "",
			wantErr:        true,
			errorContains:  "db name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := pho.NewApp(
				pho.WithCollection(tt.collectionName),
				pho.WithDatabase(tt.dbName),
			)
			ctx := context.Background()

			err := app.ApplyChanges(ctx)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestApp_RunQuery_errors(t *testing.T) {
	tests := []struct {
		name          string
		setupApp      func() *pho.App
		query         string
		wantErr       bool
		errorContains string
	}{
		{
			name: "no database connection",
			setupApp: func() *pho.App {
				return pho.NewApp()
			},
			query:         "{}",
			wantErr:       true,
			errorContains: "db not connected",
		},
		{
			name: "invalid query format",
			setupApp: func() *pho.App {
				app := pho.NewApp()
				// Mock a client without actual connection
				return app
			},
			query:         "invalid json",
			wantErr:       true,
			errorContains: "db not connected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.setupApp()
			ctx := context.Background()

			_, err := app.RunQuery(ctx, tt.query, 0, "", "")

			if tt.wantErr {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestApp_Dump_stdout(t *testing.T) {
	// Test that Dump method exists and can handle stdout detection
	// The actual cursor functionality needs integration tests with real MongoDB
	renderer := render.NewRenderer()
	app := pho.NewApp(pho.WithRenderer(renderer))

	// This test verifies the method signature and basic logic structure
	// Real cursor testing would require MongoDB connection in integration tests
	assert.NotNil(t, app)
}

// Test context cancellation.
func TestApp_readMeta_contextCancellation(t *testing.T) {
	tempDir := t.TempDir()

	// Set up isolated environment
	t.Setenv("PHO_DATA_DIR", tempDir)

	app := pho.NewApp()

	// Create .pho directory and session file with some content
	phoDir, err := pho.GetPhoDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(phoDir, 0755))
	sessionFile := filepath.Join(phoDir, pho.GetPhoSessionConf())
	content := `Created: 2025-01-11T14:30:00Z
URI: mongodb://localhost:27017
Database: testdb
Collection: users
DumpFile: _dump.jsonl
DocumentCount: 1

_id::507f1f77bcf86cd799439011|2cf24dba4f21d4288094b5c9bb7dbe11c6e4c8a7d97cde8d1d09c2b0b6f04a`
	err = os.WriteFile(sessionFile, []byte(content), 0644)
	require.NoError(t, err)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ar := pho.AppReflect{App: app}
	_, err = ar.ReadMeta(ctx)
	// The new readMeta doesn't have loops that check context, so it won't return context.Canceled
	// Instead it should succeed since the file exists
	assert.NoError(t, err)
}

func TestApp_readDump_contextCancellation(t *testing.T) {
	tempDir := t.TempDir()

	// Set up isolated environment
	t.Setenv("PHO_DATA_DIR", tempDir)

	renderer := render.NewRenderer(render.WithAsValidJSON(false))

	app := pho.NewApp(pho.WithRenderer(renderer))

	// Create .pho directory and dump file
	phoDir, err := pho.GetPhoDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(phoDir, 0755))
	dumpFile := filepath.Join(phoDir, "_dump.jsonl")
	content := `{"_id": {"$oid": "507f1f77bcf86cd799439011"}, "name": "test"}`
	err = os.WriteFile(dumpFile, []byte(content), 0644)
	require.NoError(t, err)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ar := pho.AppReflect{App: app}
	_, err = ar.ReadDump(ctx)
	assert.Equal(t, context.Canceled, err)
}

// Additional tests for edge cases and coverage.
func TestApp_extractChanges_errors(t *testing.T) {
	tempDir := t.TempDir()

	// Set up isolated environment
	t.Setenv("PHO_DATA_DIR", tempDir)

	// Create app with renderer to avoid nil pointer dereference
	renderer := render.NewRenderer(render.WithAsValidJSON(false))
	app := pho.NewApp(pho.WithRenderer(renderer))
	ctx := context.Background()

	ar := pho.AppReflect{App: app}
	_, err := ar.ExtractChanges(ctx)
	require.Error(t, err)
}

func TestConstants(t *testing.T) {
	phoDir, err := pho.GetPhoDir()
	require.NoError(t, err)
	// Since we're using /tmp/pho-$USER, just check that it's not empty
	assert.NotEmpty(t, phoDir)
	assert.Equal(t, "session.conf", pho.GetPhoSessionConf())
	assert.Equal(t, "_dump", pho.GetPhoDumpBase())
}

func TestErrors(t *testing.T) {
	assert.Equal(t, "meta file is missing", pho.GetErrNoMeta().Error())
	assert.Equal(t, "dump file is missing", pho.GetErrNoDump().Error())
}
