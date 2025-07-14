package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"pho/internal/logging"
	"pho/internal/pho"
	"pho/internal/render"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

const (
	defaultDocumentLimit = 10000 // Default limit for document retrieval
)

var (
	// Version is injected via ldflags during build
	Version = "dev"
)

// App represents the CLI application.
type App struct {
	cmd *cli.Command
}

// New creates a new CLI application.
func New() *App {
	return &App{
		cmd: &cli.Command{
			Name:  "pho",
			Usage: "MongoDB document editor - query, edit, and apply changes interactively",
			Description: `Pho is a MongoDB document editor that allows querying, editing, and applying changes back to MongoDB collections through an interactive editor workflow.

Core workflow:
1. Query - Connect to MongoDB and query documents with filters/projections
2. Edit - Dump documents to temporary files and open in editor (vim, etc.)
3. Review - Compare original vs edited documents to detect changes
4. Apply - Execute changes back to MongoDB or generate shell commands

Examples:
  # Query and save for later editing (default)
  pho --db mydb --collection users --query '{"active": true}'

  # Query and immediately open editor
  pho --db mydb --collection users --query '{"active": true}' --edit

  # Edit documents from previous query
  pho edit

  # Review changes after editing
  pho review

  # Apply changes to database
  pho apply`,
			Commands: []*cli.Command{
				{
					Name:    "version",
					Aliases: []string{"v"},
					Usage:   "Show version information",
					Description: "Display the current version of pho",
					Action:  versionAction,
				},
				{
					Name:    "query",
					Aliases: []string{"q"},
					Usage:   "Query MongoDB and save for later editing (default behavior)",
					Description: `Query MongoDB documents and save them for later editing.
This is the default command when no subcommand is specified. Use --edit to immediately open editor after query.`,
					Action: queryAction,
					Flags:  getCommonFlags(),
				},
				{
					Name:    "edit",
					Aliases: []string{"e"},
					Usage:   "Edit documents from previous query session",
					Description: `Open the editor with documents from the most recent query session.
Use this after running query with --edit-later flag.`,
					Action: editAction,
					Flags:  getEditFlags(),
				},
				{
					Name:    "review",
					Aliases: []string{"r"},
					Usage:   "Review changes made to documents",
					Description: `Review changes that have been made to documents in the editor.
Shows a diff of what will be changed without applying the changes.`,
					Action: reviewAction,
					Flags:  getReviewFlags(),
				},
				{
					Name:    "apply",
					Aliases: []string{"a"},
					Usage:   "Apply changes to MongoDB",
					Description: `Apply changes that have been made to documents back to MongoDB.
This will execute the actual database operations.`,
					Action: applyAction,
					Flags:  getConnectionFlags(),
				},
			},
			Flags:  getCommonFlags(),
			Action: defaultAction, // Default action when no subcommand is specified
		},
	}
}

func (a *App) GetCmd() *cli.Command { return a.cmd }

// Run executes the CLI application.
func (a *App) Run(ctx context.Context, args []string) error {
	return a.cmd.Run(ctx, args)
}

// getConnectionFlags returns flags for MongoDB connection.
func getConnectionFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "uri",
			Aliases: []string{"u"},
			Value:   "mongodb://localhost:27017",
			Usage:   "MongoDB URI Connection String",
			Sources: cli.EnvVars("MONGODB_URI"),
		},
		&cli.StringFlag{
			Name:    "host",
			Aliases: []string{"H"},
			Usage:   "MongoDB hostname (alternative to --uri)",
			Sources: cli.EnvVars("MONGODB_HOST"),
		},
		&cli.StringFlag{
			Name:    "port",
			Aliases: []string{"P"},
			Usage:   "MongoDB port (used with --host)",
			Sources: cli.EnvVars("MONGODB_PORT"),
		},
		&cli.StringFlag{
			Name:    "db",
			Aliases: []string{"d"},
			Usage:   "MongoDB database name",
			Sources: cli.EnvVars("MONGODB_DB"),
		},
		&cli.StringFlag{
			Name:    "collection",
			Aliases: []string{"c"},
			Usage:   "MongoDB collection name",
			Sources: cli.EnvVars("MONGODB_COLLECTION"),
		},
	}
}

