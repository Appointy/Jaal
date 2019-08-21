package jaal

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.appointy.com/jaal/schemabuilder"
)

func TestHttpCall(t *testing.T) {
	schema := schemabuilder.NewSchema()

	query := schema.Query()
	query.FieldFunc("mirror", func(args struct{ Value float64 }) float64 {
		return args.Value * -1
	})

	builtSchema := schema.MustBuild()
	handler := HTTPHandler(builtSchema)

	server := httptest.NewServer(handler)

	t.Run("Without variables", func(t *testing.T) {
		query := `query Test{
					mirror(value: 1.1)
				}`
		expected := map[string]interface{}{"mirror": -1.1}

		response, err := HttpCall(server.URL, query, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, response, expected)
	})

	t.Run("With variables", func(t *testing.T) {
		query := `query Test($value: Int){
					mirror(value: $value)
				}`
		variables := map[string]interface{}{"value": 1.1}
		expected := map[string]interface{}{"mirror": -1.1}

		response, err := HttpCall(server.URL, query, variables, nil)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, response, expected)
	})

}

func TestHttpCallErrors(t *testing.T) {
	type object struct{}

	schema := schemabuilder.NewSchema()
	payload := schema.Object("object", object{})
	payload.FieldFunc("mirror", func(args struct{ Value int64 }) int64 {
		return args.Value * -1
	})

	query := schema.Query()
	query.FieldFunc("mirror", func(args struct{ Value int64 }) int64 {
		return args.Value * -1
	})
	query.FieldFunc("nil", func(args struct{}) (*object, error) {
		return nil, nil
	})

	builtSchema := schema.MustBuild()
	handler := HTTPHandler(builtSchema)

	server := httptest.NewServer(handler)

	t.Run("With errors", func(t *testing.T) {
		query := `query Test{
					search(value: 1)
				}`

		if _, err := HttpCall(server.URL, query, nil, nil); len(err) == 0 {
			t.Fatalf("Expected error x")
		}
	})

	t.Run("Without data and error", func(t *testing.T) {
		query := `query Test{
					nil{
						mirror
					}
				}`
		variables := map[string]interface{}{}
		expected := map[string]interface{}{"nil": nil}

		response, err := HttpCall(server.URL, query, variables, nil)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, response, expected)
	})

}
