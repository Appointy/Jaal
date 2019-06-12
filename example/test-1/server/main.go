package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"github.com/appointy/idgen"
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
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

	if err := jaal.RegisterSubType("channelStream", func(source *pubsub.Message) (sourceChannel, error) {
		var temp sourceChannel
		if err := json.Unmarshal(source.Data, &temp); err != nil {
			return sourceChannel{}, err
		}
		return temp, nil
	}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	obj.FieldFunc("postStream", func(source *schemabuilder.Subscription, args struct {
		In postStreamReq
	}) *post {
		temp := source.Source.(sourcePost)
		if args.In.Tag == temp.Tag {
			return &post{
				Id:    idgen.New("post"),
				Title: temp.Title,
				Tag:   temp.Tag,
			}
		}
		return nil
	})
	if err := jaal.RegisterSubType("postStream", func(source *pubsub.Message) (sourcePost, error) {
		var temp sourcePost
		if err := json.Unmarshal(source.Data, &temp); err != nil {
			return sourcePost{}, err
		}
		return temp, nil
	}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
	//jaal.runSubscriptionServices()
	introspection.AddIntrospectionToSchema(schema)
	http.Handle("/graphql", jaal.HTTPHandler(schema))
	http.Handle("/graphql/sub", jaal.HTTPSubHandler(schema))

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

	jaal.RunSubscriptionServices(ctx, sub)

	fmt.Println("Running")
	go func() {
		for {
			time.Sleep(2 * time.Second)
			temp := sourceChannel{}
			var temp2 sourcePost
			t := rand.Intn(100)
			if t < 33 {
				temp = sourceChannel{
					Id:        idgen.New("src"),
					FirstName: "Serial",
					LastName:  "Killer",
				}
			} else if t >= 33 && t < 66 {
				temp = sourceChannel{
					Id:        idgen.New("src"),
					FirstName: "Dirty",
					LastName:  "Shoe",
				}
			} else {
				temp2 = sourcePost{
					Title: "Master of Skins",
					Tag:   "Huer",
				}
			}
			var data []byte
			if (temp != sourceChannel{}) {
				data, _ = json.Marshal(temp)
			} else {
				data, _ = json.Marshal(temp2)
			}
			top.Publish(ctx, &pubsub.Message{
				Data: data,
			})
		}
	}()

	http.ListenAndServe(":3000", nil)
}