// getEditFlags returns flags for the edit command.
func getEditFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "editor",
			Aliases: []string{"e"},
			Value:   "vim",
			Usage:   "Editor command to use for editing documents",
		},
		&cli.StringFlag{
			Name:    "extjson-mode",
			Aliases: []string{"m"},
			Value:   "canonical",
			Usage:   "ExtJSON output mode: canonical, relaxed, or shell",
		},
		&cli.BoolFlag{
			Name:    "compact",
			Aliases: []string{"C"},
			Usage:   "Use compact JSON output (no indentation)",
		},
		&cli.BoolFlag{
			Name:    "line-numbers",
			Aliases: []string{"n"},
			Value:   true,
			Usage:   "Show line numbers in output",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Enable verbose output with detailed progress information",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"Q"},
			Usage:   "Suppress all non-essential output (quiet mode)",
		},
	}
}

// getReviewFlags returns flags for the review command.
func getReviewFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "extjson-mode",
			Aliases: []string{"m"},
			Value:   "canonical",
			Usage:   "ExtJSON output mode: canonical, relaxed, or shell",
		},
		&cli.BoolFlag{
			Name:    "compact",
			Aliases: []string{"C"},
			Usage:   "Use compact JSON output (no indentation)",
		},
		&cli.BoolFlag{
			Name:    "line-numbers",
			Aliases: []string{"n"},
			Value:   true,
			Usage:   "Show line numbers in output",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Enable verbose output with detailed progress information",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"Q"},
			Usage:   "Suppress all non-essential output (quiet mode)",
		},
	}
}

// getCommonFlags returns all flags including connection and query flags.
func getCommonFlags() []cli.Flag {
	connectionFlags := getConnectionFlags()
	queryFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "query",
			Aliases: []string{"q"},
			Value:   "{}",
			Usage:   "MongoDB query as a JSON document",
		},
		&cli.Int64Flag{
			Name:    "limit",
			Aliases: []string{"l"},
			Value:   defaultDocumentLimit,
			Usage:   "Maximum number of documents to retrieve",
		},
		&cli.StringFlag{
			Name:  "sort",
			Usage: "Sort order for documents (JSON format, e.g. '{\"_id\": 1}')",
		},
		&cli.StringFlag{
			Name:  "projection",
			Usage: "Projection for documents (JSON format, e.g. '{\"field\": 1}')",
		},
		&cli.StringFlag{
			Name:    "editor",
			Aliases: []string{"e"},
			Value:   "vim",
			Usage:   "Editor command to use for editing documents",
		},
		&cli.BoolFlag{
			Name:  "edit",
			Usage: "Immediately open editor after query (combines query+edit stages)",
		},
		&cli.StringFlag{
			Name:    "extjson-mode",
			Aliases: []string{"m"},
			Value:   "canonical",
			Usage:   "ExtJSON output mode: canonical, relaxed, or shell",
		},
		&cli.BoolFlag{
			Name:    "compact",
			Aliases: []string{"C"},
			Usage:   "Use compact JSON output (no indentation)",
		},
		&cli.BoolFlag{
			Name:    "line-numbers",
			Aliases: []string{"n"},
			Value:   true,
			Usage:   "Show line numbers in output",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Enable verbose output with detailed progress information",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"Q"},
			Usage:   "Suppress all non-essential output (quiet mode)",
		},
	}

	return append(connectionFlags, queryFlags...)
}

// cliCommandInterface defines the minimal interface needed for CLI operations.
type cliCommandInterface interface {
	Bool(name string) bool
	String(name string) string
	Int64(name string) int64
}

