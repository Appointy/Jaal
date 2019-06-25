package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"github.com/appointy/idgen"
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
	"go.appointy.com/jaal/subscription"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
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
	}) *channel {
		var ch channel
		if err := gob.NewDecoder(bytes.NewReader(source.Payload)).Decode(&ch); err != nil {
			panic(err)
		}
		if args.In.Name == ch.Name {
			return &ch
		}
		return nil
	})

	obj.FieldFunc("postStream", func(source *schemabuilder.Subscription, args struct {
		In postStreamReq
	}) *post {
		var p post
		if err := gob.NewDecoder(bytes.NewReader(source.Payload)).Decode(&p); err != nil {
			panic(err)
		}
		if args.In.Tag == p.Tag {
			return &post{
				Id:    idgen.New("post"),
				Title: p.Title,
				Tag:   p.Tag,
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

	fmt.Println("objects")
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
	fmt.Println("built")
	introspection.AddIntrospectionToSchema(schema)
	subscription.RunSubscriptionServices()
	http.Handle("/graphql", jaal.HTTPSubHandler(schema))
	ctx := context.Background()
	s := pstest.NewServer()
	defer s.Close()
	conn, err := grpc.Dial(s.Addr, grpc.WithInsecure())
	if err != nil {
		fmt.Println("failed to create server")
	}
	defer conn.Close()
	cli, err := pubsub.NewClient(ctx, "some-project", option.WithGRPCConn(conn))
	if err != nil {
		fmt.Println("failed to create client:", err)
		return
	}
	top, err := cli.CreateTopic(ctx, "topName")
	if err != nil {
		fmt.Println("failed to create topic:", err)
		return
	}
	sub, err := cli.CreateSubscription(ctx, "subName", pubsub.SubscriptionConfig{
		Topic:       top,
		AckDeadline: 10 * time.Second,
	})
	if err != nil {
		fmt.Println("failed to create subscription:", err)
		return
	}

	fmt.Println("Running")
	// Non-blocking receiver
	go func() {
		if err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			fmt.Println("Sending to source")
			// Just have to pass the subscription event to the jaal.Source []byte channel.
			subscription.Source <- m.Data
			m.Ack()
		}); err != nil {
			fmt.Println("Error in receiving PubSub message")
		}
	}()

	// Publisher
	go func() {
		for {
			time.Sleep(2 * time.Second)
			var temp *channel
			var temp2 *post
			t := rand.Intn(100)
			if t < 33 {
				temp = &channel{
					Id:        idgen.New("src"),
					Name: "Serial Killer",
					Email: ":P",
				}
			} else if t >= 33 && t < 66 {
				temp = &channel{
					Id:        idgen.New("src"),
					Name: "Dirty Shoe",
					Email:  "Assassin.Groot@Nonsense.home",
				}
			} else {
				temp2 = &post{
					Title: "Master of Skins",
					Tag:   "Huer",
				}
			}
			var data bytes.Buffer
			p := subscription.NewPublisher()
			if temp != nil {
				if err := gob.NewEncoder(&data).Encode(*temp); err != nil {
					panic(err)
				}
				d, err := p.WrapSourceEvent("channelStream", data.Bytes())
				if err != nil {
					panic(fmt.Errorf("failed to gob: %v", err))
				}
				top.Publish(ctx, &pubsub.Message{
					Data: d,
				});
				fmt.Println("Published")
			} else {
				if err := gob.NewEncoder(&data).Encode(*temp2); err != nil {
					panic(err)
				}
				d, err := p.WrapSourceEvent("postStream", data.Bytes())
				if err != nil {
					panic(fmt.Errorf("failed to gob: %v", err))
				}
				top.Publish(ctx, &pubsub.Message{
					Data: d,
				})
				fmt.Println("Published")
			}
		}
	}()

	if err := http.ListenAndServe(":3000", nil); err != nil {
		fmt.Println(err)
	}
}
