package pho

import "pho/internal/render"

// Option represents an option for configuring the Pho client.
type Option func(*App)

// WithURI sets the MongoDB URI for Pho.
func WithURI(v string) Option { return func(c *App) { c.uri = v } }

// WithDatabase sets the MongoDB database for the Pho client.
func WithDatabase(v string) Option { return func(c *App) { c.dbName = v } }

// WithCollection sets the MongoDB collection for the Pho client.
func WithCollection(v string) Option { return func(c *App) { c.collectionName = v } }

// WithRenderer sets the Renderer instance for the Pho App.
func WithRenderer(v *render.Renderer) Option { return func(c *App) { c.render = v } }