// queryAction handles the main query and edit workflow.
func queryAction(ctx context.Context, cmd *cli.Command) error {
	// Create logger with appropriate verbosity level
	logger := createLogger(cmd)

	logger.Verbose("Starting query action with verbosity level: %s", logger.GetLevel().String())

	// Parse and validate ExtJSON mode
	extjsonModeStr := cmd.String("extjson-mode")
	logger.Debug("ExtJSON mode: %s", extjsonModeStr)
	extjsonMode, err := parseExtJSONMode(extjsonModeStr)
	if err != nil {
		logger.Error("Invalid ExtJSON mode: %s", err)
		return err
	}

	// Create pho app with configuration
	uri := prepareMongoURI(cmd.String("uri"), cmd.String("host"), cmd.String("port"))
	db := cmd.String("db")
	collection := cmd.String("collection")

	logger.Debug("Configuration: URI=%s, DB=%s, Collection=%s", uri, db, collection)
	logger.Verbose("Creating pho application instance")

	p := pho.NewApp(
		pho.WithURI(uri),
		pho.WithDatabase(db),
		pho.WithCollection(collection),
		pho.WithRenderer(render.NewRenderer(
			render.WithExtJSONMode(extjsonMode),
			render.WithShowLineNumbers(cmd.Bool("line-numbers")),
			render.WithCompactJSON(cmd.Bool("compact")),
		)),
	)

	// Setup context with signal handling
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	// Connect to database
	logger.Verbose("Connecting to MongoDB database")
	if err := p.ConnectDB(ctx); err != nil {
		logger.Error("Failed to connect to database: %s", err)
		return fmt.Errorf("failed on connecting to db: %w", err)
	}
	defer p.Close(ctx)
	logger.Success("Connected to MongoDB database")

	// Execute query
	query := cmd.String("query")
	limit := cmd.Int64("limit")
	logger.Verbose("Executing query: %s (limit: %d)", query, limit)

	cursor, err := p.RunQuery(ctx, query, limit, cmd.String("sort"), cmd.String("projection"))
	if err != nil {
		logger.Error("Query execution failed: %s", err)
		return fmt.Errorf("failed on executing query: %w", err)
	}
	defer cursor.Close(ctx)
	logger.Success("Query executed successfully")

	// Determine the workflow based on flags
	editImmediately := cmd.Bool("edit")

	// Check for existing session before starting edit workflow
	hasSession, existingSession, err := p.HasActiveSession(ctx)
	if err != nil {
		logger.Error("Failed to check for existing session: %s", err)
		return fmt.Errorf("failed to check for existing session: %w", err)
	}

	if hasSession {
		logger.Warning("Previous session found")
		fmt.Fprintf(os.Stderr, "Previous session found (created %s ago)\n", formatDuration(existingSession.Age()))
		fmt.Fprintf(os.Stderr, "Previous: db=%s collection=%s query=%s\n",
			existingSession.QueryParams.Database,
			existingSession.QueryParams.Collection,
			existingSession.QueryParams.Query)
		fmt.Fprint(os.Stderr, "Starting new session will discard previous changes. Continue? (y/N): ")

		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			logger.Info("Operation cancelled by user")
			return errors.New("operation cancelled: previous session exists")
		}

		// Clear previous session
		logger.Verbose("Clearing previous session")
		if err := p.ClearSession(ctx); err != nil {
			logger.Error("Failed to clear previous session: %s", err)
			return fmt.Errorf("failed to clear previous session: %w", err)
		}
	}

	// Setup dump destination
	logger.Verbose("Setting up dump destination")
	out, dumpPath, err := p.SetupDumpDestination()
	if err != nil {
		logger.Error("Failed to setup dump destination: %s", err)
		return fmt.Errorf("failed on setting dump destination: %w", err)
	}
	defer out.Close()
	logger.Debug("Dump file path: %s", dumpPath)

	logger.Verbose("Dumping documents to file")
	if err := p.Dump(ctx, cursor, out); err != nil {
		logger.Error("Failed to dump to file: %s", err)
		return fmt.Errorf("failed on dumping: %w", err)
	}
	logger.Success("Documents dumped to file")

	// Save session metadata after successful dump
	logger.Verbose("Saving session metadata")
	queryParams := pho.QueryParameters{
		URI:        uri,
		Database:   db,
		Collection: collection,
		Query:      query,
		Limit:      limit,
		Sort:       cmd.String("sort"),
		Projection: cmd.String("projection"),
	}

	if err := p.SaveSession(ctx, queryParams); err != nil {
		logger.Error("Failed to save session metadata: %s", err)
		return fmt.Errorf("failed to save session metadata: %w", err)
	}
	logger.Success("Session metadata saved")

	// Default behavior: save for later editing
	if !editImmediately {
		logger.Success("Query results saved for later editing. Use 'pho edit' to open editor.")
		return nil
	}

	// When --edit flag is used, immediately open editor
	editCommand := cmd.String("editor")
	logger.Verbose("Opening editor: %s", editCommand)
	if err := p.OpenEditor(editCommand, dumpPath); err != nil {
		logger.Error("Failed to open editor: %s", err)
		return fmt.Errorf("failed on opening [%s]: %w", editCommand, err)
	}
	logger.Success("Editor session completed")

	return nil
}

