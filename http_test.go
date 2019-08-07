package jaal_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/schemabuilder"
)

func testHTTPRequest(req *http.Request) *httptest.ResponseRecorder {
	schema := schemabuilder.NewSchema()

	query := schema.Query()
	query.FieldFunc("mirror", func(args struct{ Value int64 }) int64 {
		return args.Value * -1
	})

	builtSchema := schema.MustBuild()

	rr := httptest.NewRecorder()
	handler := jaal.HTTPHandler(builtSchema)

	handler.ServeHTTP(rr, req)
	return rr
}

func TestHTTPMustPost(t *testing.T) {
	req, err := http.NewRequest("GET", "/graphql", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := testHTTPRequest(req)

	if rr.Code != 200 {
		t.Errorf("expected 200, but received %d", rr.Code)
	}

	if diff := pretty.Compare(rr.Body.String(), `{"data":null,"errors":[{"message":"request must be a POST","extensions":{"code":"Unknown"},"paths":[]}]}`); diff != "" {
		t.Errorf("expected response to match, but received %s", diff)
	}
}

func TestHTTPParseQuery(t *testing.T) {
	req, err := http.NewRequest("POST", "/graphql", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := testHTTPRequest(req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, but received %d", rr.Code)
	}

	if diff := pretty.Compare(rr.Body.String(), `{"data":null,"errors":[{"message":"request must include a query","extensions":{"code":"Unknown"},"paths":[]}]}`); diff != "" {
		t.Errorf("expected response to match, but received %s", diff)
	}
}

func TestHTTPMustHaveQuery(t *testing.T) {
	req, err := http.NewRequest("POST", "/graphql", strings.NewReader(`{"query":""}`))
	if err != nil {
		t.Fatal(err)
	}

	rr := testHTTPRequest(req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, but received %d", rr.Code)
	}

	if diff := pretty.Compare(rr.Body.String(), `{"data":null,"errors":[{"message":"must have a single query","extensions":{"code":"Unknown"},"paths":[]}]}`); diff != "" {
		t.Errorf("expected response to match, but received %s", diff)
	}
}

func TestHTTPSuccess(t *testing.T) {
	req, err := http.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "query TestQuery($value: int64) { mirror(value: $value) }", "variables": { "value": 1 }}`))
	if err != nil {
		t.Fatal(err)
	}

	rr := testHTTPRequest(req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, but received %d", rr.Code)
	}

	if diff := pretty.Compare(rr.Body.String(), "{\"data\":{\"mirror\":-1},\"errors\":null}"); diff != "" {
		t.Errorf("expected response to match, but received %s", diff)
	}
}

func TestHTTPContentType(t *testing.T) {
	req, err := http.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "query TestQuery($value: int64) { mirror(value: $value) }", "variables": { "value": 1 }}`))
	if err != nil {
		t.Fatal(err)
	}

	rr := testHTTPRequest(req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, but received %d", rr.Code)
	}

	if diff := pretty.Compare(rr.HeaderMap.Get("Content-Type"), "application/json"); diff != "" {
		t.Errorf("expected response to match, but received %s", diff)
	}
}
