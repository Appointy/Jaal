package main

import (
	"context"
	"strings"

	"go.appointy.com/appointy/jaal/schemabuilder"
)

type node struct {
	customers []Customer
	providers []Provider
}

type Customer struct {
	Id   string
	Name string
}

type Provider struct {
	Id    string
	Email string
}

type NodeInterface struct {
	schemabuilder.Interface
	*Customer
	*Provider
}

func (s *node) registerNodeInterface(schema *schemabuilder.Schema) {
	schema.Query().FieldFunc("node", func(ctx context.Context, args struct {
		Id schemabuilder.ID
	}) *NodeInterface {

		if strings.Contains(args.Id.Value, "cus") {
			for _, cus := range s.customers {
				if cus.Id == args.Id.Value {
					return &NodeInterface{
						Customer: &cus,
					}
				}
			}
		}

		for _, pro := range s.providers {
			if pro.Id == args.Id.Value {
				return &NodeInterface{
					Provider: &pro,
				}
			}
		}

		return &NodeInterface{
			Customer: &Customer{Id: args.Id.Value},
			Provider: &Provider{Id: args.Id.Value},
		}
	})

	s.registerA(schema)
	s.registerB(schema)
}

func (s *node) registerA(schema *schemabuilder.Schema) {
	obj := schema.Object("Customer", Customer{})
	obj.FieldFunc("id", func(in *Customer) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("name", func(in *Customer) string {
		return in.Name
	})
}

func (s *node) registerB(schema *schemabuilder.Schema) {
	obj := schema.Object("Provider", Provider{})
	obj.FieldFunc("id", func(in *Provider) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("email", func(in *Provider) string {
		return in.Email
	})
}

//func main() {
//	s := node{
//		customers: []Customer{
//			{
//				Id:   "cus_01DBF6E5CE9JY03HP3XGAVRAAC",
//				Name: "Anuj",
//			},
//		},
//		providers: []Provider{
//			{
//				Id:    "pro_01DBF6E5CE9JY03HP3XGMTCFR7",
//				Email: "anuj.g@appointy.com",
//			},
//		},
//	}
//
//	fmt.Println(s.customers[0], s.providers[0])
//
//	builder := schemabuilder.NewSchema()
//	s.registerNodeInterface(builder)
//
//	schema := builder.MustBuild()
//
//	introspection.AddIntrospectionToSchema(schema)
//
//	http.Handle("/graphql", jaal.HTTPHandler(schema))
//	fmt.Println("Running")
//	if err := http.ListenAndServe(":3000", nil); err != nil {
//		panic(err)
//	}
//}
