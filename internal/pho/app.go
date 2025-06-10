package pho

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"pho/internal/diff"
	"pho/internal/hashing"
	"pho/internal/render"
	"pho/internal/restore"
	"pho/pkg/jsonl"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	phoDir      = ".pho"
	phoMetaFile = "_meta"

	// TODO: file extension should be handled automatically
	//       it should be switched from .jsonl to .json
	//       depending on output
	//       It's important because we want to use the fact that text editors will
	//       use automatic syntax highlighting
	phoDumpFile = "_dump.jsonl"
)

var (
	ErrNoMeta = fmt.Errorf("meta file is missing")
	ErrNoDump = fmt.Errorf("dump file is missing")
)

// App represents the Pho app.
type App struct {
	uri            string
	dbName         string
	collectionName string

	dbClient *mongo.Client

	render *render.Renderer
}

// NewApp creates a new Pho app with the given options.
func NewApp(opts ...Option) *App {
	c := &App{}
	for _, opt := range opts {
		opt(c)
	}
	return c
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

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(app.uri))
	if err != nil {
		return err
	}
	app.dbClient = client
	return nil
}

// Close closes the MongoDB connection.
func (app *App) Close(ctx context.Context) error {
	if app.dbClient == nil {
		return nil
	}

	return app.dbClient.Disconnect(ctx)
}

