package puff

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// performRequest is a helper to make HTTP requests to the router.
func performRequest(r *Router, method, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	rr := httptest.NewRecorder()
	// Assign a dummy PuffApp if the router's puff field is nil, as NewContext expects it.
	// This is a minimal setup for testing purposes.
	if r.puff == nil {
		// Initialize with default config to prevent panic in c.response()
		r.puff = &PuffApp{
			Config: DefaultConfig(), // Use the DefaultConfig() helper
		}
	}
	r.ServeHTTP(rr, req)
	return rr
}

// checkResponse is a helper to check the response status code and body.
func checkResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedBodySubstring string) {
	t.Helper()
	res := rr.Result() // Get the result once
	var reqPath string

	// Safely access request path for logging, even if res.Request is nil
	if res.Request != nil && res.Request.URL != nil {
		reqPath = res.Request.URL.Path
	} else {
		// Fallback if the request context isn't fully available on the response.
		// This might happen if ServeHTTP panics very early.
		// For now, using a placeholder. Ideally, pass original path to checkResponse.
		reqPath = "(request path not available from response)"
	}

	if res.StatusCode != expectedStatus {
		t.Errorf("handler returned wrong status code for path %s: got %v want %v. Body: %s", reqPath, res.StatusCode, expectedStatus, rr.Body.String())
	}
	if expectedBodySubstring != "" && !strings.Contains(rr.Body.String(), expectedBodySubstring) {
		t.Errorf("handler returned unexpected body for path %s: got %v want substring %v", reqPath, rr.Body.String(), expectedBodySubstring)
	}
}

// TestStaticRoutes tests basic static route functionality.
func TestStaticRoutes(t *testing.T) {
	r := NewRouter("testStatic")
	r.Get("/static/path", nil, func(c *Context) { c.Text(http.StatusOK, "static path GET") })
	r.Post("/static/path", nil, func(c *Context) { c.Text(http.StatusCreated, "static path POST") })
	r.Put("/static/path", nil, func(c *Context) { c.Text(http.StatusOK, "static path PUT") })
	
	r.Get("/foo", nil, func(c *Context) { c.Text(http.StatusOK, "foo GET") })
	r.Get("/bar", nil, func(c *Context) { c.Text(http.StatusOK, "bar GET") })
	r.Get("/parent/child", nil, func(c *Context) { c.Text(http.StatusOK, "parent/child GET") })
	r.Get("/", nil, func(c *Context) { c.Text(http.StatusOK, "root GET") })


	// Test GET /static/path
	rr := performRequest(r, "GET", "/static/path", nil)
	checkResponse(t, rr, http.StatusOK, "static path GET")

	// Test POST /static/path
	rr = performRequest(r, "POST", "/static/path", nil)
	checkResponse(t, rr, http.StatusCreated, "static path POST")
	
	// Test PUT /static/path
	rr = performRequest(r, "PUT", "/static/path", nil)
	checkResponse(t, rr, http.StatusOK, "static path PUT")

	// Test /foo
	rr = performRequest(r, "GET", "/foo", nil)
	checkResponse(t, rr, http.StatusOK, "foo GET")

	// Test /bar
	rr = performRequest(r, "GET", "/bar", nil)
	checkResponse(t, rr, http.StatusOK, "bar GET")
	
	// Test /parent/child
	rr = performRequest(r, "GET", "/parent/child", nil)
	checkResponse(t, rr, http.StatusOK, "parent/child GET")

	// Test GET / (root)
	rr = performRequest(r, "GET", "/", nil)
	checkResponse(t, rr, http.StatusOK, "root GET")

	// Test path normalization (trailing slash)
	rr = performRequest(r, "GET", "/static/path/", nil) 
	checkResponse(t, rr, http.StatusOK, "static path GET")
}

