package pho

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"pho/internal/hashing"
	"pho/internal/render"
	"pho/pkg/extjson"
	"pho/pkg/jsonl"
	"strings"
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

func (app *App) Dump(ctx context.Context, cursor *mongo.Cursor, out io.Writer) error {
	renderCfg := app.render.GetConfiguration()

	// Create or
	var hashesFile *os.File
	if out != os.Stdout {
		var err error
		if hashesFile, err = app.setupHashDestination(); err != nil {
			// todo: it should be a soft error (warning)
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
				// todo: reconsider and refactor
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

	file, err := os.Create(destinationPath) // todo: 0600
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
func (app *App) OpenEditor(editor string, filePath string) error {

	// Depending on which editor is selected, we can have custom args
	// for syntax, etc

	commandArgs := make([]string, 0)
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

		parts := strings.Split(line, "|")
		if len(parts) != 2 {
		}
		meta[parsed.GetIdentifier()] = parsed
		iLine++
	}

	return &ParsedMeta{Lines: meta}, nil
}

func (app *App) readDump() ([]bson.M, error) {
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

	results := make([]bson.M, len(raws), len(raws))
	for i, raw := range raws {
		results[i] = bson.M(raw)
	}
	return results, nil
}

func (app *App) extractChanges() (Changes, error) {
	meta, err := app.readMeta()
	if err != nil {
		return nil, fmt.Errorf("failed on reading meta: %w", err)
	}

	dump, err := app.readDump()
	if err != nil {
		return nil, fmt.Errorf("failed on reading dump: %w", err)
	}

	changes := make(Changes, len(dump), len(dump))

	idsMet := make(map[string]struct{})
	for i, obj := range dump {
		hashData, err := hashing.Hash(obj)
		if err != nil {
			return nil, fmt.Errorf("corrupted obj[%d] could not hash: %w", i, err)
		}
		checksumAfter := hashData.GetChecksum()

		//slog.Log(context.Background(), slog.LevelInfo, "AFTER "+hashData.String())

		id := hashData.GetIdentifier()
		idsMet[id] = struct{}{}

		if hashDataBefore, ok := meta.Lines[id]; ok {
			//slog.Log(context.Background(), slog.LevelInfo, "BEFORE "+checksumBefore)

			if hashDataBefore.GetChecksum() == checksumAfter {
				changes[i] = NewChange(hashData, Actions.Noop, "")
			} else {
				identifiedBy, identifierValue := hashData.GetIdentifierParts()
				objData, _ := extjson.NewMarshaller(true).Marshal(obj)
				cmd := fmt.Sprintf(`db.getCollection("%s").updateOne({%s:"%s"},{$set:%s})`,
					"test", // todo collection name
					identifiedBy,
					identifierValue,
					string(objData),
				)
				changes[i] = NewChange(hashData, Actions.Updated, cmd)
			}
		} else {
			changes[i] = NewChange(hashData, Actions.Added, "")
		}
	}

	for existedBeforeIdentifier, hashData := range meta.Lines {
		if _, ok := idsMet[existedBeforeIdentifier]; ok {
			continue
		}

		changes = append(changes, NewChange(hashData, Actions.Deleted, ""))
	}

	return changes, nil
}

func (app *App) ReviewChanges() error {
	changes, err := app.extractChanges()
	if err != nil {
		if errors.Is(err, ErrNoMeta) || errors.Is(err, ErrNoDump) {
			return fmt.Errorf("no dump data to be reviewed")
		}
		return fmt.Errorf("failed on extracting changes: %w", err)
	}

	fmt.Println("Effective changes: ", changes.EffectiveCount())

	for _, effCh := range changes.Effective() {
		fmt.Println(effCh.hash.GetIdentifier(), " -> ", effCh.action)
	}

	for _, effCh := range changes.Effective() {
		if effCh.command != "" {
			fmt.Println(effCh.command)
		} else {
			fmt.Println("..")
		}
	}

	return nil
}
