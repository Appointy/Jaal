package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/appointy/idgen"
	"go.appointy.com/appointy/jaal"
	"go.appointy.com/appointy/jaal/introspection"
	"go.appointy.com/appointy/jaal/schemabuilder"
)

type node struct {
	customers []A
	providers []B
}

type A struct {
	Id   string
	Name string
}

type B struct {
	Id    string
	Email string
}

type NodeInterface struct {
	schemabuilder.Interface
	*A
	*B
}

func (s *node) registerNodeInterface(schema *schemabuilder.Schema) {
	schema.Query().FieldFunc("node", func(ctx context.Context, args struct {
		Id schemabuilder.ID
	}) *NodeInterface {
		if strings.Contains(args.Id.Value, "cus") {
			for _, cus := range s.customers {
				if cus.Id == args.Id.Value {
					return &NodeInterface{
						A: &cus,
					}
				}
			}
		}

		for _, pro := range s.providers {
			if pro.Id == args.Id.Value {
				return &NodeInterface{
					B: &pro,
				}
			}
		}

		return &NodeInterface{
			A: &A{Id: args.Id.Value},
			B: &B{Id: args.Id.Value},
		}
	})

	s.registerA(schema)
	s.registerB(schema)
}

func (s *node) registerA(schema *schemabuilder.Schema) {
	obj := schema.Object("A", A{})
	obj.FieldFunc("id", func(in *A) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("name", func(in *A) string {
		return in.Name
	})
}

func (s *node) registerB(schema *schemabuilder.Schema) {
	obj := schema.Object("B", B{})
	obj.FieldFunc("id", func(in *B) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("email", func(in *B) string {
		return in.Email
	})
}

func main() {
	s := node{
		customers: []A{
			{
				Id:   idgen.New("cus"),
				Name: "Anuj",
			},
		},
		providers: []B{
			{
				Id:    idgen.New("pro"),
				Email: "anuj.g@appointy.com",
			},
		},
	}

	fmt.Println(s.customers[0], s.providers[0])

	builder := schemabuilder.NewSchema()
	s.registerNodeInterface(builder)

	schema := builder.MustBuild()

	introspection.AddIntrospectionToSchema(schema)

	http.Handle("/graphql", jaal.HTTPHandler(schema))
	fmt.Println("Running")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		panic(err)
	}
}