// editAction handles opening editor for existing session.
func editAction(ctx context.Context, cmd *cli.Command) error {
	logger := createLogger(cmd)

	logger.Verbose("Starting edit action")

	// Parse and validate ExtJSON mode (needed for renderer)
	extjsonModeStr := cmd.String("extjson-mode")
	if extjsonModeStr == "" {
		extjsonModeStr = "canonical" // default value
	}
	extjsonMode, err := parseExtJSONMode(extjsonModeStr)
	if err != nil {
		logger.Error("Invalid ExtJSON mode: %s", err)
		return err
	}

	// Create pho app with renderer configuration
	p := pho.NewApp(
		pho.WithRenderer(render.NewRenderer(
			render.WithExtJSONMode(extjsonMode),
			render.WithShowLineNumbers(cmd.Bool("line-numbers")),
			render.WithCompactJSON(cmd.Bool("compact")),
		)),
	)

	// Check if there's an active session
	hasSession, existingSession, err := p.HasActiveSession(ctx)
	if err != nil {
		logger.Error("Failed to check for existing session: %s", err)
		return fmt.Errorf("failed to check for existing session: %w", err)
	}

	if !hasSession {
		logger.Error("No active session found")
		return fmt.Errorf("no active session found. Run 'pho query --edit-later' first to create a session")
	}

	logger.Verbose("Found active session (created %s ago)", formatDuration(existingSession.Age()))
	logger.Debug("Session: db=%s collection=%s query=%s",
		existingSession.QueryParams.Database,
		existingSession.QueryParams.Collection,
		existingSession.QueryParams.Query)

	// Get existing dump file path from session (don't create new file)
	dumpPath := fmt.Sprintf("%s/%s", p.GetPhoDir(), existingSession.DumpFile)
	logger.Debug("Using existing dump file: %s", dumpPath)

	// Open editor with existing dump file
	editCommand := cmd.String("editor")
	logger.Verbose("Opening editor: %s", editCommand)
	if err := p.OpenEditor(editCommand, dumpPath); err != nil {
		logger.Error("Failed to open editor: %s", err)
		return fmt.Errorf("failed on opening [%s]: %w", editCommand, err)
	}
	logger.Success("Editor session completed")

	return nil
}

// reviewAction handles reviewing changes.
func reviewAction(ctx context.Context, cmd *cli.Command) error {
	logger := createLogger(cmd)

	logger.Verbose("Starting review action")

	// Parse and validate ExtJSON mode (needed for renderer)
	extjsonModeStr := cmd.String("extjson-mode")
	if extjsonModeStr == "" {
		extjsonModeStr = "canonical" // default value
	}
	extjsonMode, err := parseExtJSONMode(extjsonModeStr)
	if err != nil {
		logger.Error("Invalid ExtJSON mode: %s", err)
		return err
	}

	// Create pho app with renderer configuration
	p := pho.NewApp(
		pho.WithRenderer(render.NewRenderer(
			render.WithExtJSONMode(extjsonMode),
			render.WithShowLineNumbers(cmd.Bool("line-numbers")),
			render.WithCompactJSON(cmd.Bool("compact")),
		)),
	)

	// Check if there's an active session and load metadata
	hasSession, _, err := p.HasActiveSession(ctx)
	if err != nil {
		logger.Error("Failed to check for existing session: %s", err)
		return fmt.Errorf("failed to check for existing session: %w", err)
	}

	if !hasSession {
		logger.Error("No active session found")
		return fmt.Errorf("no active session found. Run 'pho query' first to create a session")
	}

	// Load session metadata to configure the app
	if err := p.ConnectDBForApply(ctx); err != nil {
		logger.Error("Failed to load session configuration: %s", err)
		return fmt.Errorf("failed to load session configuration: %w", err)
	}

	logger.Verbose("Reviewing changes in documents")
	if err := p.ReviewChanges(ctx); err != nil {
		logger.Error("Failed to review changes: %s", err)
		return fmt.Errorf("failed on reviewing changes: %w", err)
	}
	logger.Success("Change review completed")
	return nil
}

