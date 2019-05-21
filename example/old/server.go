package main

import (
	"go.appointy.com/appointy/jaal/graphql"
	"go.appointy.com/appointy/jaal/schemabuilder"
)

type channel struct {
	Id       string
	Name     string
	Email    string
	Resource resource
	Variants []variant
}

type variant struct {
	Id   string
	Name string
}

type resource struct {
	Id   string
	Name string
	Type ResourceType
}

type ResourceType int64

const (
	ZERO ResourceType = iota
	ONE
	TWO
	THREE
	FOUR
)

type createChannelReq struct {
	Id       string
	Name     string
	Email    string
	Resource resource
	Variants []variant
}

type getChannelReq struct {
	Id string
}

// server is our graphql server.
type server struct {
	channels []channel
}

// schema builds the graphql schema.
func (s *server) schema() *graphql.Schema {
	builder := schemabuilder.NewSchema()

	s.registerEnum(builder)
	s.registerMutation(builder)
	s.registerCreateChannelReq(builder)
	s.registerChannel(builder)
	s.registerQuery(builder)
	s.registerGetChannelReq(builder)

	return builder.MustBuild()
}
