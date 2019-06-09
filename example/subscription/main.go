package main

import (
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/introspection"
)

func registerSubscription(schema *schemabuilder.Schema) {
	obj := builder.Subscription()

	obj.FieldFunc("post", func(args struct{
		id string
	}) post{
		for _, v := range posts {
			
		}
		return 
	})
}

func registerObjects(scehma *schemabuilder.Schema) {

}

func registerInputObjects(schema *schemabuilder.Schema) {

}

func main() {
	builder := schemabuilder.NewSchema()
	registerSubscription(builder)
}