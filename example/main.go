package main

import (
	"fmt"

	"github.com/ThePuffProject/puff"
)

type TestInputSchema1 struct {
	ID   int    `kind:"path" description:"The ID field specifies the ID of the drink."`
	Name string `kind:"query" description:"Name creates a name for the drink."`
}

func main() {
	app := puff.DefaultApp("hello world")
	r := puff.NewRouter("untitled router", "/api")

	puff.Get(r, "/drinks/{ID}", func(ctx *puff.Context, schema *TestInputSchema1) {
		ctx.SendResponse(puff.GenericResponse{
			StatusCode:  200,
			ContentType: "text/html",
			Content:     fmt.Sprintf("<h1>drink id: %d</h1><p>%s</p>", schema.ID, schema.Name),
		})
	})

	app.IncludeRouter(r)
	app.ListenAndServe(":8080")
}
