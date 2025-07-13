package pho_test

import (
	"testing"

	"pho/internal/pho"
	"pho/internal/render"

	"github.com/stretchr/testify/assert"
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
			app := &pho.App{}
			option := pho.WithURI(tt.uri)
			option(app)

			ar := pho.AppReflect{App: app}
			assert.Equal(t, tt.uri, ar.GetURI())
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
			app := &pho.App{}
			option := pho.WithDatabase(tt.dbName)
			option(app)

			ar := pho.AppReflect{App: app}
			assert.Equal(t, tt.dbName, ar.GetDBName())
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
			app := &pho.App{}
			option := pho.WithCollection(tt.collectionName)
			option(app)

			ar := pho.AppReflect{App: app}
			assert.Equal(t, tt.collectionName, ar.GetCollectionName())
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
			app := &pho.App{}
			option := pho.WithRenderer(tt.renderer)
			option(app)

			ar := pho.AppReflect{App: app}
			assert.Equal(t, tt.renderer, ar.GetRender())
		})
	}
}

func TestWithRenderer_configured(t *testing.T) {
	// Test with a configured renderer
	renderer := render.NewRenderer(
		render.WithAsValidJSON(true),
		render.WithCompactJSON(false),
	)

	app := &pho.App{}
	option := pho.WithRenderer(renderer)
	option(app)

	ar := pho.AppReflect{App: app}
	assert.Equal(t, renderer, ar.GetRender())

	// Verify the renderer configuration is preserved
	assert.True(t, ar.GetRender().GetConfiguration().AsValidJSON)
}

func TestOptions_chainable(t *testing.T) {
	// Test that options can be chained together
	renderer := render.NewRenderer()

	app := pho.NewApp(
		pho.WithURI("mongodb://localhost:27017"),
		pho.WithDatabase("testdb"),
		pho.WithCollection("testcoll"),
		pho.WithRenderer(renderer),
	)

	ar := pho.AppReflect{App: app}
	assert.Equal(t, "mongodb://localhost:27017", ar.GetURI())
	assert.Equal(t, "testdb", ar.GetDBName())
	assert.Equal(t, "testcoll", ar.GetCollectionName())
	assert.Equal(t, renderer, ar.GetRender())
}

func TestOptions_override(t *testing.T) {
	// Test that later options override earlier ones
	app := pho.NewApp(
		pho.WithURI("mongodb://first:27017"),
		pho.WithURI("mongodb://second:27017"),
		pho.WithDatabase("firstdb"),
		pho.WithDatabase("seconddb"),
		pho.WithCollection("firstcoll"),
		pho.WithCollection("secondcoll"),
	)

	ar := pho.AppReflect{App: app}
	assert.Equal(t, "mongodb://second:27017", ar.GetURI())
	assert.Equal(t, "seconddb", ar.GetDBName())
	assert.Equal(t, "secondcoll", ar.GetCollectionName())
}

func TestOptions_emptyApp(t *testing.T) {
	// Test that options work on an empty app
	var app pho.App

	pho.WithURI("test://uri")(&app)
	pho.WithDatabase("testdb")(&app)
	pho.WithCollection("testcoll")(&app)
	pho.WithRenderer(render.NewRenderer())(&app)

	ar := pho.AppReflect{App: &app}
	assert.Equal(t, "test://uri", ar.GetURI())
	assert.Equal(t, "testdb", ar.GetDBName())
	assert.Equal(t, "testcoll", ar.GetCollectionName())
	assert.NotNil(t, ar.GetRender())
}

func TestOptions_partialApplication(t *testing.T) {
	// Test applying only some options
	app := pho.NewApp(
		pho.WithURI("mongodb://localhost:27017"),
		pho.WithDatabase("testdb"),
		// Note: no collection or renderer
	)

	ar := pho.AppReflect{App: app}

	assert.Equal(t, "mongodb://localhost:27017", ar.GetURI())
	assert.Equal(t, "testdb", ar.GetDBName())
	assert.Empty(t, ar.GetCollectionName())
	assert.Nil(t, ar.GetRender())
}

func TestOption_typeSignature(t *testing.T) {
	// Test that Option type works as expected
	var option = pho.WithURI("test")

	app := &pho.App{}
	option(app)

	ar := pho.AppReflect{App: app}
	assert.Equal(t, "test", ar.GetURI())
}

func TestOptions_orderIndependence(t *testing.T) {
	// Test that option order doesn't matter (except for overrides)
	app1 := pho.NewApp(
		pho.WithURI("mongodb://localhost:27017"),
		pho.WithDatabase("testdb"),
		pho.WithCollection("testcoll"),
	)

	app2 := pho.NewApp(
		pho.WithCollection("testcoll"),
		pho.WithDatabase("testdb"),
		pho.WithURI("mongodb://localhost:27017"),
	)

	ar1 := pho.AppReflect{App: app1}
	ar2 := pho.AppReflect{App: app2}

	assert.Equal(t, ar2.GetURI(), ar1.GetURI())
	assert.Equal(t, ar2.GetDBName(), ar1.GetDBName())
	assert.Equal(t, ar2.GetCollectionName(), ar1.GetCollectionName())
}