// TestParameterRoutes tests routes with path parameters.
func TestParameterRoutes(t *testing.T) {
	r := NewRouter("testParams")

	// Assuming c.PathParams is populated by ServeHTTP: c.PathParams = paramsMap
	r.Get("/users/{id}", nil, func(c *Context) {
		id := c.PathParams["id"]
		c.Text(http.StatusOK, fmt.Sprintf("User ID: %s", id))
	})
	r.Get("/users/{userId}/books/{bookId}", nil, func(c *Context) {
		userId := c.PathParams["userId"]
		bookId := c.PathParams["bookId"]
		c.Text(http.StatusOK, fmt.Sprintf("User: %s, Book: %s", userId, bookId))
	})

	// Test single parameter
	rr := performRequest(r, "GET", "/users/123", nil)
	checkResponse(t, rr, http.StatusOK, "User ID: 123")

	// Test multiple parameters
	rr = performRequest(r, "GET", "/users/456/books/789", nil)
	checkResponse(t, rr, http.StatusOK, "User: 456, Book: 789")

	// Parameter vs. Static precedence
	staticRouter := NewRouter("paramVsStatic")
	staticRouter.Get("/items/special", nil, func(c *Context) {
		c.Text(http.StatusOK, "special item")
	})
	staticRouter.Get("/items/{itemId}", nil, func(c *Context) {
		itemId := c.PathParams["itemId"]
		c.Text(http.StatusOK, fmt.Sprintf("item: %s", itemId))
	})

	rr = performRequest(staticRouter, "GET", "/items/special", nil)
	checkResponse(t, rr, http.StatusOK, "special item")

	rr = performRequest(staticRouter, "GET", "/items/abc", nil)
	checkResponse(t, rr, http.StatusOK, "item: abc")
}

// TestWildcardRoutes tests wildcard route functionality.
func TestWildcardRoutes(t *testing.T) {
	r := NewRouter("testWildcard")

	// Assuming c.PathParams is populated by ServeHTTP
	r.Get("/files/*filepath", nil, func(c *Context) {
		filepath := c.PathParams["filepath"]
		c.Text(http.StatusOK, fmt.Sprintf("File: %s", filepath))
	})

	rr := performRequest(r, "GET", "/files/docs/report.txt", nil)
	checkResponse(t, rr, http.StatusOK, "File: docs/report.txt")

	rr = performRequest(r, "GET", "/files/image.png", nil)
	checkResponse(t, rr, http.StatusOK, "File: image.png")

	// Wildcard vs. Static/Parameter precedence
	precedenceRouter := NewRouter("wildcardPrecedence")
	precedenceRouter.Get("/data/config", nil, func(c *Context) {
		c.Text(http.StatusOK, "static config")
	})
	precedenceRouter.Get("/data/{resource}", nil, func(c *Context) {
		resource := c.PathParams["resource"]
		c.Text(http.StatusOK, fmt.Sprintf("resource: %s", resource))
	})
	precedenceRouter.Get("/data/*catchall", nil, func(c *Context) {
		// Note: extractParamName in ServeHTTP uses "wildcard" for anonymous "*"
		// If "catchall" is from "*{name}", then it's "catchall"
		catchall := c.PathParams["catchall"]
		c.Text(http.StatusOK, fmt.Sprintf("catchall: %s", catchall))
	})

	rr = performRequest(precedenceRouter, "GET", "/data/config", nil)
	checkResponse(t, rr, http.StatusOK, "static config")
	
	rr = performRequest(precedenceRouter, "GET", "/data/user-settings", nil)
	checkResponse(t, rr, http.StatusOK, "resource: user-settings")

	rr = performRequest(precedenceRouter, "GET", "/data/archive/old/file.txt", nil)
	checkResponse(t, rr, http.StatusOK, "catchall: archive/old/file.txt")
}

// TestNodeSplitting tests route reachability implying correct radix properties.
func TestNodeSplitting(t *testing.T) {
	r := NewRouter("testSplitting")

	r.Get("/common/prefix/route1", nil, func(c *Context) { c.Text(http.StatusOK, "route1") })
	r.Get("/common/prefix/route2", nil, func(c *Context) { c.Text(http.StatusOK, "route2") })
	r.Get("/common/another", nil, func(c *Context) { c.Text(http.StatusOK, "another") })
	r.Get("/long/path/that/is/unique", nil, func(c *Context) { c.Text(http.StatusOK, "unique long path") })

	rr := performRequest(r, "GET", "/common/prefix/route1", nil)
	checkResponse(t, rr, http.StatusOK, "route1")

	rr = performRequest(r, "GET", "/common/prefix/route2", nil)
	checkResponse(t, rr, http.StatusOK, "route2")

	rr = performRequest(r, "GET", "/common/another", nil)
	checkResponse(t, rr, http.StatusOK, "another")

	rr = performRequest(r, "GET", "/long/path/that/is/unique", nil)
	checkResponse(t, rr, http.StatusOK, "unique long path")
}

