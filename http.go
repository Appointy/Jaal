package jaal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/jerrors"
)

type HandlerOption func(*handlerOptions)

type handlerOptions struct {
	Middlewares []MiddlewareFunc
}

// HTTPHandler implements the handler required for executing the graphql queries and mutations
func HTTPHandler(schema *graphql.Schema, opts ...HandlerOption) http.Handler {
	h := &httpHandler{
		handler: handler{
			schema:   schema,
			executor: &graphql.Executor{},
		},
	}

	o := handlerOptions{}
	for _, opt := range opts {
		opt(&o)
	}

	prev := h.execute
	for i := range o.Middlewares {
		prev = o.Middlewares[len(o.Middlewares)-1-i](prev)
	}
	h.exec = prev

	return h
}

type handler struct {
	schema   *graphql.Schema
	executor *graphql.Executor
}

type httpHandler struct {
	handler

	exec HandlerFunc
}

type httpPostBody struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type httpResponse struct {
	Data   interface{}      `json:"data"`
	Errors []*jerrors.Error `json:"errors"`
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeResponse := func(value interface{}, err error) {
		response := httpResponse{}
		if err != nil {
			response.Errors = []*jerrors.Error{jerrors.ConvertError(err)}
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

	root := h.schema.Query
	if query.Kind == "mutation" {
		root = h.schema.Mutation
	}

	if err := graphql.ValidateQuery(r.Context(), root, query.SelectionSet); err != nil {
		writeResponse(nil, err)
		return
	}

	output, err := h.exec(r.Context(), root, query)
	writeResponse(output, err)
}

func (h *httpHandler) execute(ctx context.Context, root graphql.Type, query *graphql.Query) (interface{}, error) {
	return h.executor.Execute(ctx, root, nil, query)
}
