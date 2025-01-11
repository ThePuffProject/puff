<h1 style="text-align:center;"> Documentation for Puff </h1>

## Creating a Puff App

```golang
package main
import "puff"

func main(){
    app *puff.PuffApp := puff.DefaultApp()
}
```

More options are available when using puff.App and puff.Config.

```golang
config := puff.Config{
    Name: "my_app_name_here",
    Version: "v0.0.1",
    DocsURL: "/docs", // DocsURL is the page to serve the OpenAPI Schema.
}
app *puff.PuffApp := puff.App(config)
```

## Creating a Router

```golang
package main
import "puff"

func main(){
    app *puff.PuffApp := puff.DefaultApp()
    router *puff.Router := puff.NewRouter(
        Name: "my_router_name_here",
        Prefix: ""
    )
    app.Mount(router) // Including the router is mandatory to serve the router on the app.
}
```

It is possible to do `router.Mount(anotherRouter)`.

## Example Router Tree

<img src="example router structure.png"></img>

## Writing a GET Request

```golang
router.GET("/", puff.Field{}, func(c *puff.Context){
    c.SendResponse(puff.GenericResponse{Content: "Hello, world!"})
})
```

## Response Types

There are a few provided response types that you can send through `*puff.Context.SendResponse` during route handling.

### JSONResponse

```golang
package main
import "puff"

type User struct {
    Name string `json:"name"`
}

func main(){
    app := puff.DefaultApp()
    app.Get("/", "", nil, func (c *puff.Context){
        user1 := User{
            Name: "John Doe",
        }
        c.SendResponse(puff.JSONResponse{
            StatusCode: 200, // StatusCode defaults to 200 if not provided.
            Content: user1,
        })
    })
```

### HTMLResponse

```golang
c.SendResponse(puff.HTMLResponse{
    StatusCode: 200, // StatusCode defaults to 200 if not provided.
    Content: `<pre>Hello, world!</pre>`, // Content defaults to an empty string if not provided.
})
```

### FileResponse

```golang
c.SendResponse(puff.FileResponse{
        StatusCode:  200 // StatusCode defaults to 200 if not provided.
	FilePath:    "path/to/assets/image.jpg" // FilePath to file
	FileContent []byte
	ContentType string // ContentType is inferred from extenstion of
})
```

### StreamingResponse

```golang
c.SendResponse(puff.StreamingResponse{
    StatusCode: 200,
    StreamHandler: func(s *chan string){
        defer close(s) // The connection does not close until you close the channel.
        for i := range 3 {
            s <- i
            time.Sleep(5 * time.Second)
        }
    }
})
```

### RedirectResponse

```golang
c.SendResponse(puff.RedirectResponse{
    StatusCode: 308,
    To: "https://google.com"
})
```

### GenericResponse

```golang
c.SendResponse(puff.GenericResponse{
    StatusCode: 200,
    Content: "Hello, world!"
    ContentType: "text/plain"
})
```

## Input Schemas

Input schemas specify what types of inputs your application takes.

Example Usage:

```
package main
import (
    "puff"
    "fmt"
)

type HelloWorldInput struct {
    Name string `kind:"query"`
}

func main(){
    app := puff.App("Input Schemas Example")

    hello_world_input := new(HelloWorldInput)
    app.Get(path: "/", description: "greets you by name", fields: hello_world_input, func (c *Context) {
        c.SendResponse(puff.GenericResponse {
            Content: fmt.Sprintf(hello_world_input.Name)
        })
    })
}
```

The schema, `HelloWorldInput` in this example, specifies a query parameter of type string.

**IMPORTANT**: The **ENTIRE body** will be unmarshalled into any field with kind `body`. This is unlike the behavior for `header`, `cookie`, and `query`, whom all have a key value structure that will be used based on the `name`.

Niceties:

No error handling with inputs, requests will automatically be rejected.

Puff's OpenAPI generation supports the `json` tag during definition generation to specify names for fields not part of the main input schema.

More examples:

```
package main
import (
    "puff"
    "fmt"
    "time"
)

type NewUserInput struct {
    Body struct {
        // notice how the entire response structure must be passed in
        Name        string
        DateOfBirth time.Time
        Latitude    float32
        Longitude   float64
    } // notice how the kind of Body is assumed
}

func main(){
    app := puff.App("Input Schemas Example")

    new_user_input := new(NewUserInput)
    app.Get(path: "/new", description: "creates a user and greets you", fields: new_user_input, func (c *Context) {
        // ...
        c.SendResponse(puff.GenericResponse {
            Content: fmt.Sprintf("Hello, %s. Welcome!", new_user_input.Name)
        })
    })
}
```

