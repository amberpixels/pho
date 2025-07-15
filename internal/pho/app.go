package pho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"pho/internal/diff"
	"pho/internal/hashing"
	"pho/internal/render"
	"pho/internal/restore"
	"pho/pkg/jsonl"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	phoSessionConf    = "session.conf"         // Session config file
	phoDumpBase       = "_dump"                // Base filename without extension
	connectionTimeout = 500 * time.Millisecond // Timeout for connection preflight check
	sessionsSubDir    = "sessions"             // Sessions subdirectory in config dir
)

var (
	ErrNoMeta = errors.New("meta file is missing")
	ErrNoDump = errors.New("dump file is missing")
)

// App represents the Pho app.
type App struct {
	uri            string
	dbName         string
	collectionName string

	dbClient *mongo.Client

	render *render.Renderer
}

// getPhoDataDir returns the directory for storing temporary data files.
// Defaults to /tmp/pho-$USER but can be overridden with PHO_DATA_DIR environment variable.
func getPhoDataDir() (string, error) {
	if envDir := os.Getenv("PHO_DATA_DIR"); envDir != "" {
		return envDir, nil
	}

	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	return filepath.Join("/tmp", "pho-"+currentUser.Username), nil
}

// getPhoConfigDir returns the directory for storing configuration and session registry.
// Defaults to ~/.pho but can be overridden with PHO_CONFIG_DIR environment variable.
func getPhoConfigDir() (string, error) {
	if envDir := os.Getenv("PHO_CONFIG_DIR"); envDir != "" {
		return envDir, nil
	}

	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	return filepath.Join(currentUser.HomeDir, ".pho"), nil
}

// getSessionsDir returns the directory for storing session registry files.
func getSessionsDir() (string, error) {
	configDir, err := getPhoConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, sessionsSubDir), nil
}

// NewApp creates a new Pho app with the given options.
func NewApp(opts ...Option) *App {
	c := &App{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// getDumpFileExtension determines the appropriate file extension based on renderer configuration.
func (app *App) getDumpFileExtension() string {
	config := app.render.GetConfiguration()

	// If output is valid JSON (array format), use .json extension
	if config.AsValidJSON {
		return ".json"
	}

	// For compact JSON (line-by-line) or default, use .jsonl extension
	return ".jsonl"
}

// getDumpFilename returns the complete dump filename with appropriate extension.
func (app *App) getDumpFilename() string {
	return phoDumpBase + app.getDumpFileExtension()
}

// ConnectDB establishes the connection to the MongoDB server.
func (app *App) ConnectDB(ctx context.Context) error {
	if app.uri == "" {
		return errors.New("URI is required")
	}
	if app.dbName == "" {
		return errors.New("DB Name is required")
	}
	if app.collectionName == "" {
		return errors.New("collection name is required")
	}

	clientOpts := options.Client().
		ApplyURI(app.uri).
		SetServerSelectionTimeout(connectionTimeout).
		SetConnectTimeout(connectionTimeout)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return err
	}

	// Perform preflight connection check with shorter timeout
	pingCtx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		// Close the client if ping fails
		_ = client.Disconnect(ctx)
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	app.dbClient = client
	return nil
}

// ConnectDBForApply connects to database using metadata if available, otherwise uses app configuration.
func (app *App) ConnectDBForApply(ctx context.Context) error {
	// Try to read metadata to get connection details
	metadata, err := app.readMeta(ctx)
	if err == nil && metadata.URI != "" && metadata.Database != "" && metadata.Collection != "" {
		// Use connection details from metadata
		clientOpts := options.Client().
			ApplyURI(metadata.URI).
			SetServerSelectionTimeout(connectionTimeout).
			SetConnectTimeout(connectionTimeout)

		client, err := mongo.Connect(ctx, clientOpts)
		if err != nil {
			return fmt.Errorf("failed connecting with metadata URI: %w", err)
		}

		// Perform preflight connection check with shorter timeout
		pingCtx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
		defer cancel()

		if err := client.Ping(pingCtx, nil); err != nil {
			_ = client.Disconnect(ctx)
			return fmt.Errorf("failed to connect to MongoDB: %w", err)
		}

		app.dbClient = client

		// Update app configuration to match metadata for consistency
		app.uri = metadata.URI
		app.dbName = metadata.Database
		app.collectionName = metadata.Collection

		return nil
	}

	// Fall back to normal connection if metadata is not available or incomplete
	return app.ConnectDB(ctx)
}

// Close closes the MongoDB connection.
func (app *App) Close(ctx context.Context) error {
	if app.dbClient == nil {
		return nil
	}

	return app.dbClient.Disconnect(ctx)
}

// RunQuery executes a query against the MongoDB collection.
func (app *App) RunQuery(
	ctx context.Context,
	query string,
	limit int64,
	sort string,
	projection string,
) (*mongo.Cursor, error) {
	if app.dbClient == nil {
		return nil, errors.New("db not connected")
	}

	col := app.dbClient.Database(app.dbName).Collection(app.collectionName)

	// Build MongoDB options based on flags
	findOptions := options.Find()
	if limit > 0 {
		findOptions.SetLimit(limit)
	}
	if sort != "" {
		findOptions.SetSort(parseSort(sort))
	}
	if projection != "" {
		findOptions.SetProjection(parseProjection(projection))
	}

	queryBson, err := parseQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse given query: %w", err)
	}

	// Perform MongoDB query
	cur, err := col.Find(ctx, queryBson, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to perform collection.Find: %w", err)
	}

	return cur, nil
}

