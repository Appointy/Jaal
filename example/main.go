package main

import (
	"context"
	"fmt"
	"net/http"

	"go.appointy.com/appointy/jaal"
	"go.appointy.com/appointy/jaal/gtypes"
	"go.appointy.com/appointy/jaal/introspection"

	"github.com/appointy/idgen"
	"github.com/golang/protobuf/ptypes/empty"
	"go.appointy.com/appointy/jaal/example/pb"
	"go.appointy.com/appointy/jaal/schemabuilder"
)

type server struct {
	classes []*pb.Class
}

func main() {
	s := server{
		classes: []*pb.Class{},
	}
	s.classes = append(s.classes, &pb.Class{
		Id:     "cls_01DBH2SE828M81TSAM2B52958F",
		Charge: &pb.Class_PerInstance{PerInstance: "Testing one of"},
	})
	fmt.Println(s.classes[0])

	builder := schemabuilder.NewSchema()
	pb.RegisterTypes(builder)
	s.registerQuery(builder)
	s.registerMutation(builder)
	gtypes.RegisterWellKnownTypes()

	schema := builder.MustBuild()

	introspection.AddIntrospectionToSchema(schema)

	http.Handle("/graphql", jaal.HTTPHandler(schema))
	fmt.Println("Running")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		panic(err)
	}

}

func (s *server) registerQuery(schema *schemabuilder.Schema) {
	schema.Query().FieldFunc("class", func(ctx context.Context, args struct {
		In *pb.GetClassReq
	}) *pb.Class {
		for _, class := range s.classes {
			if class.Id == args.In.Id {
				return class
			}
		}
		return &pb.Class{}
	})
}

func (s *server) registerMutation(schema *schemabuilder.Schema) {
	schema.Mutation().FieldFunc("createClass", func(ctx context.Context, args struct {
		In *pb.CreateClassReq
	}) *pb.Class {
		args.In.Class.Id = idgen.New("cls")
		args.In.Class.Empty = &empty.Empty{}
		s.classes = append(s.classes, args.In.Class)

		return args.In.Class
	})
}
