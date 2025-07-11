package pho

import (
	"testing"

	"pho/internal/render"
)

func TestWithURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{
			name: "localhost URI",
			uri:  "mongodb://localhost:27017",
		},
		{
			name: "remote URI",
			uri:  "mongodb://user:pass@remote.example.com:27017/dbname",
		},
		{
			name: "atlas URI",
			uri:  "mongodb+srv://user:pass@cluster.mongodb.net/dbname",
		},
		{
			name: "empty URI",
			uri:  "",
		},
		{
			name: "complex URI with options",
			uri:  "mongodb://localhost:27017/dbname?ssl=true&replicaSet=rs0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			option := WithURI(tt.uri)
			option(app)

			if app.uri != tt.uri {
				t.Errorf("WithURI() set uri = %v, want %v", app.uri, tt.uri)
			}
		})
	}
}

func TestWithDatabase(t *testing.T) {
	tests := []struct {
		name   string
		dbName string
	}{
		{
			name:   "simple database name",
			dbName: "testdb",
		},
		{
			name:   "database with underscores",
			dbName: "test_database_name",
		},
		{
			name:   "database with hyphens",
			dbName: "test-database-name",
		},
		{
			name:   "empty database name",
			dbName: "",
		},
		{
			name:   "numeric database name",
			dbName: "db123",
		},
		{
			name:   "single character",
			dbName: "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			option := WithDatabase(tt.dbName)
			option(app)

			if app.dbName != tt.dbName {
				t.Errorf("WithDatabase() set dbName = %v, want %v", app.dbName, tt.dbName)
			}
		})
	}
}

func TestWithCollection(t *testing.T) {
	tests := []struct {
		name           string
		collectionName string
	}{
		{
			name:           "simple collection name",
			collectionName: "users",
		},
		{
			name:           "collection with underscores",
			collectionName: "user_profiles",
		},
		{
			name:           "collection with dots",
			collectionName: "analytics.events",
		},
		{
			name:           "empty collection name",
			collectionName: "",
		},
		{
			name:           "numeric collection name",
			collectionName: "collection123",
		},
		{
			name:           "long collection name",
			collectionName: "very_long_collection_name_with_many_words",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			option := WithCollection(tt.collectionName)
			option(app)

			if app.collectionName != tt.collectionName {
				t.Errorf("WithCollection() set collectionName = %v, want %v", app.collectionName, tt.collectionName)
			}
		})
	}
}

func TestWithRenderer(t *testing.T) {
	tests := []struct {
		name     string
		renderer *render.Renderer
	}{
		{
			name:     "new renderer",
			renderer: render.NewRenderer(),
		},
		{
			name:     "nil renderer",
			renderer: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			option := WithRenderer(tt.renderer)
			option(app)

			if app.render != tt.renderer {
				t.Errorf("WithRenderer() set render = %v, want %v", app.render, tt.renderer)
			}
		})
	}
}

func TestWithRenderer_configured(t *testing.T) {
	// Test with a configured renderer
	renderer := render.NewRenderer(
		render.WithAsValidJSON(true),
		render.WithCompactJSON(false),
	)

	app := &App{}
	option := WithRenderer(renderer)
	option(app)

	if app.render != renderer {
		t.Errorf("WithRenderer() set render = %v, want %v", app.render, renderer)
	}

	// Verify the renderer configuration is preserved
	if !app.render.GetConfiguration().AsValidJSON {
		t.Errorf("WithRenderer() renderer configuration not preserved")
	}
}

func TestOptions_chainable(t *testing.T) {
	// Test that options can be chained together
	renderer := render.NewRenderer()

	app := NewApp(
		WithURI("mongodb://localhost:27017"),
		WithDatabase("testdb"),
		WithCollection("testcoll"),
		WithRenderer(renderer),
	)

	if app.uri != "mongodb://localhost:27017" {
		t.Errorf("Chained options: uri = %v, want mongodb://localhost:27017", app.uri)
	}
	if app.dbName != "testdb" {
		t.Errorf("Chained options: dbName = %v, want testdb", app.dbName)
	}
	if app.collectionName != "testcoll" {
		t.Errorf("Chained options: collectionName = %v, want testcoll", app.collectionName)
	}
	if app.render != renderer {
		t.Errorf("Chained options: render = %v, want %v", app.render, renderer)
	}
}

