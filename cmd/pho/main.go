package main

import (
	"context"
	"flag"
	"fmt"
	"os"
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

	// TODO: timeouts
	// TODO: ensure more complex ways of mongo connection are supported

	// Other related-based args:
	limitPtr := flag.Int64("limit", 10000, "Limit for number of documents to retrieve")
	sortPtr := flag.String("sort", "", "Sort order for documents")
	projectionPtr := flag.String("projection", "", "Projection")

	editPtr := flag.String("edit", "vim", "Edit results in the editor")
	flag.StringVar(editPtr, "e", "", "Shorthand for --edit")

	reviewChangesPtr := flag.Bool("review-changes", false, "Review changes")
	applyChangesPtr := flag.Bool("apply-changes", false, "Apply changes")

	flag.Parse()

	// if nothing was specified, let's fallback to a default URI
	uri := "localhost:27017"
	if *uriPtr != "" {
		uri = *uriPtr
	} else if *hostPtr != "" {
		port := "27017"
		if *portPtr != "" {
			port = *portPtr
		}

		uri = *hostPtr + ":" + port
	}

	if !strings.HasPrefix(uri, "mongodb://") {
		uri = "mongodb://" + uri
	}

	p := pho.NewApp(
		pho.WithURI(uri),
		pho.WithDatabase(*dbPtr),
		pho.WithCollection(*collectionPtr),

		pho.WithRenderer(render.NewRenderer(
			// TODO: from cli args
			render.WithExtJSONMode(render.ExtJSONModes.Canonical),
			render.WithShowLineNumbers(true),
		)),
	)

	// TODO(ctx): Use reasonable timeout + make it interrupt-able from CLI
	ctx := context.TODO()

	// Unless it's `--review-changes` we must connect
	if !*reviewChangesPtr {
		if err := p.ConnectDB(ctx); err != nil {
			return fmt.Errorf("failed on connecting to db: %w", err)
		}
		defer p.Close(ctx)
	}

	// For review-/apply- changes mode we need collection name as well
	// It should not be required to be passed as flag
	// Query-stage collection/db name should be stored in meta
	// TODO(db-connection-details-in-meta): implement ^^
	switch true {
	case *reviewChangesPtr:
		if err := p.ReviewChanges(ctx); err != nil {
			return fmt.Errorf("failed on reviewing changes: %w", err)
		}

		return nil
	case *applyChangesPtr:
		if err := p.ApplyChanges(ctx); err != nil {
			return fmt.Errorf("failed on reviewing changes: %w", err)
		}

		return nil
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
