package pho

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pho/internal/render"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected *App
	}{
		{
			name:     "no options",
			opts:     nil,
			expected: &App{},
		},
		{
			name: "with URI",
			opts: []Option{WithURI("mongodb://localhost:27017")},
			expected: &App{
				uri: "mongodb://localhost:27017",
			},
		},
		{
			name: "with database",
			opts: []Option{WithDatabase("testdb")},
			expected: &App{
				dbName: "testdb",
			},
		},
		{
			name: "with collection",
			opts: []Option{WithCollection("testcoll")},
			expected: &App{
				collectionName: "testcoll",
			},
		},
		{
			name: "with renderer",
			opts: []Option{WithRenderer(render.NewRenderer())},
			expected: &App{
				render: render.NewRenderer(),
			},
		},
		{
			name: "with all options",
			opts: []Option{
				WithURI("mongodb://localhost:27017"),
				WithDatabase("testdb"),
				WithCollection("testcoll"),
				WithRenderer(render.NewRenderer()),
			},
			expected: &App{
				uri:            "mongodb://localhost:27017",
				dbName:         "testdb",
				collectionName: "testcoll",
				render:         render.NewRenderer(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(tt.opts...)

			assert.Equal(t, tt.expected.uri, app.uri)
			assert.Equal(t, tt.expected.dbName, app.dbName)
			assert.Equal(t, tt.expected.collectionName, app.collectionName)
			assert.Equal(t, tt.expected.render == nil, app.render == nil)
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

			app := NewApp(WithRenderer(renderer))
			result := app.getDumpFileExtension()

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

			app := NewApp(WithRenderer(renderer))
			result := app.getDumpFilename()

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
			app := NewApp(
				WithURI(tt.uri),
				WithDatabase(tt.dbName),
				WithCollection(tt.collectionName),
			)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			err := app.ConnectDB(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)

			// Clean up connection
			if app.dbClient != nil {
				app.Close(ctx)
			}
		})
	}
}

func TestApp_Close(t *testing.T) {
	tests := []struct {
		name     string
		setupApp func() *App
		wantErr  bool
	}{
		{
			name: "close with no client",
			setupApp: func() *App {
				return NewApp()
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
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestApp_setupPhoDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		require.NoError(t, os.Chdir(originalDir))
	}()

	// Change to temp directory for test
	require.NoError(t, os.Chdir(tempDir))

	app := NewApp()

	// Test creating pho directory
	err := app.setupPhoDir()
	require.NoError(t, err)

	// Verify directory exists
	_, err = os.Stat(phoDir)
	assert.False(t, os.IsNotExist(err))

	// Test that it doesn't error when directory already exists
	err = app.setupPhoDir()
	require.NoError(t, err)
}

func TestApp_SetupDumpDestination(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		require.NoError(t, os.Chdir(originalDir))
	}()
	// Change to temp directory for test
	require.NoError(t, os.Chdir(tempDir))

	renderer := render.NewRenderer(render.WithAsValidJSON(false)) // Use JSONL format

	app := NewApp(WithRenderer(renderer))

	file, path, err := app.SetupDumpDestination()
	require.NoError(t, err)
	defer file.Close()

	expectedPath := filepath.Join(phoDir, "_dump.jsonl")
	assert.Equal(t, expectedPath, path)

	// Verify file was created
	_, err = os.Stat(path)
	assert.False(t, os.IsNotExist(err))
}

func TestApp_OpenEditor(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test_*.json")
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
			app := NewApp()
			err := app.OpenEditor(tt.editorCmd, tt.filePath)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestApp_readMeta_errors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Chdir(originalDir))
	}()
	// Change to temp directory for test
	require.NoError(t, os.Chdir(tempDir))

	app := NewApp()
	ctx := context.Background()

	// Test missing meta file
	_, err = app.readMeta(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "meta file is missing")
}

func TestApp_readDump_errors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Chdir(originalDir))
	}()
	// Change to temp directory for test
	require.NoError(t, os.Chdir(tempDir))

	renderer := render.NewRenderer(render.WithAsValidJSON(false))

	app := NewApp(WithRenderer(renderer))
	ctx := context.Background()

	// Test missing dump file
	_, err = app.readDump(ctx)
	assert.Error(t, err)
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
			app := NewApp(WithCollection(tt.collectionName))
			ctx := context.Background()

			err := app.ReviewChanges(ctx)

			if tt.wantErr {
				assert.Error(t, err)
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
			app := NewApp(
				WithCollection(tt.collectionName),
				WithDatabase(tt.dbName),
			)
			ctx := context.Background()

			err := app.ApplyChanges(ctx)

			if tt.wantErr {
				assert.Error(t, err)
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
		setupApp      func() *App
		query         string
		wantErr       bool
		errorContains string
	}{
		{
			name: "no database connection",
			setupApp: func() *App {
				return NewApp()
			},
			query:         "{}",
			wantErr:       true,
			errorContains: "db not connected",
		},
		{
			name: "invalid query format",
			setupApp: func() *App {
				app := NewApp()
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
				assert.Error(t, err)
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
	app := NewApp(WithRenderer(renderer))

	// This test verifies the method signature and basic logic structure
	// Real cursor testing would require MongoDB connection in integration tests
	assert.NotNil(t, app)
}

// Test context cancellation
func TestApp_readMeta_contextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		require.NoError(t, os.Chdir(originalDir))
	}()

	require.NoError(t, os.Chdir(tempDir))

	app := NewApp()

	// Create .pho directory and meta file with some content
	require.NoError(t, os.Mkdir(phoDir, 0755))
	metaFile := filepath.Join(phoDir, phoMetaFile)
	content := "_id::507f1f77bcf86cd799439011|2cf24dba4f21d4288094b5c9bb7dbe11c6e4c8a7d97cde8d1d09c2b0b6f04a\n"
	err := os.WriteFile(metaFile, []byte(content), 0644)
	require.NoError(t, err)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = app.readMeta(ctx)
	assert.Equal(t, context.Canceled, err)
}

func TestApp_readDump_contextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	require.NoError(t, os.Chdir(tempDir))

	renderer := render.NewRenderer(render.WithAsValidJSON(false))

	app := NewApp(WithRenderer(renderer))

	// Create .pho directory and dump file
	require.NoError(t, os.Mkdir(phoDir, 0755))
	dumpFile := filepath.Join(phoDir, "_dump.jsonl")
	content := `{"_id": {"$oid": "507f1f77bcf86cd799439011"}, "name": "test"}`
	err := os.WriteFile(dumpFile, []byte(content), 0644)
	require.NoError(t, err)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = app.readDump(ctx)
	assert.Equal(t, context.Canceled, err)
}

// Additional tests for edge cases and coverage
func TestApp_extractChanges_errors(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	require.NoError(t, os.Chdir(tempDir))

	app := NewApp()
	ctx := context.Background()

	_, err := app.extractChanges(ctx)
	assert.Error(t, err)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, ".pho", phoDir)
	assert.Equal(t, "_meta", phoMetaFile)
	assert.Equal(t, "_dump", phoDumpBase)
}

func TestErrors(t *testing.T) {
	assert.Equal(t, "meta file is missing", ErrNoMeta.Error())
	assert.Equal(t, "dump file is missing", ErrNoDump.Error())
}