// Dump dumps decoded mongo cursor into given writer
//
//	TODO: as an idea: let's add a top comment in the dump that will tell if changes were applied or not
//		e.g `// changes (if any) were not applied yet`
//		will be automatically updated (after --apply-changes) ->
//		`// changes (X updates, Y deletes, Z inserts, N noops) were applied`
//		This may be an overwhelming for this function, so  think how to implement this properly
func (app *App) Dump(ctx context.Context, cursor *mongo.Cursor, out io.Writer) error {
	renderCfg := app.render.GetConfiguration()

	// Collect metadata when dumping to file (not stdout)
	var metadata *ParsedMeta
	if out != os.Stdout {
		metadata = &ParsedMeta{
			URI:        app.uri,
			Database:   app.dbName,
			Collection: app.collectionName,
			Lines:      make(map[string]*hashing.HashData),
		}
	}

	lineNumber := 0
	for cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			if renderCfg.IgnoreFailures {
				continue
			}

			return fmt.Errorf("failed on decoding line [%d]: %w", lineNumber, err)
		}

		// Store hash data in metadata when dumping to file
		if metadata != nil {
			resultHashData, err := hashing.Hash(result)
			if err != nil {
				if renderCfg.IgnoreFailures {
					// TODO: reconsider and refactor
					//       that's not so accurate, as failure is on hashing part
					//       but IgnoreFailures is a flag of rendering part
					continue
				}

				return fmt.Errorf("failed on hashing line [%d]: %w", lineNumber, err)
			}
			metadata.Lines[resultHashData.GetIdentifier()] = resultHashData
		}

		resultBytes, err := app.render.FormatResult(result)
		if err != nil {
			if renderCfg.IgnoreFailures {
				continue
			}

			return fmt.Errorf("failed on formatting line [%d]: %w", lineNumber, err)
		}

		if lineNumberBytes := app.render.FormatLineNumber(lineNumber); lineNumberBytes != nil {
			resultBytes = append(lineNumberBytes, resultBytes...)
		}

		if _, err = out.Write(resultBytes); err != nil {
			if renderCfg.IgnoreFailures {
				continue
			}

			return fmt.Errorf("failed on writing a line [%d]: %w", lineNumber, err)
		}

		lineNumber++
	}

	// Write metadata file after processing all documents
	if metadata != nil {
		if err := app.writeMetadata(metadata); err != nil {
			// TODO: it should be a soft error (warning)
			//       so we still dump data, but not letting to edit it
			return fmt.Errorf("failed writing metadata: %w", err)
		}
	}

	return nil
}

// GetPhoDir returns the pho data directory path.
func (app *App) GetPhoDir() (string, error) {
	return getPhoDataDir()
}

// setupPhoDir ensures pho data directory exists or creates it.
func (app *App) setupPhoDir() error {
	phoDir, err := getPhoDataDir()
	if err != nil {
		return fmt.Errorf("could not get pho data dir: %w", err)
	}

	_, err = os.Stat(phoDir)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("could not validate pho dir: %w", err)
	}

	if err := os.MkdirAll(phoDir, 0750); err != nil {
		return fmt.Errorf("could not create pho dir: %w", err)
	}

	return nil
}

// setupConfigDir ensures pho config directory exists or creates it.
func (app *App) setupConfigDir() error {
	configDir, err := getPhoConfigDir()
	if err != nil {
		return fmt.Errorf("could not get pho config dir: %w", err)
	}

	_, err = os.Stat(configDir)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("could not validate pho config dir: %w", err)
	}

	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("could not create pho config dir: %w", err)
	}

	return nil
}

