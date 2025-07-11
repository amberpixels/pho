package pho

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

			assert.Equal(t, tt.uri, app.uri)
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

			assert.Equal(t, tt.dbName, app.dbName)
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

			assert.Equal(t, tt.collectionName, app.collectionName)
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

			assert.Equal(t, tt.renderer, app.render)
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

	assert.Equal(t, renderer, app.render)

	// Verify the renderer configuration is preserved
	assert.True(t, app.render.GetConfiguration().AsValidJSON)
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

	assert.Equal(t, "mongodb://localhost:27017", app.uri)
	assert.Equal(t, "testdb", app.dbName)
	assert.Equal(t, "testcoll", app.collectionName)
	assert.Equal(t, renderer, app.render)
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

	assert.Equal(t, "mongodb://second:27017", app.uri)
	assert.Equal(t, "seconddb", app.dbName)
	assert.Equal(t, "secondcoll", app.collectionName)
}

func TestOptions_emptyApp(t *testing.T) {
	// Test that options work on an empty app
	var app App

	WithURI("test://uri")(&app)
	WithDatabase("testdb")(&app)
	WithCollection("testcoll")(&app)
	WithRenderer(render.NewRenderer())(&app)

	assert.Equal(t, "test://uri", app.uri)
	assert.Equal(t, "testdb", app.dbName)
	assert.Equal(t, "testcoll", app.collectionName)
	assert.NotNil(t, app.render)
}

func TestOptions_partialApplication(t *testing.T) {
	// Test applying only some options
	app := NewApp(
		WithURI("mongodb://localhost:27017"),
		WithDatabase("testdb"),
		// Note: no collection or renderer
	)

	assert.Equal(t, "mongodb://localhost:27017", app.uri)
	assert.Equal(t, "testdb", app.dbName)
	assert.Equal(t, "", app.collectionName)
	assert.Nil(t, app.render)
}

func TestOption_typeSignature(t *testing.T) {
	// Test that Option type works as expected
	var option Option = WithURI("test")

	app := &App{}
	option(app)

	assert.Equal(t, "test", app.uri)
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

	assert.Equal(t, app2.uri, app1.uri)
	assert.Equal(t, app2.dbName, app1.dbName)
	assert.Equal(t, app2.collectionName, app1.collectionName)
}