Supported Types:

```
- bool
- int
- int8
- int16
- int32
- int64
- uint
- uint8
- uint16
- uint32
- uint64
- float32
- float64
- array
- map
- pointer
- slice
- string
- struct
```

The struct tag can take:
| Field | Required | Description | Possible Values |
| -------- | -- | -- |------- |
| name | no | overrides the name (by default its the name of the structfield) | anything |
| kind | yes | where should the parameter be found | `query`, `path`, `header`, `cookie`, `body`, `formdata` |
| description | no | a brief description of the parameter | anything |
| required | no | specifies if the field is required. defaults to true for everything except cookie | `true`, `false`|
| deprecated | no | marks field as deprecated. defaults to false. | `true`, `false`|
| format | no | the format of the parameter. | examples: `email`, `password`, `uint64`|

When passing in the input, it must be a pointer to something with the input schema as the type.

**NOTE**: The proccessing of the input schema may panic. Here's what the panic messages mean.

| Message                                              | Meaning                                                        |
| ---------------------------------------------------- | -------------------------------------------------------------- |
| unsupported type <>                                  | The type is not supported (see supported types).               |
| ... unexported fields                                | ALL struct field names MUST be capitalied.                     |
| field must be POINTER to structure                   | The value you passed in for the input schema is not a pointer. |
| field must be pointer to STRUCT                      | The value you passed in for the input schma is not a struct.   |
| specified kind on field <> in struct tag must be ... | The kind on the field is not a supported kind.                 |

## Middlewares

Middlewares provide many useful tools to enhance your application. Puff comes with many middlewares in the middleware package.

To install the middleware package:

`go get https://github.com/ThePuffProject/puff/middleware`

### Attaching a Middleware

```golang
package main

import (
	"github.com/ThePuffProject/puff"
	"github.com/ThePuffProject/puff/middleware"
)

func main() {
	app := puff.DefaultApp("Restaurant Microservice")
	app.Use(middleware.CSRF())
}
```

The middleware package provides many middlewares. You can view the middleware docs at [the middleware pkg documentation](https://pkg.go.dev/github.com/ThePuffProject/puff/middleware#section-documentation).

### The Middleware Standard

Each middleware should have all the following.

#### Configuration

```golang
type MyMiddlewareConfig struct {
    // include configuration fields here
}
```

The `MyMiddlewareConfig` should contain configuration fields for your middleware.

#### Default Configuration

```golang
var DefaultMyMiddlewareConfig MyMiddlewareConfig = MyMiddlewareConfig{
    // specify default configuration here
}
```

The `DefaultMyMiddlewareConfig` should completely populate MyMiddlewareConfig.

#### Middleware Function

```golang
func MyMiddleware() puff.Middleware
```

The `MyMiddleware` function should return a `puff.Middleware` without asking for a configuration. Instead, it should use the default configuration at `DefaultMyMiddlewareConfig`.

#### Middleware Function with Configuration

```golang
func MyMiddlewareWithConfig(*MyMiddlewareConfig) puff.Middleware
```

The `MyMiddlewareWithConfig` function should take in a pointer to `MyMiddlewareConfig` and return a `puff.Middleware`.

## Using Transport Layer Security

### Step 1: Obtain public and private key certificates.

#### Self-Signed Certicate

Using OpenSSL: `openssl req -new -x509 -sha256 -key private.key -out public.crt -days 3650`

It will ask you a for a new passkey and information for your certificates. When finished, there should be two files in the current working directory, public.crt and private.key.

### Step 2: Provide the path to these files in the configuration.

Assuming this directory structure:

```
🗂️ my_app
│   go.mod
│   main.go
│   private.key
│   public.crt
```

`TLSPublicKeyFile` in `puff` should be `"public.crt"`.

`TLSPrivateKeyFile` in `puff` should be `"private.key"`.

#### Usage in Default App

```golang
package main
import "puff"

func main(){
    app *puff.PuffApp := puff.DefaultApp()
    app.TLSPublicKeyFile = "public.crt"
    app.TLSPrivateKeyFile = "private.key"
}
```