// setupSessionsDir ensures sessions directory exists or creates it.
func (app *App) setupSessionsDir() error {
	if err := app.setupConfigDir(); err != nil {
		return err
	}

	sessionsDir, err := getSessionsDir()
	if err != nil {
		return fmt.Errorf("could not get sessions dir: %w", err)
	}

	_, err = os.Stat(sessionsDir)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("could not validate sessions dir: %w", err)
	}

	if err := os.MkdirAll(sessionsDir, 0750); err != nil {
		return fmt.Errorf("could not create sessions dir: %w", err)
	}

	return nil
}

// writeMetadata writes metadata to the unified session.conf file.
func (app *App) writeMetadata(metadata *ParsedMeta) error {
	if err := app.setupPhoDir(); err != nil {
		return err
	}

	// Get data directory path
	dataDir, err := getPhoDataDir()
	if err != nil {
		return fmt.Errorf("could not get pho data dir: %w", err)
	}

	// Check if session.conf already exists to merge with existing data
	sessionPath := filepath.Join(dataDir, phoSessionConf)
	sessionConfig := &SessionConfig{}

	if data, err := os.ReadFile(sessionPath); err == nil {
		// Session config exists, parse it
		if err := sessionConfig.FromSessionConf(data); err != nil {
			return fmt.Errorf("failed to parse existing session config: %w", err)
		}
	} else {
		// No existing session config, this shouldn't happen in normal flow
		// but let's handle it gracefully by creating a minimal config
		sessionConfig = &SessionConfig{
			URI:        metadata.URI,
			Database:   metadata.Database,
			Collection: metadata.Collection,
			Lines:      make(map[string]*hashing.HashData),
		}
	}

	// Update only the metadata-specific fields, preserve session fields
	sessionConfig.URI = metadata.URI
	sessionConfig.Database = metadata.Database
	sessionConfig.Collection = metadata.Collection
	sessionConfig.Lines = metadata.Lines

	// Update document count based on the number of hash lines
	sessionConfig.DocumentCount = len(metadata.Lines)

	// Write updated session config
	data, err := sessionConfig.ToSessionConf()
	if err != nil {
		return fmt.Errorf("failed to serialize session config: %w", err)
	}

	if err := os.WriteFile(sessionPath, data, 0600); err != nil {
		return fmt.Errorf("failed writing session config file: %w", err)
	}

	return nil
}

// SetupDumpDestination sets up writer (*os.File) for dump to be written in.
func (app *App) SetupDumpDestination() (*os.File, string, error) {
	if err := app.setupPhoDir(); err != nil {
		return nil, "", err
	}

	// Get data directory path
	dataDir, err := getPhoDataDir()
	if err != nil {
		return nil, "", fmt.Errorf("could not get pho data dir: %w", err)
	}

	destinationPath := filepath.Join(dataDir, app.getDumpFilename())

	file, err := os.Create(destinationPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed creating buffer file: %w", err)
	}

	return file, destinationPath, nil
}

// OpenEditor opens file under filePath in given editor.
func (app *App) OpenEditor(editorCmd string, filePath string) error {
	// Depending on which editor is selected, we can have custom args
	// for syntax, etc

	parts := strings.Split(editorCmd, " ")
	editor := parts[0]
	commandArgs := parts[1:]

	switch editor {
	case "vim", "nvim", "vi":
		// Set syntax JSON
	default:
		// more cases
	}

	commandArgs = append(commandArgs, filePath)

	cmd := exec.Command(editor, commandArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmd.Run failed: %w", err)
	}

	return nil
}

func (app *App) readMeta(_ context.Context) (*ParsedMeta, error) {
	if err := app.setupPhoDir(); err != nil {
		return nil, err
	}

	// Get data directory path
	dataDir, err := getPhoDataDir()
	if err != nil {
		return nil, fmt.Errorf("could not get pho data dir: %w", err)
	}

	// Read from session.conf format
	sessionPath := filepath.Join(dataDir, phoSessionConf)
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("could not open: %w", ErrNoMeta)
		}
		return nil, fmt.Errorf("could not open: %w", err)
	}

	sessionConfig := &SessionConfig{}
	if err := sessionConfig.FromSessionConf(data); err != nil {
		return nil, fmt.Errorf("failed to parse session config: %w", err)
	}

	return sessionConfig.ToParsedMeta(), nil
}

