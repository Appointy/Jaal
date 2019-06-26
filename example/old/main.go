package main

// import (
// 	"context"
// 	"fmt"
// 	"net/http"

// 	"github.com/appointy/idgen"
// 	"go.appointy.com/jaal"
// 	"go.appointy.com/jaal/example/old/pb"
// 	"go.appointy.com/jaal/gtypes"
// 	"go.appointy.com/jaal/introspection"
// 	"go.appointy.com/jaal/schemabuilder"
// )

// type server struct {
// 	classes []*pb.Class
// }

// func main() {
// 	s := server{
// 		classes: []*pb.Class{},
// 	}
// 	s.classes = append(s.classes, &pb.Class{
// 		Id:     "cls_01DBH2SE828M81TSAM2B52958F",
// 		Charge: &pb.Class_PerInstance{PerInstance: "Testing one of"},
// 	})
// 	fmt.Println(s.classes[0])

// 	builder := schemabuilder.NewSchema()
// 	pb.RegisterTypes(builder)
// 	s.registerQuery(builder)
// 	s.registerMutation(builder)
// 	gtypes.RegisterWellKnownTypes()

// 	schema := builder.MustBuild()

// 	introspection.AddIntrospectionToSchema(schema)

// 	http.Handle("/graphql", jaal.HTTPHandler(schema))
// 	fmt.Println("Running")
// 	if err := http.ListenAndServe(":3030", nil); err != nil {
// 		panic(err)
// 	}

// }

// func (s *server) registerQuery(schema *schemabuilder.Schema) {
// 	schema.Query().FieldFunc("class", func(ctx context.Context, args struct {
// 		Id schemabuilder.ID
// 	}) *pb.Class {
// 		for _, class := range s.classes {
// 			if class.Id == args.Id.Value {
// 				return class
// 			}
// 		}
// 		return &pb.Class{}
// 	})
// }

// func (s *server) registerMutation(schema *schemabuilder.Schema) {
// 	schema.Mutation().FieldFunc("createClass", func(ctx context.Context, args struct {
// 		In *pb.CreateClassReq
// 	}) *pb.Class {
// 		args.In.Class.Id = idgen.New("cls")
// 		s.classes = append(s.classes, args.In.Class)

// 		return args.In.Class
// 	})
// }