func TestOptions_override(t *testing.T) {
	// Test that later options override earlier ones
	app := NewApp(
		WithURI("mongodb://first:27017"),
		WithURI("mongodb://second:27017"),
		WithDatabase("firstdb"),
		WithDatabase("seconddb"),
		WithCollection("firstcoll"),
		WithCollection("secondcoll"),
	)

	if app.uri != "mongodb://second:27017" {
		t.Errorf("Option override: uri = %v, want mongodb://second:27017", app.uri)
	}
	if app.dbName != "seconddb" {
		t.Errorf("Option override: dbName = %v, want seconddb", app.dbName)
	}
	if app.collectionName != "secondcoll" {
		t.Errorf("Option override: collectionName = %v, want secondcoll", app.collectionName)
	}
}

func TestOptions_emptyApp(t *testing.T) {
	// Test that options work on an empty app
	var app App

	WithURI("test://uri")(&app)
	WithDatabase("testdb")(&app)
	WithCollection("testcoll")(&app)
	WithRenderer(render.NewRenderer())(&app)

	if app.uri != "test://uri" {
		t.Errorf("Options on empty app: uri = %v, want test://uri", app.uri)
	}
	if app.dbName != "testdb" {
		t.Errorf("Options on empty app: dbName = %v, want testdb", app.dbName)
	}
	if app.collectionName != "testcoll" {
		t.Errorf("Options on empty app: collectionName = %v, want testcoll", app.collectionName)
	}
	if app.render == nil {
		t.Errorf("Options on empty app: render should not be nil")
	}
}

func TestOptions_partialApplication(t *testing.T) {
	// Test applying only some options
	app := NewApp(
		WithURI("mongodb://localhost:27017"),
		WithDatabase("testdb"),
		// Note: no collection or renderer
	)

	if app.uri != "mongodb://localhost:27017" {
		t.Errorf("Partial options: uri = %v, want mongodb://localhost:27017", app.uri)
	}
	if app.dbName != "testdb" {
		t.Errorf("Partial options: dbName = %v, want testdb", app.dbName)
	}
	if app.collectionName != "" {
		t.Errorf("Partial options: collectionName = %v, want empty", app.collectionName)
	}
	if app.render != nil {
		t.Errorf("Partial options: render = %v, want nil", app.render)
	}
}

func TestOption_typeSignature(t *testing.T) {
	// Test that Option type works as expected
	var option Option = WithURI("test")

	app := &App{}
	option(app)

	if app.uri != "test" {
		t.Errorf("Option type signature: uri = %v, want test", app.uri)
	}
}

func TestOptions_orderIndependence(t *testing.T) {
	// Test that option order doesn't matter (except for overrides)
	app1 := NewApp(
		WithURI("mongodb://localhost:27017"),
		WithDatabase("testdb"),
		WithCollection("testcoll"),
	)

	app2 := NewApp(
		WithCollection("testcoll"),
		WithDatabase("testdb"),
		WithURI("mongodb://localhost:27017"),
	)

	if app1.uri != app2.uri {
		t.Errorf("Option order: app1.uri = %v, app2.uri = %v", app1.uri, app2.uri)
	}
	if app1.dbName != app2.dbName {
		t.Errorf("Option order: app1.dbName = %v, app2.dbName = %v", app1.dbName, app2.dbName)
	}
	if app1.collectionName != app2.collectionName {
		t.Errorf("Option order: app1.collectionName = %v, app2.collectionName = %v", app1.collectionName, app2.collectionName)
	}
}
