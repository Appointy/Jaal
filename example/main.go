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
}

type createChannelReq struct {
	Id    string
	Name  string
	Email string
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
			Name:  args.In.Name,
			Id:    idgen.New("ch"),
			Email: args.In.Email,
		}
		s.channels = append(s.channels, ch)

		return ch
	})
}

// schema builds the graphql schema.
func (s *server) schema() *graphql.Schema {
	builder := schemabuilder.NewSchema()
	builder.Object("channel", channel{})

	inputObject := builder.InputObject("createChannelReq", createChannelReq{})
	inputObject.FieldFunc("id", func(in *createChannelReq, id schemabuilder.ID) {
		in.Id = id.Value
	})

	builder.InputObject("getChannelReq", getChannelReq{})

	s.registerQuery(builder)
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
			},
		},
	}

	fmt.Println(server)

	schema := server.schema()
	http.Handle("/graphql", jaal.HTTPHandler(schema))
	fmt.Println("Running")

	http.ListenAndServe(":3000", nil)
}
