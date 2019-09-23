package jaal

import (
	"encoding/json"
	"errors"
	"net/http"

	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/internal"
)

// HTTPHandler implements the handler required for executing the graphql queries and mutations
func HTTPHandler(schema *graphql.Schema) http.Handler {
	return &httpHandler{
		handler{
			schema:   schema,
			executor: &graphql.Executor{},
		},
	}
}

type handler struct {
	schema   *graphql.Schema
	executor *graphql.Executor
}

type httpHandler struct {
	handler
}

type httpPostBody struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type httpResponse struct {
	Data   interface{}       `json:"data"`
	Errors []*internal.Error `json:"errors"`
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeResponse := func(value interface{}, err error) {
		response := httpResponse{}
		if err != nil {
			response.Errors = []*internal.Error{internal.ConvertError(err)}
		} else {
			response.Data = value
		}

		responseJSON, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.Write(responseJSON)
	}

	if r.Method != "POST" {
		writeResponse(nil, errors.New("request must be a POST"))
		return
	}

	if r.Body == nil {
		writeResponse(nil, errors.New("request must include a query"))
		return
	}

	var params httpPostBody
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeResponse(nil, err)
		return
	}

	query, err := graphql.Parse(params.Query, params.Variables)
	if err != nil {
		writeResponse(nil, err)
		return
	}

	schema := h.schema.Query
	if query.Kind == "mutation" {
		schema = h.schema.Mutation
	}

	if err := graphql.ValidateQuery(r.Context(), schema, query.SelectionSet); err != nil {
		writeResponse(nil, err)
		return
	}
	output, err := h.executor.Execute(r.Context(), schema, nil, query)
	if err != nil {
		writeResponse(nil, err)
		return
	}
	writeResponse(output, nil)
}