// TestMountedRouters tests functionality of mounting sub-routers.
func TestMountedRouters(t *testing.T) {
	parentRouter := NewRouter("parent")
	subRouter := NewRouter("sub")

	subRouter.Get("/resource", nil, func(c *Context) { c.Text(http.StatusOK, "sub-router resource") })
	subRouter.Get("/{id}", nil, func(c *Context) {
		idVal := c.PathParams["id"]
		c.Text(http.StatusOK, fmt.Sprintf("sub-router id: %s", idVal))
	})
	
	parentRouter.Mount("/sub", subRouter)

	rr := performRequest(parentRouter, "GET", "/sub/resource", nil)
	checkResponse(t, rr, http.StatusOK, "sub-router resource")

	rr = performRequest(parentRouter, "GET", "/sub/xyz123", nil)
	checkResponse(t, rr, http.StatusOK, "sub-router id: xyz123")

	// Test mounting at root
	rootMountParent := NewRouter("rootMountParent")
	rootSubRouter := NewRouter("rootSub")
	rootSubRouter.Get("/resource", nil, func(c *Context) { c.Text(http.StatusOK, "root sub resource") })
	
	rootMountParent.Mount("/", rootSubRouter)
	rr = performRequest(rootMountParent, "GET", "/resource", nil)
	checkResponse(t, rr, http.StatusOK, "root sub resource")

	// Test nested mounting
	grandParent := NewRouter("grandParent")
	p := NewRouter("p")
	s := NewRouter("s")
	s.Get("/final", nil, func(c *Context) { c.Text(http.StatusOK, "nested final") })
	p.Mount("/s", s)
	grandParent.Mount("/p", p)

	rr = performRequest(grandParent, "GET", "/p/s/final", nil)
	checkResponse(t, rr, http.StatusOK, "nested final")
}

// TestMethodNotAllowed tests 405 scenarios.
func TestMethodNotAllowed(t *testing.T) {
	r := NewRouter("test405")
	r.Get("/onlyget", nil, func(c *Context) { c.Text(http.StatusOK, "GET only") })

	rr := performRequest(r, "POST", "/onlyget", nil)
	checkResponse(t, rr, http.StatusMethodNotAllowed, "") 

	// Check Allow header (depends on ErrMethodNotAllowed implementation)
	// The current ErrMethodNotAllowed in context.go does not set Allow header.
	// If it were to be added, this check would be useful:
	// allowHeader := rr.Header().Get("Allow")
	// if !strings.Contains(allowHeader, "GET") {
	// 	t.Errorf("Expected 'Allow' header to contain 'GET', got: %s", allowHeader)
	// }
}

// TestNotFound tests 404 scenarios.
func TestNotFound(t *testing.T) {
	r := NewRouter("test404")
	r.Get("/exists", nil, func(c *Context) { c.Text(http.StatusOK, "exists") })

	rr := performRequest(r, "GET", "/doesnotexist", nil)
	checkResponse(t, rr, http.StatusNotFound, "")

	rr = performRequest(r, "GET", "/exists/nope", nil) // Partially matching
	checkResponse(t, rr, http.StatusNotFound, "")
	
	rr = performRequest(r, "GET", "/exist", nil) // Partial prefix, but not full node
	checkResponse(t, rr, http.StatusNotFound, "")
}

// TestRouteOverwriting tests behavior when registering conflicting routes.
func TestRouteOverwriting(t *testing.T) {
	// Current node.insert logic overwrites the route in the node's map.
	r := NewRouter("testOverwrite")

	r.Get("/conflict", nil, func(c *Context) {
		c.Text(http.StatusOK, "handler1")
	})
	// Registering again with the same method and path
	r.Get("/conflict", nil, func(c *Context) {
		c.Text(http.StatusOK, "handler2")
	})

	rr := performRequest(r, "GET", "/conflict", nil)
	checkResponse(t, rr, http.StatusOK, "handler2") // Expect handler2 to be called
}

// Note on Context and Params for testing:
// The tests for parameter and wildcard routes assume that `ServeHTTP` populates
// `c.PathParams (map[string]string)` on the `Context` object.
// Handlers in these tests use `c.PathParams["param_name"]` to get parameter values.
// This requires modifications to `Context` struct and `ServeHTTP` if not already done:
// 1. `context.go`: `Context` struct to include `PathParams map[string]string`.
// 2. `router.go`: `ServeHTTP` method to set `c.PathParams = paramsMap` before handler execution.
// These changes are external to this test file but are crucial for param value checking.
// Removing potential trailing characters or lines causing "expected declaration" error.
