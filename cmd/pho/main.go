package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"pho/internal/logging"
	"pho/internal/pho"
	"pho/internal/render"
	"strings"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "pho",
		Usage: "MongoDB document editor - query, edit, and apply changes interactively",
		Description: `Pho is a MongoDB document editor that allows querying, editing, and applying changes back to MongoDB collections through an interactive editor workflow.

Core workflow:
1. Query - Connect to MongoDB and query documents with filters/projections  
2. Edit - Dump documents to temporary files and open in editor (vim, etc.)
3. Diff - Compare original vs edited documents to detect changes
4. Apply - Execute changes back to MongoDB or generate shell commands

Examples:
  # Query and edit documents
  pho --db mydb --collection users --query '{"active": true}' --edit vim
  
  # Just output to stdout without editing
  pho --db mydb --collection users --query '{"active": true}' --edit ""
  
  # Review changes after editing
  pho review
  
  # Apply changes to database
  pho apply`,
		Commands: []*cli.Command{
			{
				Name:    "query",
				Aliases: []string{"q"},
				Usage:   "Query MongoDB and optionally edit documents",
				Description: `Query MongoDB documents and optionally open them in an editor for modification.
This is the default command when no subcommand is specified.`,
				Action: queryAction,
				Flags:  getCommonFlags(),
			},
			{
				Name:    "review",
				Aliases: []string{"r"},
				Usage:   "Review changes made to documents",
				Description: `Review changes that have been made to documents in the editor.
Shows a diff of what will be changed without applying the changes.`,
				Action: reviewAction,
				Flags:  getConnectionFlags(),
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
		Action: queryAction, // Default action when no subcommand is specified
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// getConnectionFlags returns flags for MongoDB connection
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

// getCommonFlags returns all flags including connection and query flags
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
			Value:   10000,
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
			Name:    "edit",
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

	return append(connectionFlags, queryFlags...)
}

// getVerbosityLevel determines the verbosity level from CLI flags
func getVerbosityLevel(cmd *cli.Command) logging.VerbosityLevel {
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

// createLogger creates a logger with the appropriate verbosity level
func createLogger(cmd *cli.Command) *logging.Logger {
	level := getVerbosityLevel(cmd)
	return logging.NewLogger(level)
}

// queryAction handles the main query and edit workflow
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

	editCommand := cmd.String("edit")

	// When not in --edit mode, simply output to stdout
	if editCommand == "" {
		logger.Verbose("Outputting results to stdout")
		if err := p.Dump(ctx, cursor, os.Stdout); err != nil {
			logger.Error("Failed to dump output: %s", err)
			return fmt.Errorf("failed on dumping: %w", err)
		}
		logger.Success("Results output completed")
		return nil
	}

	// Setup dump destination and open editor
	logger.Verbose("Setting up dump destination for editor")
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

	logger.Verbose("Opening editor: %s", editCommand)
	if err := p.OpenEditor(editCommand, dumpPath); err != nil {
		logger.Error("Failed to open editor: %s", err)
		return fmt.Errorf("failed on opening [%s]: %w", editCommand, err)
	}
	logger.Success("Editor session completed")

	return nil
}

// reviewAction handles reviewing changes
func reviewAction(ctx context.Context, cmd *cli.Command) error {
	logger := createLogger(cmd)
	
	logger.Verbose("Starting review action")
	
	p := pho.NewApp(
		pho.WithURI(prepareMongoURI(cmd.String("uri"), cmd.String("host"), cmd.String("port"))),
		pho.WithDatabase(cmd.String("db")),
		pho.WithCollection(cmd.String("collection")),
	)

	logger.Verbose("Reviewing changes in documents")
	if err := p.ReviewChanges(ctx); err != nil {
		logger.Error("Failed to review changes: %s", err)
		return fmt.Errorf("failed on reviewing changes: %w", err)
	}
	logger.Success("Change review completed")
	return nil
}

// applyAction handles applying changes to MongoDB
func applyAction(ctx context.Context, cmd *cli.Command) error {
	logger := createLogger(cmd)
	
	logger.Verbose("Starting apply action")
	
	p := pho.NewApp(
		pho.WithURI(prepareMongoURI(cmd.String("uri"), cmd.String("host"), cmd.String("port"))),
		pho.WithDatabase(cmd.String("db")),
		pho.WithCollection(cmd.String("collection")),
	)

	// Setup context with signal handling
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	// For apply changes, we need to connect to the database using stored metadata
	// or command line parameters if metadata is not available
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

// parseExtJSONMode validates and returns the ExtJSON mode
func parseExtJSONMode(mode string) (render.ExtJSONMode, error) {
	switch mode {
	case "canonical":
		return render.ExtJSONModes.Canonical, nil
	case "relaxed":
		return render.ExtJSONModes.Relaxed, nil
	case "shell":
		return render.ExtJSONModes.Shell, nil
	default:
		return render.ExtJSONModes.Canonical, fmt.Errorf("invalid extjson-mode: %s (valid options: canonical, relaxed, shell)", mode)
	}
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
