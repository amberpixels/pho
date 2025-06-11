package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"pho/internal/pho"
	"pho/internal/render"
	"strings"
)

func main() {
	if err := Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Run() error {
	// TODO: handle nice --help (with list of flags + defaults)

	// Connection args are as much similar to mongodump args as possible
	uriPtr := flag.String("uri", "mongodb://localhost:27017", "MongoDB URI Connection String")
	hostPtr := flag.String("host", "", "MongoDB Hostname")
	portPtr := flag.String("port", "", "MongoDB Port")
	dbPtr := flag.String("db", "", "MongoDB database name")
	collectionPtr := flag.String("collection", "", "MongoDB collection name")
	queryPtr := flag.String("query", "{}", "MongoDB query as a JSON document")

	// TODO: shorthands (-q for --query, -h for --host, etc)

	// TODO: ensure more complex ways of mongo connection are supported

	// Other related-based args:
	limitPtr := flag.Int64("limit", 10000, "Limit for number of documents to retrieve")
	sortPtr := flag.String("sort", "", "Sort order for documents")
	projectionPtr := flag.String("projection", "", "Projection")

	editPtr := flag.String("edit", "vim", "Edit results in the editor")
	flag.StringVar(editPtr, "e", "", "Shorthand for --edit")

	// ExtJSON configuration flags
	extjsonModePtr := flag.String("extjson-mode", "canonical", "ExtJSON mode (canonical, relaxed, shell)")
	flag.StringVar(extjsonModePtr, "m", "canonical", "Shorthand for --extjson-mode")
	compactJSONPtr := flag.Bool("compact", false, "Use compact JSON output")
	flag.BoolVar(compactJSONPtr, "c", false, "Shorthand for --compact")
	lineNumbersPtr := flag.Bool("line-numbers", true, "Show line numbers in output")
	flag.BoolVar(lineNumbersPtr, "l", true, "Shorthand for --line-numbers")

	reviewChangesPtr := flag.Bool("review-changes", false, "Review changes")
	applyChangesPtr := flag.Bool("apply-changes", false, "Apply changes")

	flag.Parse()

	// Parse and validate ExtJSON mode
	var extjsonMode render.ExtJSONMode
	switch *extjsonModePtr {
	case "canonical":
		extjsonMode = render.ExtJSONModes.Canonical
	case "relaxed":
		extjsonMode = render.ExtJSONModes.Relaxed
	case "shell":
		extjsonMode = render.ExtJSONModes.Shell
	default:
		return fmt.Errorf("invalid extjson-mode: %s (valid options: canonical, relaxed, shell)", *extjsonModePtr)
	}

	p := pho.NewApp(
		pho.WithURI(prepareMongoURI(uriPtr, hostPtr, portPtr)),
		pho.WithDatabase(*dbPtr),
		pho.WithCollection(*collectionPtr),

		pho.WithRenderer(render.NewRenderer(
			render.WithExtJSONMode(extjsonMode),
			render.WithShowLineNumbers(*lineNumbersPtr),
			render.WithCompactJSON(*compactJSONPtr),
		)),
	)

	// Ctx respects OS signals
	// TODO: handle timeouts (maybe?)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	switch true {
	case *reviewChangesPtr:
		if err := p.ReviewChanges(ctx); err != nil {
			return fmt.Errorf("failed on reviewing changes: %w", err)
		}

		return nil
	case *applyChangesPtr:
		// For apply changes, we need to connect to the database using stored metadata
		// or command line parameters if metadata is not available
		if err := p.ConnectDBForApply(ctx); err != nil {
			return fmt.Errorf("failed on connecting to db for apply: %w", err)
		}
		defer p.Close(ctx)

		if err := p.ApplyChanges(ctx); err != nil {
			return fmt.Errorf("failed on applying changes: %w", err)
		}

		return nil
	default:
		// Normal query mode - connect with provided parameters
		if err := p.ConnectDB(ctx); err != nil {
			return fmt.Errorf("failed on connecting to db: %w", err)
		}
		defer p.Close(ctx)
	}

	cursor, err := p.RunQuery(ctx, *queryPtr, *limitPtr, *sortPtr, *projectionPtr)
	if err != nil {
		return fmt.Errorf("failed on executing query: %w", err)
	}
	defer cursor.Close(ctx)

	// When not in --edit mode, simply output to stdout
	if *editPtr == "" {
		if err := p.Dump(ctx, cursor, os.Stdout); err != nil {
			return fmt.Errorf("failed on dumping: %w", err)
		}

		return nil
	}

	// Prepare buffer file
	// editor will open this file

	out, dumpPath, err := p.SetupDumpDestination()
	if err != nil {
		return fmt.Errorf("failed on setting dump destination: %w", err)
	}
	defer out.Close()

	if err := p.Dump(ctx, cursor, out); err != nil {
		return fmt.Errorf("failed on dumping: %w", err)
	}

	if err := p.OpenEditor(*editPtr, dumpPath); err != nil {
		return fmt.Errorf("failed on opening [%s]: %w", *editPtr, err)
	}

	return nil
}

func prepareMongoURI(uriPtr, hostPtr, portPtr *string) string {
	// if nothing was specified, let's fallback to a default URI
	result := "localhost:27017"
	if *uriPtr != "" {
		result = *uriPtr
	} else if *hostPtr != "" {
		port := "27017"
		if *portPtr != "" {
			port = *portPtr
		}

		result = *hostPtr + ":" + port
	}

	if !strings.HasPrefix(result, "mongodb://") {
		result = "mongodb://" + result
	}

	return result
}
