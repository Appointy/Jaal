package graphql_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

func TestClone(t *testing.T) {
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
	type UnionType struct {
		schemabuilder.Union
		*A
		*B
	}
	type EnumType int32
	type InputType struct {
		ID schemabuilder.ID
	}

	schema := schemabuilder.NewSchema()
	schema.Enum(EnumType(0), map[string]interface{}{
		"INTERFACE": EnumType(0),
		"UNION":     EnumType(1),
	})

	query := schema.Query()
	query.FieldFunc("inner", func() Inner {
		return Inner{}
	})

	inner := schema.Object("inner", Inner{})
	inner.FieldFunc("interfaceType", func(args struct {
		Input InputType
	}) []*InterfaceType {
		retList := make([]*InterfaceType, 2)
		retList[0] = &InterfaceType{A: &A{Name: "a", Id: 1, UniqueA: int64(2)}}
		retList[1] = &InterfaceType{B: &B{Name: "b", Id: 2, UniqueB: int64(3)}}
		return retList
	})
	inner.FieldFunc("unionType", func() []*UnionType {
		retList := make([]*UnionType, 2)
		retList[0] = &UnionType{A: &A{Name: "a", Id: 1, UniqueA: int64(2)}}
		retList[1] = &UnionType{B: &B{Name: "b", Id: 2, UniqueB: int64(3)}}
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

	input := schema.InputObject("InputType", InputType{})
	input.FieldFunc("id", func(target *InputType, source schemabuilder.ID) {
		target.ID = source
	})

	copy := schema.Clone()
	builtSchema1 := schema.MustBuild()
	builtSchema2 := copy.MustBuild()
	introspection.AddIntrospectionToSchema(builtSchema1)
	introspection.AddIntrospectionToSchema(builtSchema2)

	q, err := graphql.Parse(`
							{
								__schema {
									queryType {
										fields {
											name
											type {
												kind
												ofType {
													kind
													name
												}
											}
										}
									}
								}
							}
	  `, nil)
	if err != nil {
		panic(err)
	}

	if err := graphql.ValidateQuery(context.Background(), builtSchema1.Query, q.SelectionSet); err != nil {
		t.Error(err)
	}
	e := graphql.Executor{}
	val1, err := e.Execute(context.Background(), builtSchema1.Query, nil, q)
	if err != nil {
		t.Error(err)
	}

	if err := graphql.ValidateQuery(context.Background(), builtSchema2.Query, q.SelectionSet); err != nil {
		t.Error(err)
	}
	val2, err := e.Execute(context.Background(), builtSchema2.Query, nil, q)
	if err != nil {
		t.Error(err)
	}

	result1, err := json.Marshal(val1)
	if err != nil {
		t.Error(err)
	}
	result2, err := json.Marshal(val2)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, string(result1), string(result2))
}
