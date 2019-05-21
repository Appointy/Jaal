package main

import (
	"context"
	"fmt"
	"net/http"

	"go.appointy.com/appointy/jaal/gtypes"

	"go.appointy.com/appointy/jaal"

	"github.com/appointy/idgen"
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
		Id:idgen.New("cls"),
		Charge:&pb.Class_Lumpsum{Lumpsum:1000},
	})
	fmt.Println(s.classes[0])


	builder := schemabuilder.NewSchema()
	pb.RegisterTypes(builder)
	s.registerQuery(builder)
	s.registerMutation(builder)
	gtypes.RegisterStringStringMap()

	schema := builder.MustBuild()
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
		s.classes = append(s.classes, args.In.Class)

		return args.In.Class
	})
}
