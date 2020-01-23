package jaal

import "context"

import "go.saastack.io/jaal/graphql"

type HandlerFunc func(context.Context, graphql.Type, *graphql.Query) (interface{}, error)

type MiddlewareFunc func(HandlerFunc) HandlerFunc

func WithMiddlewares(mm ...MiddlewareFunc) HandlerOption {
	return func(h *handlerOptions) {
		h.Middlewares = append(h.Middlewares, mm...)
	}
}
