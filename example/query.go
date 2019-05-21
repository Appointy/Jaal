package main

import (
	"context"

	"go.appointy.com/appointy/jaal/schemabuilder"
)

// registerQuery registers the root query type.
func (s *server) registerQuery(schema *schemabuilder.Schema) {
	obj := schema.Query()

	obj.FieldFunc("channel", func(ctx context.Context, args struct {
		In getChannelReq
	}) channel {
		for _, ch := range s.channels {
			if ch.Id == args.In.Id {
				return ch
			}
		}

		return channel{}
	})
}

func (s *server) registerChannel(schema *schemabuilder.Schema) {
	obj := schema.Object("channel", channel{})
	obj.FieldFunc("id", func(ctx context.Context, in *channel) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("name", func(ctx context.Context, in *channel) string {
		return in.Name
	})
	obj.FieldFunc("email", func(ctx context.Context, in *channel) string {
		return in.Email
	})
	obj.FieldFunc("resource", func(ctx context.Context, in *channel) resource {
		return in.Resource
	})
	obj.FieldFunc("variants", func(ctx context.Context, in *channel) []variant {
		return in.Variants
	})

	obj = schema.Object("resource", resource{})
	obj.FieldFunc("id", func(ctx context.Context, in *resource) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("name", func(ctx context.Context, in *resource) string {
		return in.Name
	})
	obj.FieldFunc("type", func(ctx context.Context, in *resource) ResourceType {
		return in.Type
	})

	obj = schema.Object("variant", variant{})
	obj.FieldFunc("id", func(ctx context.Context, in *variant) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("name", func(ctx context.Context, in *variant) string {
		return in.Name
	})
}

func (s *server) registerGetChannelReq(schema *schemabuilder.Schema) {
	inputObject := schema.InputObject("getChannelReq",getChannelReq{})
	inputObject.FieldFunc("id", func(in *getChannelReq, id schemabuilder.ID) {
		in.Id = id.Value
	})
}