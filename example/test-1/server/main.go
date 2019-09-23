package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/appointy/idgen"
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
	"golang.org/x/net/context"
)

type channel struct {
	Id       string
	Name     string
	Email    string
	Metadata map[string]string
}

type post struct {
	Id    string
	Title string
	Tag   string
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

type postStreamReq struct {
	Tag string
}

// server is our graphql server.
type server struct {
	channels []channel
}

// Struct for channelStream
type sourceChannel struct {
	Id        string
	FirstName string
	LastName  string
}

type sourcePost struct {
	Title string
	Tag   string
}

// registerQuery registers the root query type.
func (s *server) registerQuery(schema *schemabuilder.Schema) {
	obj := schema.Query()

	obj.FieldFunc("channel", func(ctx context.Context, args struct {
		In getChannelReq
	}) *channel {
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
	}) (*channel, error) {
		var ch channel
		if err := gob.NewDecoder(bytes.NewReader(source.Payload)).Decode(&ch); err != nil {
			panic(err)
		}
		if args.In.Name == ch.Name {
			return &ch, nil
		}
		return nil, graphql.ErrNoUpdate
	})

	obj.FieldFunc("postStream", func(source *schemabuilder.Subscription, args struct {
		In postStreamReq
	}) (*post, error) {
		var p post
		if err := gob.NewDecoder(bytes.NewReader(source.Payload)).Decode(&p); err != nil {
			panic(err)
		}
		if args.In.Tag == p.Tag {
			return &post{
				Id:    idgen.New("post"),
				Title: p.Title,
				Tag:   p.Tag,
			}, nil
		}
		return nil, graphql.ErrNoUpdate
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

	obj = builder.Object("post", post{})

	obj.FieldFunc("id", func(ctx context.Context, in *post) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})

	obj.FieldFunc("title", func(ctx context.Context, in *post) string {
		return in.Title
	})

	obj.FieldFunc("tag", func(ctx context.Context, in *post) string {
		return in.Tag
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

	inputObject = builder.InputObject("postStreamReq", postStreamReq{})
	inputObject.FieldFunc("tag", func(in *postStreamReq, tag *string) {
		in.Tag = *tag
	})

	s.registerQuery(builder)
	s.registerMutation(builder)
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
	introspection.AddIntrospectionToSchema(schema)
	ctx := context.Background()
	top, err := pubsub.OpenTopic(ctx, "mem://topicA")
	if err != nil {
		fmt.Println(err)
	}
	defer top.Shutdown(ctx)
	sub, err := pubsub.OpenSubscription(ctx, "mem://topicA")
	if err != nil {
		fmt.Println(err)
	}
	defer sub.Shutdown(ctx)
	handler, f := jaal.HTTPSubHandler(schema, sub)
	http.Handle("/graphql", handler)
	fmt.Println("Running...")

	// Publisher
	go func() {
		i := 0
		for {
			i++
			time.Sleep(2 * time.Second)
			var temp *channel
			var temp2 *post
			t := rand.Intn(100)
			if t < 33 {
				temp = &channel{
					Id:    idgen.New("src"),
					Name:  "Serial Killer",
					Email: ":P",
				}
			} else if t >= 33 && t < 66 {
				temp = &channel{
					Id:    idgen.New("src"),
					Name:  "Dirty Shoe",
					Email: "Assassin.Groot@Nonsense.home",
				}
			} else {
				temp2 = &post{
					Title: "Master of Skins",
					Tag:   "Huer",
				}
			}
			var data bytes.Buffer
			if temp != nil {
				if err := gob.NewEncoder(&data).Encode(*temp); err != nil {
					panic(err)
				}
				if err := top.Send(ctx, &pubsub.Message{
					Body:     data.Bytes(),
					Metadata: map[string]string{"type": "channelStream"},
				}); err != nil {
					fmt.Println(err)
					return
				}
			} else {
				if err := gob.NewEncoder(&data).Encode(*temp2); err != nil {
					panic(err)
				}
				if err := top.Send(ctx, &pubsub.Message{
					Body:     data.Bytes(),
					Metadata: map[string]string{"type": "postStream"},
				}); err != nil {
					fmt.Println(err)
					return
				}
			}
			if t < 10 {
				sub.Shutdown(ctx)
			}
		}
	}()
	f()
	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Println(err)
	}
}
