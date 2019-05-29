package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/appointy/idgen"
	"github.com/golang/protobuf/ptypes"
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/example/pb"
	"go.appointy.com/jaal/gtypes"
	"go.appointy.com/jaal/introspection"
)

type server struct {
	classes []*pb.Class
}

func main() {
	meta := make(map[string]*pb.Value)
	meta["key"] = &pb.Value{
		Name: "name",
		Game: "game",
	}
	s := &server{classes: []*pb.Class{{
		Id:     idgen.New("cls"),
		Parent: idgen.New("par"),
		Area:   99.99,
		Charge: &pb.Class_Lumpsum{
			Lumpsum: 5454,
		},
		Instructors: []*pb.ServiceProvider{
			{
				Id:        idgen.New("spr"),
				FirstName: "Anuj Gupta",
			},
		},
		IsDeleted: false,
		Metadata:  meta,
		StartDate: ptypes.TimestampNow(),
		Strength:  100,
		Type:      pb.ClassType_REGULAR,
	}}}
	client := pb.NewLocalClassesClient(s)
	pb.RegisterClassesOperations(gtypes.Schema, client)

	schema := gtypes.Schema.MustBuild()
	introspection.AddIntrospectionToSchema(schema)

	http.Handle("/graphql", jaal.HTTPHandler(schema))
	fmt.Println("Running")
	fmt.Println(s.classes[0])
	if err := http.ListenAndServe(":3030", nil); err != nil {
		panic(err)
	}
}

func (s *server) GetClass(ctx context.Context, in *pb.GetClassReq) (*pb.Class, error) {
	for _, class := range s.classes {
		if class.Id == in.Id {
			return class, nil
		}
	}
	return &pb.Class{}, nil
}

func (s *server) CreateClass(ctx context.Context, in *pb.CreateClassReq) (*pb.Class, error) {
	in.Class.Id = idgen.New("cls")
	s.classes = append(s.classes, in.Class)
	return in.Class, nil
}

//   {
// 	class(id:"cls_01DC2D6W8BCJY14RDJ4Q0SRZE8"){
// 	  area
// 	  charge{
// 		...on Class_Lumpsum{
// 		  lumpsum
// 		}
// 		...on Class_PerInstance{
// 		  perInstance
// 		}
// 	  }
// 	  duration
// 	  id
// 	  instructors{
// 		firstName
// 		id
// 	  }
// 	  isDeleted
// 	  metadata
// 	  parent
// 	  startDate
// 	  strength
// 	  type
// 	}
//   }

// mutation Create {
// 	createClass(input: {clientMutationId: "client_123545", parent: "par_01DC2D6W8BCJY14RDJ4Q1SNNNN", class: {area: 1909, classLumpsum: {lumpsum: 999}, duration: "938938s", instructors: [{id: "sp123456", firstName: "Britney"}], isDeleted: false, strength: 25, type: SERIES, metadata: "eyJtYXNzYWdlIjp7Im5hbWUiOiJCcml0bmV5IiwiZ2FtZSI6IkNyaWNrZXQifX0=", parent: "par_99999", startDate: "2012-11-01T22:08:41+00:00"}}) {
// 	  payload {
// 		id
// 	  }
// 	}
//   }
