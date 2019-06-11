package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/appointy/idgen"
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
	"golang.org/x/net/context"
)

type channel struct {
	Id       string
	Name     string
	Email    string
	Metadata map[string]string
}

type createChannelReq struct {
	Id    string
	Name  string
	Email string
}

type getChannelReq struct {
	Id string
}

type channelStreamReq struct {
	Name string
}

// server is our graphql server.
type server struct {
	channels []channel
}

type sourceChannel struct {
	Id        string
	FirstName string
	LastName  string
}

// registerQuery registers the root query type.
func (s *server) registerQuery(schema *schemabuilder.Schema) {
	obj := schema.Query()

	obj.FieldFunc("channel", func(ctx context.Context, args struct {
		In getChannelReq
	}) *channel {
		fmt.Println("dddddd")
		for _, ch := range s.channels {
			if ch.Id == args.In.Id {
				return &ch
			}
		}

		return nil
	})
}

func (s *server) registerMutation(schema *schemabuilder.Schema) {
	obj := schema.Mutation()

	obj.FieldFunc("createChannel", func(ctx context.Context, args struct {
		Ouch createChannelReq
	}) *channel {

		ch := channel{
			Name:  args.Ouch.Name,
			Id:    idgen.New("ch"),
			Email: args.Ouch.Email,
		}
		s.channels = append(s.channels, ch)
		fmt.Println(s)
		return &ch
	})
}

func (s *server) registerSubscription(schema *schemabuilder.Schema) {
	obj := schema.Subscription()

	obj.FieldFunc("channelStream", func(source *schemabuilder.Subscription, args struct {
		In channelStreamReq
	}) *channel {
		temp := source.Source.(sourceChannel)
		if args.In.Name == (temp.FirstName + " " + temp.LastName) {
			return &channel{
				Id:    temp.Id,
				Name:  temp.FirstName + " " + temp.LastName,
				Email: temp.FirstName + "@appointy.com",
			}
		}
		return nil
	})
}

// schema builds the graphql schema.
func (s *server) schema() *graphql.Schema {
	builder := schemabuilder.NewSchema()
	obj := builder.Object("channel", channel{})

	obj.FieldFunc("id", func(ctx context.Context, in *channel) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("name", func(ctx context.Context, in *channel) string {
		return in.Name
	})
	obj.FieldFunc("email", func(ctx context.Context, in *channel) string {
		return in.Email
	})

	inputObject := builder.InputObject("createChannelReq", createChannelReq{})
	inputObject.FieldFunc("id", func(in *createChannelReq, id *schemabuilder.ID) {
		in.Id = id.Value
	})
	inputObject.FieldFunc("name", func(in *createChannelReq, name *string) {
		in.Email = *name
	})
	inputObject.FieldFunc("email", func(in *createChannelReq, value *string) {
		in.Name = *value
	})

	inputObject = builder.InputObject("getChannelReq", getChannelReq{})
	inputObject.FieldFunc("id", func(in *getChannelReq, id *schemabuilder.ID) {
		in.Id = id.Value
	})

	inputObject = builder.InputObject("channelStreamReq", channelStreamReq{})
	inputObject.FieldFunc("name", func(in *channelStreamReq, name *string) {
		in.Name = *name
	})

	fmt.Println("objects")
	s.registerQuery(builder)
	s.registerMutation(builder)
	jaal.RegisterSubType("channelStream")
	// jaal.RegisterSubType
	s.registerSubscription(builder)

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
	fmt.Println("built")

	introspection.AddIntrospectionToSchema(schema)
	http.Handle("/graphql", jaal.HTTPHandler(schema))
	http.Handle("/graphql/sub", jaal.HTTPSubHandler(schema))
	fmt.Println("Running")
	go jaal.AddClientDaemon("channelStream")
	go jaal.SourceSubTypeTrigger("channelStream")
	go func() {
		for {
			time.Sleep(2 * time.Second)
			jaal.SubStreamManager.Lock.RLock()
			jaal.SubStreamManager.SubTypeStreams["channelStream"] <- sourceChannel{
				idgen.New("source"),
				"Table",
				"Saheb",
			}
			fmt.Println("Sent into stream - 1")
			jaal.SubStreamManager.Lock.RUnlock()
		}
	}()
	go func() {
		for {
			time.Sleep(2 * time.Second)
			jaal.SubStreamManager.Lock.RLock()
			jaal.SubStreamManager.SubTypeStreams["channelStream"] <- sourceChannel{
				idgen.New("source"),
				"Uptown",
				"Funk",
			}
			fmt.Println("Sent into stream - 2")
			jaal.SubStreamManager.Lock.RUnlock()
		}
	}()
	http.ListenAndServe(":3000", nil)
}