// RunQuery executes a query against the MongoDB collection.
func (app *App) RunQuery(ctx context.Context, query string, limit int64, sort string, projection string) (*mongo.Cursor, error) {
	if app.dbClient == nil {
		return nil, fmt.Errorf("db not connected")
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

	// Create or
	var hashesFile *os.File
	if out != os.Stdout {
		var err error
		if hashesFile, err = app.setupHashDestination(); err != nil {
			// TODO: it should be a soft error (warning)
			//       so we still dump data, but not letting to edit it
			return fmt.Errorf("failed creating hashes file")
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
		if _, err := hashesFile.WriteString(resultHashData.String() + "\n"); err != nil {
			return fmt.Errorf("failed on saving hash line [%d]: %w", lineNumber, err)
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

	return nil
}

// setupPhoDir ensures .pho directory exists or creates it
func (app *App) setupPhoDir() error {
	_, err := os.Stat(phoDir)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("could not validate pho dir: %w", err)
	}

	if err := os.Mkdir(phoDir, 0755); err != nil {
		return fmt.Errorf("could not create pho dir: %w", err)
	}

	return nil
}

// setupHashDestination sets up writer (*os.File) for hashes to be written in
func (app *App) setupHashDestination() (*os.File, error) {
	if err := app.setupPhoDir(); err != nil {
		return nil, err
	}

	destinationPath := filepath.Join(phoDir, phoMetaFile)

	file, err := os.Create(destinationPath) // TODO: 0600
	if err != nil {
		return nil, fmt.Errorf("failed creating hashes file: %w", err)
	}

	return file, nil
}

// SetupDumpDestination sets up writer (*os.File) for dump to be written in
func (app *App) SetupDumpDestination() (*os.File, string, error) {
	if err := app.setupPhoDir(); err != nil {
		return nil, "", err
	}

	destinationPath := filepath.Join(phoDir, phoDumpFile)

	file, err := os.Create(destinationPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed creating buffer file: %w", err)
	}

	return file, destinationPath, nil
}

// OpenEditor opens file under filePath in given editor
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

func (app *App) readMeta() (*ParsedMeta, error) {
	// TODO: use ctx for reading

	if err := app.setupPhoDir(); err != nil {
		return nil, err
	}

	metaFilePath := filepath.Join(phoDir, phoMetaFile)
	metaReader, err := os.Open(metaFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("could not open: %w", ErrNoMeta)
		}

		return nil, fmt.Errorf("could not open: %w", err)
	}

	var iLine int
	meta := map[string]*hashing.HashData{}
	scanner := bufio.NewScanner(metaReader)
	for scanner.Scan() {
		line := scanner.Text()

		parsed, err := hashing.Parse(line)
		if err != nil {
			return nil, fmt.Errorf("corrupted line#%d: corrupted meta line: %w", iLine, err)
		}

		// TODO: lost code, did we do something with parts before?
		// parts := strings.Split(line, "|")
		// if len(parts) != 2 {}

		meta[parsed.GetIdentifier()] = parsed
		iLine++
	}

	return &ParsedMeta{Lines: meta}, nil
}

func (app *App) readDump() ([]bson.M, error) {
	// TODO: use ctx for reading

	if err := app.setupPhoDir(); err != nil {
		return nil, err
	}

	dumpFilePath := filepath.Join(phoDir, phoDumpFile)
	dumpReader, err := os.Open(dumpFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("could not open dump: %w", ErrNoMeta)
		}
		return nil, fmt.Errorf("could not open dump: %w", err)
	}

	raws, err := jsonl.DecodeAll[DumpDoc](dumpReader)
	if err != nil {
		return nil, fmt.Errorf("could not decode dump: %w", err)
	}

	results := make([]bson.M, len(raws))
	for i, raw := range raws {
		results[i] = bson.M(raw)
	}
	return results, nil
}

func (app *App) extractChanges() (diff.Changes, error) {
	meta, err := app.readMeta()
	if err != nil {
		return nil, fmt.Errorf("failed on reading meta: %w", err)
	}

	dump, err := app.readDump()
	if err != nil {
		return nil, fmt.Errorf("failed on reading dump: %w", err)
	}

	return diff.CalculateChanges(meta.Lines, dump)
}

// ReviewChanges output changes in mongo-shell format
func (app *App) ReviewChanges(ctx context.Context) error {
	if app.collectionName == "" {
		return fmt.Errorf("collection name is required")
	}

	allChanges, err := app.extractChanges()
	if err != nil {
		if errors.Is(err, ErrNoMeta) || errors.Is(err, ErrNoDump) {
			return fmt.Errorf("no dump data to be reviewed")
		}
		return fmt.Errorf("failed on extracting changes: %w", err)
	}

	changes := allChanges.EffectiveOnes()

	fmt.Println("// Effective changes: ", changes.Len())
	fmt.Println("// Noop changes: ", allChanges.FilterByAction(diff.ActionsDict.Noop).Len())

	mongoShellRestorer := restore.NewMongoShellRestorer(app.collectionName)

	for _, ch := range changes {
		if mongoCmd, err := mongoShellRestorer.Build(ch); err != nil {
			fmt.Println("could not build mongo shell command: ", err)
		} else {
			fmt.Println(mongoCmd)
		}
	}

	return nil
}

// ApplyChanges applies (executes) the changes
func (app *App) ApplyChanges(ctx context.Context) error {
	if app.collectionName == "" {
		return fmt.Errorf("collection name is required")
	}
	if app.dbName == "" {
		return fmt.Errorf("db name is required")
	}

	col := app.dbClient.Database(app.dbName).Collection(app.collectionName)

	allChanges, err := app.extractChanges()
	if err != nil {
		if errors.Is(err, ErrNoMeta) || errors.Is(err, ErrNoDump) {
			return fmt.Errorf("no dump data to be reviewed")
		}
		return fmt.Errorf("failed on extracting changes: %w", err)
	}

	changes := allChanges.EffectiveOnes()

	// TODO: make level of verbosity an app flag

	fmt.Println("// Effective changes: ", changes.Len())
	fmt.Println("// Noop changes: ", allChanges.FilterByAction(diff.ActionsDict.Noop).Len())

	mongoClientRestorer := restore.NewMongoClientRestorer(col)

	for _, ch := range changes {
		if mongoCmd, err := mongoClientRestorer.Build(ch); err != nil {
			fmt.Println("could not build mongo shell command: ", err)
		} else {
			err := mongoCmd(ctx)
			if err != nil {
				fmt.Println("failed to apply change: %w", err)
			}
		}
	}

	return nil
}
