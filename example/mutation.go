package main

import (
	"context"

	"github.com/appointy/idgen"
	"go.appointy.com/appointy/jaal/schemabuilder"
)

func (s *server) registerMutation(schema *schemabuilder.Schema) {
	obj := schema.Mutation()

	obj.FieldFunc("createChannel", func(ctx context.Context, args struct {
		In createChannelReq
	}) channel {

		ch := channel{
			Name:     args.In.Name,
			Id:       idgen.New("ch"),
			Email:    args.In.Email,
			Resource: args.In.Resource,
			Variants: args.In.Variants,
		}
		s.channels = append(s.channels, ch)

		return ch
	})

	inputObject := schema.InputObject("createChannelReq", createChannelReq{})
	inputObject.FieldFunc("id", func(in *createChannelReq, id schemabuilder.ID) {
		in.Id = id.Value
	})
	inputObject.FieldFunc("name", func(in *createChannelReq, name string) {
		in.Name = name
	})
	inputObject.FieldFunc("email", func(in *createChannelReq, email string) {
		in.Email = email
	})
	inputObject.FieldFunc("resource", func(in *createChannelReq, resource resource) {
		in.Resource = resource
	})
	inputObject.FieldFunc("variants", func(in *createChannelReq, variants []variant) {
		in.Variants = variants
	})

	inputObject = schema.InputObject("resource", resource{})
	inputObject.FieldFunc("id", func(in *resource, id schemabuilder.ID) {
		in.Id = id.Value
	})
	inputObject.FieldFunc("name", func(in *resource, name string) {
		in.Name = name
	})
	inputObject.FieldFunc("type", func(in *resource, rType ResourceType) {
		in.Type = rType
	})

	inputObject = schema.InputObject("variant", variant{})
	inputObject.FieldFunc("id", func(in *variant, id schemabuilder.ID) {
		in.Id = id.Value
	})
	inputObject.FieldFunc("name", func(in *variant, name string) {
		in.Name = name
	})

}

func (s *server) registerEnum(schema *schemabuilder.Schema) {
	schema.Enum(ResourceType(1), map[string]interface{}{
		"ONE":   ResourceType(1),
		"TWO":   ResourceType(2),
		"THREE": ResourceType(3),
	})
}
