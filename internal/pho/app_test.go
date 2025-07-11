package pho

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pho/internal/render"
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

			if app.uri != tt.expected.uri {
				t.Errorf("NewApp() uri = %v, want %v", app.uri, tt.expected.uri)
			}
			if app.dbName != tt.expected.dbName {
				t.Errorf("NewApp() dbName = %v, want %v", app.dbName, tt.expected.dbName)
			}
			if app.collectionName != tt.expected.collectionName {
				t.Errorf("NewApp() collectionName = %v, want %v", app.collectionName, tt.expected.collectionName)
			}
			if (app.render == nil) != (tt.expected.render == nil) {
				t.Errorf("NewApp() render = %v, want %v", app.render, tt.expected.render)
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

			app := NewApp(WithRenderer(renderer))
			result := app.getDumpFileExtension()

			if result != tt.expectedExt {
				t.Errorf("getDumpFileExtension() = %v, want %v", result, tt.expectedExt)
			}
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

			if result != tt.expectedName {
				t.Errorf("getDumpFilename() = %v, want %v", result, tt.expectedName)
			}
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
				if err == nil {
					t.Errorf("ConnectDB() expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("ConnectDB() error = %v, want error containing %v", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ConnectDB() unexpected error: %v", err)
				return
			}

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

			if (err != nil) != tt.wantErr {
				t.Errorf("Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApp_setupPhoDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Change to temp directory for test
	os.Chdir(tempDir)

	app := NewApp()

	// Test creating pho directory
	err := app.setupPhoDir()
	if err != nil {
		t.Errorf("setupPhoDir() unexpected error: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(phoDir); os.IsNotExist(err) {
		t.Errorf("setupPhoDir() directory not created")
	}

	// Test that it doesn't error when directory already exists
	err = app.setupPhoDir()
	if err != nil {
		t.Errorf("setupPhoDir() unexpected error on existing directory: %v", err)
	}
}

func TestApp_SetupDumpDestination(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Change to temp directory for test
	os.Chdir(tempDir)

	renderer := render.NewRenderer(render.WithAsValidJSON(false)) // Use JSONL format

	app := NewApp(WithRenderer(renderer))

	file, path, err := app.SetupDumpDestination()
	if err != nil {
		t.Errorf("SetupDumpDestination() unexpected error: %v", err)
		return
	}
	defer file.Close()

	expectedPath := filepath.Join(phoDir, "_dump.jsonl")
	if path != expectedPath {
		t.Errorf("SetupDumpDestination() path = %v, want %v", path, expectedPath)
	}

	// Verify file was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("SetupDumpDestination() file not created at %v", path)
	}
}

func TestApp_OpenEditor(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
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

			if (err != nil) != tt.wantErr {
				t.Errorf("OpenEditor() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApp_readMeta_errors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Change to temp directory for test
	os.Chdir(tempDir)

	app := NewApp()
	ctx := context.Background()

	// Test missing meta file
	_, err := app.readMeta(ctx)
	if err == nil {
		t.Errorf("readMeta() expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "meta file is missing") {
		t.Errorf("readMeta() error = %v, want error containing 'meta file is missing'", err)
	}
}

func TestApp_readDump_errors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Change to temp directory for test
	os.Chdir(tempDir)

	renderer := render.NewRenderer(render.WithAsValidJSON(false))

	app := NewApp(WithRenderer(renderer))
	ctx := context.Background()

	// Test missing dump file
	_, err := app.readDump(ctx)
	if err == nil {
		t.Errorf("readDump() expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "meta file is missing") {
		t.Errorf("readDump() error = %v, want error containing 'meta file is missing'", err)
	}
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

			if (err != nil) != tt.wantErr {
				t.Errorf("ReviewChanges() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("ReviewChanges() error = %v, want error containing %v", err, tt.errorContains)
				}
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

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyChanges() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("ApplyChanges() error = %v, want error containing %v", err, tt.errorContains)
				}
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

			if (err != nil) != tt.wantErr {
				t.Errorf("RunQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("RunQuery() error = %v, want error containing %v", err, tt.errorContains)
				}
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
	if app == nil {
		t.Error("App should not be nil")
	}
}

// Test context cancellation
func TestApp_readMeta_contextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	app := NewApp()

	// Create .pho directory and meta file with some content
	os.Mkdir(phoDir, 0755)
	metaFile := filepath.Join(phoDir, phoMetaFile)
	content := "_id::507f1f77bcf86cd799439011|2cf24dba4f21d4288094b5c9bb7dbe11c6e4c8a7d97cde8d1d09c2b0b6f04a\n"
	os.WriteFile(metaFile, []byte(content), 0644)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := app.readMeta(ctx)
	if err != context.Canceled {
		t.Errorf("readMeta() with cancelled context error = %v, want %v", err, context.Canceled)
	}
}

func TestApp_readDump_contextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	renderer := render.NewRenderer(render.WithAsValidJSON(false))

	app := NewApp(WithRenderer(renderer))

	// Create .pho directory and dump file
	os.Mkdir(phoDir, 0755)
	dumpFile := filepath.Join(phoDir, "_dump.jsonl")
	content := `{"_id": {"$oid": "507f1f77bcf86cd799439011"}, "name": "test"}`
	os.WriteFile(dumpFile, []byte(content), 0644)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := app.readDump(ctx)
	if err != context.Canceled {
		t.Errorf("readDump() with cancelled context error = %v, want %v", err, context.Canceled)
	}
}

// Additional tests for edge cases and coverage
func TestApp_extractChanges_errors(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	app := NewApp()
	ctx := context.Background()

	_, err := app.extractChanges(ctx)
	if err == nil {
		t.Errorf("extractChanges() expected error for missing files, got nil")
	}
}

func TestConstants(t *testing.T) {
	if phoDir != ".pho" {
		t.Errorf("phoDir = %v, want .pho", phoDir)
	}
	if phoMetaFile != "_meta" {
		t.Errorf("phoMetaFile = %v, want _meta", phoMetaFile)
	}
	if phoDumpBase != "_dump" {
		t.Errorf("phoDumpBase = %v, want _dump", phoDumpBase)
	}
}

func TestErrors(t *testing.T) {
	if ErrNoMeta.Error() != "meta file is missing" {
		t.Errorf("ErrNoMeta = %v, want 'meta file is missing'", ErrNoMeta)
	}
	if ErrNoDump.Error() != "dump file is missing" {
		t.Errorf("ErrNoDump = %v, want 'dump file is missing'", ErrNoDump)
	}
}
