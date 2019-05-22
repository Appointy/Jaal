package graphql_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.appointy.com/appointy/jaal/graphql"
	"go.appointy.com/appointy/jaal/schemabuilder"
)

func TestInterface(t *testing.T) {
	type A struct {
		Name    string
		Id      int64
		UniqueA int64
	}

	type B struct {
		Name    string
		Id      int64
		UniqueB int64
	}

	type InterfaceType struct {
		schemabuilder.Interface
		*A
		*B
	}
	type Inner struct {
	}

	schema := schemabuilder.NewSchema()
	query := schema.Query()
	query.FieldFunc("inner", func() Inner {
		return Inner{}
	})

	inner := schema.Object("inner", Inner{})
	inner.FieldFunc("interfaceType", func() []*InterfaceType {
		retList := make([]*InterfaceType, 2)
		retList[0] = &InterfaceType{A: &A{Name: "a", Id: 1, UniqueA: int64(2)}}
		retList[1] = &InterfaceType{B: &B{Name: "b", Id: 2, UniqueB: int64(3)}}
		return retList
	})

	obj := schema.Object("A", A{})
	obj.FieldFunc("name", func(in A) string {
		return in.Name
	})
	obj.FieldFunc("id", func(in A) int64 {
		return in.Id
	})
	obj.FieldFunc("uniqueA", func(in A) int64 {
		return in.UniqueA
	})

	obj = schema.Object("B", B{})
	obj.FieldFunc("name", func(in B) string {
		return in.Name
	})
	obj.FieldFunc("id", func(in B) int64 {
		return in.Id
	})
	obj.FieldFunc("uniqueB", func(in B) int64 {
		return in.UniqueB
	})

	builtSchema := schema.MustBuild()
	q, err := graphql.Parse(`
	   {
		   inner {	
			   interfaceType {
				   name
				   id
				   ... on A { uniqueA }
				   ... on B { uniqueB }
			   }
		   }
	   }`, nil)
	if err != nil {
		panic(err)
	}

	if err := graphql.ValidateQuery(context.Background(), builtSchema.Query, q.SelectionSet); err != nil {
		t.Error(err)
	}
	e := graphql.Executor{}
	val, err := e.Execute(context.Background(), builtSchema.Query, nil, q)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, map[string]interface{}{
		"inner": map[string]interface{}{
			"interfaceType": []interface{}{
				map[string]interface{}{
					"id":      int64(1),
					"name":    "a",
					"uniqueA": int64(2),
				},
				map[string]interface{}{
					"id":      int64(2),
					"name":    "b",
					"uniqueB": int64(3),
				},
			},
		},
	}, val)

}
