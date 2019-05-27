package jaal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/srikrsna/sqlx"
	"go.appointy.com/jaal/graphql"
)

type executeQuery struct {
	schema   *graphql.Schema
	executor *graphql.Executor
	db       *sql.DB
}

type executeRequest struct {
	QueryId string `json:"query_id"`
}

//ExecuteQueryHandler handles the generation prepared queries
func ExecuteQueryHandler(schema *graphql.Schema, db *sql.DB) http.Handler {
	return &executeQuery{
		schema:   schema,
		executor: &graphql.Executor{},
		db:       db,
	}
}

//ServerHTTP gets the request prepared and executes it to generate the results
func (h *executeQuery) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeResponse := func(value interface{}, err error) {
		response := httpResponse{}
		if err != nil {
			response.Errors = []string{err.Error()}
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
		writeResponse(nil, errors.New("request must include the query id"))
		return
	}

	var params executeRequest
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeResponse(nil, err)
		return
	}

	query, err := h.getQuery(r.Context(), params.QueryId)
	if err != nil {
		writeResponse(nil, err)
	}

	schema := h.schema.Query
	if query.Kind == "mutation" {
		schema = h.schema.Mutation
	}
	output, err := h.executor.Execute(r.Context(), schema, nil, query)
	if err != nil {
		writeResponse(nil, err)
		return
	}
	writeResponse(output, nil)

}

func (h *executeQuery) getQuery(ctx context.Context, id string) (*graphql.Query, error) {
	gqlQuery := graphql.Query{}

	const query = `SELECT data FROM  graphql.queries WHERE id = $1;`
	if err := h.db.QueryRowContext(ctx, query, id).Scan(sqlx.JSON(&gqlQuery)); err != nil {
		return nil, err
	}

	return &gqlQuery, nil
}
