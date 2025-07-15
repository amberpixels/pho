package pho

import (
	"context"
	"pho/internal/diff"
	"pho/internal/render"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AppReflect struct {
	*App
}

func (a *AppReflect) GetURI() string              { return a.App.uri }
func (a *AppReflect) GetDBName() string           { return a.App.dbName }
func (a *AppReflect) GetCollectionName() string   { return a.App.collectionName }
func (a *AppReflect) GetDBClient() *mongo.Client  { return a.App.dbClient }
func (a *AppReflect) GetRender() *render.Renderer { return a.App.render }

// Export helper functions for testing.

// ParseQuery parses a query string into a bson.M.
func ParseQuery(queryStr string) (bson.M, error) { return parseQuery(queryStr) }
func ParseSort(sortStr string) bson.D            { return parseSort(sortStr) }
func ParseProjection(in string) bson.D           { return parseProjection(in) }

// Export private methods for testing.
func (a *AppReflect) GetDumpFileExtension() string                      { return a.App.getDumpFileExtension() }
func (a *AppReflect) GetDumpFilename() string                           { return a.App.getDumpFilename() }
func (a *AppReflect) SetupPhoDir() error                                { return a.App.setupPhoDir() }
func (a *AppReflect) ReadMeta(ctx context.Context) (*ParsedMeta, error) { return a.App.readMeta(ctx) }
func (a *AppReflect) ReadDump(ctx context.Context) ([]bson.M, error)    { return a.App.readDump(ctx) }
func (a *AppReflect) ExtractChanges(ctx context.Context) (diff.Changes, error) {
	return a.App.extractChanges(ctx)
}

// Export constants for testing via getter functions.
func GetPhoDir() string         { return phoDir }
func GetPhoMetaFile() string    { return phoMetaFile }
func GetPhoSessionConf() string { return phoSessionConf }
func GetPhoDumpBase() string    { return phoDumpBase }
func GetPhoSessionFile() string { return phoSessionFile }

// Export errors for testing via getter functions.
func GetErrNoMeta() error { return ErrNoMeta }
func GetErrNoDump() error { return ErrNoDump }
