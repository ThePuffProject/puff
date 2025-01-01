package puff

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/ThePuffProject/puff/openapi"
)

// handle input schema core and non-core features:

type CSRFInformation struct {
	Csrfheader string `kind:"header"`
	Csrfcookie string `kind:"cookie"`
}

type TestInputSchema1 struct {
	CSRFInformation
	ID        int     `kind:"path" description:"The ID field specifies the ID of the User to lookup."`
	Name      string  `kind:"query" description:"Name specifies the name of the User to lookup."`
	Email     string  `kind:"query" format:"email" required:"false" description:"The optional Email field allows for more specificity."`
	Latitude  float32 `kind:"formdata" required:"false" deprecated:"true"`
	Longitude float32 `kind:"formdata" required:"false" deprecated:"true"`
}

func TestHandleInputSchema(t *testing.T) {
	parameters := []openapi.Parameter{}
	fieldsType := reflect.TypeOf(TestInputSchema1{})
	err := handleInputSchema(&parameters, fieldsType)
	if err != nil {
		t.Errorf("unexpected error with handleInputSchema (TestInputSchema1): %v", err)
	}
	expected := `[{"name":"Csrfheader","in":"header","description":"","required":true,"deprecated":false,"allowReserved":false,"schema":{"type":"string","format":"string","examples":{"Example1":{"value":"string"}}}},{"name":"Csrfcookie","in":"cookie","description":"","required":true,"deprecated":false,"allowReserved":false,"schema":{"type":"string","format":"string","examples":{"Example1":{"value":"string"}}}},{"name":"ID","in":"path","description":"The ID field specifies the ID of the User to lookup.","required":true,"deprecated":false,"allowReserved":false,"schema":{"type":"integer","format":"int32","examples":{"Example1":{"value":123}}}},{"name":"Name","in":"query","description":"Name specifies the name of the User to lookup.","required":true,"deprecated":false,"allowReserved":false,"schema":{"type":"string","format":"string","examples":{"Example1":{"value":"string"}}}},{"name":"Email","in":"query","description":"The optional Email field allows for more specificity.","required":false,"deprecated":false,"allowReserved":false,"schema":{"type":"string","format":"email","examples":{"Example1":{"value":"string"}}}},{"name":"Latitude","in":"formdata","description":"","required":false,"deprecated":true,"allowReserved":false,"schema":{"type":"number","format":"float","examples":{"Example1":{"value":3.4e+38}}}},{"name":"Longitude","in":"formdata","description":"","required":false,"deprecated":true,"allowReserved":false,"schema":{"type":"number","format":"float","examples":{"Example1":{"value":3.4e+38}}}}]`

	j, err := json.Marshal(parameters)
	if err != nil {
		t.Errorf("unexpected testing error: %v", err)
		t.FailNow()
	}

	if string(j) != expected {
		t.Errorf("unexpected parameters data from handleInputSchema: %s. expected: %s", string(j), expected)
	}
}

func TestFieldsFromIncoming(t *testing.T) {
	w := httptest.NewRecorder()
	u, _ := url.Parse("http://127.0.0.1:8000/myroute/32?Name=john%20doe&Email=hello@world.com")
	req := &http.Request{
		Method: "POST",
		URL:    u,
		Body:   io.NopCloser(bytes.NewBufferString("Latitude=12.345&Longitude=54.321")),
		Header: http.Header{
			"Csrfheader":   []string{"abc123"},
			"Content-Type": []string{"application/x-www-form-urlencoded"},
		},
	}
	req.AddCookie(&http.Cookie{
		Name:  "Csrfcookie",
		Value: "abc123",
	})

	expected := TestInputSchema1{
		CSRFInformation: CSRFInformation{
			Csrfheader: "abc123",
			Csrfcookie: "abc123",
		},
		ID:        32,
		Name:      "john doe",
		Email:     "hello@world.com",
		Latitude:  12.345,
		Longitude: 54.321,
	}
	p := []openapi.Parameter{}

	app := DefaultApp("untitled")
	ctx := NewContext(w, req, app)
	router := NewRouter("untitled router", "")
	route := Get(router, "/myroute/{id}", func(_ *Context, _ *TestInputSchema1) {})

	err := handleInputSchema(&p, route.fieldsType)
	if err != nil {
		t.Errorf("unexpected handle input schema error: %v", err)
		t.FailNow()
	}

	route.params = p

	gotraw, err := fieldsFromIncoming(ctx, route, []string{"32"})
	if err != nil {
		t.Errorf("unexpected error in getting fields from incoming: %v", err)
		t.FailNow()
	}
	got, ok := gotraw.(TestInputSchema1)
	if !ok {
		t.Errorf("fields return value from fieldFromIncoming failed assertion")
		t.FailNow()
	}

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}
