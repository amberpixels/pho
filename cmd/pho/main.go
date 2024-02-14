package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"pho/pkg/pho"
	"pho/pkg/render"
	"strings"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// TODO: handle nice --help (with list of flags + defaults)

	// Connection args are as much similar to mongodump args as possible
	uriPtr := flag.String("uri", "mongodb://localhost:27017", "MongoDB URI Connection String")
	hostPtr := flag.String("host", "", "MongoDB Hostname")
	portPtr := flag.String("port", "", "MongoDB Port")
	dbPtr := flag.String("db", "", "MongoDB database name")
	collectionPtr := flag.String("collection", "", "MongoDB collection name")
	queryPtr := flag.String("query", "{}", "MongoDB query as a JSON document")

	// todo: shorthands (-q for --query, -h for --host, etc)

	// todo: timeouts
	// todo: ensure more complex ways of mongo connection are supported

	// Other related-based args:
	limitPtr := flag.Int64("limit", 10000, "Limit for number of documents to retrieve")
	sortPtr := flag.String("sort", "", "Sort order for documents")
	projectionPtr := flag.String("projection", "", "Projection")

	editPtr := flag.String("edit", "vim", "Edit results in the editor")
	flag.StringVar(editPtr, "e", "", "Shorthand for --edit")

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

	ctx := context.Background()

	p := pho.NewApp(
		pho.WithURI(uri),
		pho.WithDatabase(*dbPtr),
		pho.WithCollection(*collectionPtr),

		pho.WithRenderer(render.NewRenderer(
			// todo: from cli args
			render.WithExtJSONMode(render.ExtJSONModes.Canonical),
			render.WithShowLineNumbers(true),
		)),
	)

	if err := p.ConnectDB(ctx); err != nil {
		return fmt.Errorf("failed on connecting to db: %w", err)
	}
	defer p.Close(ctx)

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

	out, destinationPath, err := p.SetupDumpDestination()
	if err != nil {
		return fmt.Errorf("failed on setting dump destination: %w", err)
	}
	defer out.Close()

	if err := p.Dump(ctx, cursor, out); err != nil {
		return fmt.Errorf("failed on dumping: %w", err)
	}

	if err := p.OpenEditor(*editPtr, destinationPath); err != nil {
		return fmt.Errorf("failed on opening [%s]: %w", *editPtr, err)
	}

	return nil
}