func (app *App) readDump(ctx context.Context) ([]bson.M, error) {
	if err := app.setupPhoDir(); err != nil {
		return nil, err
	}

	// Get data directory path
	dataDir, err := getPhoDataDir()
	if err != nil {
		return nil, fmt.Errorf("could not get pho data dir: %w", err)
	}

	dumpFilePath := filepath.Join(dataDir, app.getDumpFilename())
	dumpReader, err := os.Open(dumpFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("could not open dump: %w", ErrNoMeta)
		}
		return nil, fmt.Errorf("could not open dump: %w", err)
	}

	// Check for context cancellation before decoding
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var results []bson.M

	// Handle different file formats based on extension
	if app.getDumpFileExtension() == ".json" {
		// For JSON array format
		var jsonArray []DumpDoc
		decoder := json.NewDecoder(dumpReader)
		if err := decoder.Decode(&jsonArray); err != nil {
			return nil, fmt.Errorf("could not decode JSON array dump: %w", err)
		}

		results = make([]bson.M, len(jsonArray))
		for i, raw := range jsonArray {
			results[i] = bson.M(raw)
		}
	} else {
		// For JSONL format (default)
		raws, err := jsonl.DecodeAll[DumpDoc](dumpReader)
		if err != nil {
			return nil, fmt.Errorf("could not decode JSONL dump: %w", err)
		}

		results = make([]bson.M, len(raws))
		for i, raw := range raws {
			results[i] = bson.M(raw)
		}
	}

	return results, nil
}

func (app *App) extractChanges(ctx context.Context) (diff.Changes, error) {
	meta, err := app.readMeta(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed on reading meta: %w", err)
	}

	dump, err := app.readDump(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed on reading dump: %w", err)
	}

	return diff.CalculateChanges(meta.Lines, dump)
}

// ReviewChanges output changes in mongo-shell format.
func (app *App) ReviewChanges(ctx context.Context) error {
	if app.collectionName == "" {
		return errors.New("collection name is required")
	}

	allChanges, err := app.extractChanges(ctx)
	if err != nil {
		if errors.Is(err, ErrNoMeta) || errors.Is(err, ErrNoDump) {
			return errors.New("no dump data to be reviewed")
		}
		return fmt.Errorf("failed on extracting changes: %w", err)
	}

	changes := allChanges.EffectiveOnes()

	_, _ = fmt.Fprintf(os.Stdout, "// Effective changes: %d\n", changes.Len())
	_, _ = fmt.Fprintf(os.Stdout, "// Noop changes: %d\n", allChanges.FilterByAction(diff.ActionsDict.Noop).Len())

	mongoShellRestorer := restore.NewMongoShellRestorer(app.collectionName)

	for _, ch := range changes {
		if mongoCmd, err := mongoShellRestorer.Build(ch); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "could not build mongo shell command: %v\n", err)
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "%s\n", mongoCmd)
		}
	}

	return nil
}

// ApplyChanges applies (executes) the changes.
func (app *App) ApplyChanges(ctx context.Context) error {
	if app.collectionName == "" {
		return errors.New("collection name is required")
	}
	if app.dbName == "" {
		return errors.New("db name is required")
	}

	col := app.dbClient.Database(app.dbName).Collection(app.collectionName)

	allChanges, err := app.extractChanges(ctx)
	if err != nil {
		if errors.Is(err, ErrNoMeta) || errors.Is(err, ErrNoDump) {
			return errors.New("no dump data to be reviewed")
		}
		return fmt.Errorf("failed on extracting changes: %w", err)
	}

	changes := allChanges.EffectiveOnes()

	// TODO: make level of verbosity an app flag

	_, _ = fmt.Fprintf(os.Stdout, "// Effective changes: %d\n", changes.Len())
	_, _ = fmt.Fprintf(os.Stdout, "// Noop changes: %d\n", allChanges.FilterByAction(diff.ActionsDict.Noop).Len())

	mongoClientRestorer := restore.NewMongoClientRestorer(col)

	var applyErrors []error
	for _, ch := range changes {
		if mongoCmd, err := mongoClientRestorer.Build(ch); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "could not build mongo shell command: %v\n", err)
			applyErrors = append(applyErrors, err)
		} else {
			err := mongoCmd(ctx)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "failed to apply change: %v\n", err)
				applyErrors = append(applyErrors, err)
			}
		}
	}

	// Only clear the session if all changes were applied successfully
	if len(applyErrors) == 0 {
		if err := app.ClearSession(ctx); err != nil {
			// This is a soft error - the changes were applied successfully
			// but we failed to clean up the session files
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to clear session after applying changes: %v\n", err)
		}
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "Session not cleared due to %d errors during application\n", len(applyErrors))
	}

	return nil
}
