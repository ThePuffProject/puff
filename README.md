# Puff

An extensible, performant, and reliable framework inspired by FastAPI.

## Features

- Automatic OpenAPI documentation generation and hosting.
- Tree-style structure Routers to group APIs.
- Extensible default middlewares.
- Customizable logger with structured and prettier logging.
- Adheres to standards set by net/http, and RFC-compliant.
- Simplicity where possible and build upon the goated stdlib when possible.
  - Only has 2 external dependencies!

## Quickstart

### Installation

```bash
go get -u github.com/ThePuffProject/puff
```

Creating a new server using Puff is simple:

```golang
import "puff"

app := puff.DefaultApp("Cool Demo App")
app.Use(middleware.Tracing())
app.Use(middleware.Logging())

app.Get("/health", nil, func(c *puff.Context) {
		c.SendResponse(puff.GenericResponse{Content: "Server is Healthy!"})
})

app.ListenAndServe(":8000")
```

This will begin the application and serve the docs at `http://localhost:8000/docs`.

The DefaultApp sets up some great defaults, but you can specify a custom config with `puff.App()`.

We also recommend setting up your own logger:

```golang
app.Logger = puff.NewLogger(puff.LoggerConfig{
		Level:      slog.LevelDebug,
		Colorize:   true,
		UseJSON:    false,
		TimeFormat: time.DateTime,
})
```

##### [View the roadmap](./roadmap.md)

##### TODO: improve docs with example on creating a simple router, and contribution docs.
