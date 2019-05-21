package main

import (
	"fmt"
	"net/http"

	"github.com/appointy/idgen"
	"go.appointy.com/appointy/jaal"
	"go.appointy.com/appointy/jaal/graphql"
	"go.appointy.com/appointy/jaal/schemabuilder"
	"golang.org/x/net/context"
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
}

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

	inputObject = schema.InputObject("variant", variant{})
	inputObject.FieldFunc("id", func(in *variant, id schemabuilder.ID) {
		in.Id = id.Value
	})
	inputObject.FieldFunc("name", func(in *variant, name string) {
		in.Name = name
	})

}

// schema builds the graphql schema.
func (s *server) schema() *graphql.Schema {
	builder := schemabuilder.NewSchema()
	builder.Object("channel", channel{})

	builder.InputObject("getChannelReq", getChannelReq{})

	s.registerMutation(builder)

	return builder.MustBuild()
}

func main() {
	// Instantiate a server, build a server, and serve the schema on port 3000.
	server := &server{
		channels: []channel{
			{
				Name:  "Table",
				Id:    idgen.New("ch"),
				Email: "table@appointy.com",
				Resource: resource{
					Id:   idgen.New("res"),
					Name: "channel",
				},
			},
		},
	}

	fmt.Println(server)

	schema := server.schema()
	http.Handle("/graphql", jaal.HTTPHandler(schema))
	fmt.Println("Running")

	http.ListenAndServe(":3000", nil)
}