// applyAction handles applying changes to MongoDB.
func applyAction(ctx context.Context, cmd *cli.Command) error {
	logger := createLogger(cmd)

	logger.Verbose("Starting apply action")

	p := pho.NewApp()

	// Check if there's an active session
	hasSession, _, err := p.HasActiveSession(ctx)
	if err != nil {
		logger.Error("Failed to check for existing session: %s", err)
		return fmt.Errorf("failed to check for existing session: %w", err)
	}

	if !hasSession {
		logger.Error("No active session found")
		return fmt.Errorf("no active session found. Run 'pho query' first to create a session")
	}

	// Setup context with signal handling
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	// Connect to database using stored session metadata
	logger.Verbose("Connecting to database for applying changes")
	if err := p.ConnectDBForApply(ctx); err != nil {
		logger.Error("Failed to connect to database for apply: %s", err)
		return fmt.Errorf("failed on connecting to db for apply: %w", err)
	}
	defer p.Close(ctx)
	logger.Success("Connected to database")

	logger.Verbose("Applying changes to MongoDB")
	if err := p.ApplyChanges(ctx); err != nil {
		logger.Error("Failed to apply changes: %s", err)
		return fmt.Errorf("failed on applying changes: %w", err)
	}
	logger.Success("Changes applied successfully")
	return nil
}

// getVerbosityLevel determines the verbosity level from CLI flags.
func getVerbosityLevel(cmd cliCommandInterface) logging.VerbosityLevel {
	verbose := cmd.Bool("verbose")
	quiet := cmd.Bool("quiet")

	// Validate conflicting flags
	if verbose && quiet {
		fmt.Fprintf(os.Stderr, "Error: --verbose and --quiet flags cannot be used together\n")
		os.Exit(1)
	}

	if quiet {
		return logging.LevelQuiet
	}
	if verbose {
		return logging.LevelVerbose
	}
	return logging.LevelNormal
}

// createLogger creates a logger with the appropriate verbosity level.
func createLogger(cmd cliCommandInterface) *logging.Logger {
	level := getVerbosityLevel(cmd)
	return logging.NewLogger(level)
}

// parseExtJSONMode validates and returns the ExtJSON mode.
func parseExtJSONMode(mode string) (render.ExtJSONMode, error) {
	switch mode {
	case "canonical":
		return render.ExtJSONModes.Canonical, nil
	case "relaxed":
		return render.ExtJSONModes.Relaxed, nil
	case "shell":
		return render.ExtJSONModes.Shell, nil
	default:
		return render.ExtJSONModes.Canonical, fmt.Errorf(
			"invalid extjson-mode: %s (valid options: canonical, relaxed, shell)",
			mode,
		)
	}
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1f hours", d.Hours())
	}
	const hoursPerDay = 24
	return fmt.Sprintf("%.1f days", d.Hours()/hoursPerDay)
}

func prepareMongoURI(uri, host, port string) string {
	// if nothing was specified, let's fallback to a default URI
	result := "localhost:27017"
	if uri != "" {
		result = uri
	} else if host != "" {
		portStr := "27017"
		if port != "" {
			portStr = port
		}

		result = host + ":" + portStr
	}

	if !strings.HasPrefix(result, "mongodb://") {
		result = "mongodb://" + result
	}

	return result
}

// versionAction handles the version command.
func versionAction(ctx context.Context, cmd *cli.Command) error {
	fmt.Printf("pho version %s\n", Version)
	return nil
}

// defaultAction handles when no subcommand is specified or unknown commands are used.
func defaultAction(ctx context.Context, cmd *cli.Command) error {
	// Check if there are any arguments beyond the program name
	args := cmd.Args()
	if args.Len() > 0 {
		// Unknown command was specified
		unknownCmd := args.First()
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n\n", unknownCmd)
		fmt.Fprintf(os.Stderr, "Run 'pho --help' for usage.\n")
		return fmt.Errorf("unknown command: %s", unknownCmd)
	}

	// No subcommand specified, check if we have enough info to run a query
	if cmd.String("db") == "" {
		fmt.Fprintf(os.Stderr, "Error: database name is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: pho [command] [options]\n\n")
		fmt.Fprintf(os.Stderr, "Available commands:\n")
		fmt.Fprintf(os.Stderr, "  query    Query MongoDB and save for editing\n")
		fmt.Fprintf(os.Stderr, "  edit     Edit documents from previous query\n")
		fmt.Fprintf(os.Stderr, "  review   Review changes made to documents\n")
		fmt.Fprintf(os.Stderr, "  apply    Apply changes to MongoDB\n")
		fmt.Fprintf(os.Stderr, "  version  Show version information\n\n")
		fmt.Fprintf(os.Stderr, "Run 'pho --help' for detailed usage information.\n")
		return fmt.Errorf("database name is required")
	}

	// If database is specified, run the query action (default behavior)
	return queryAction(ctx, cmd)
}
