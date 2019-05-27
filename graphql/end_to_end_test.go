package graphql_test

import (
	"context"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"

	"github.com/stretchr/testify/assert"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/internal"
	"go.appointy.com/jaal/schemabuilder"
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

func TestEnum(t *testing.T) {
	schema := schemabuilder.NewSchema()

	type enumType int32
	type enumType2 float64

	schema.Enum(enumType(1), map[string]interface{}{
		"firstField":  enumType(1),
		"secondField": enumType(2),
		"thirdField":  enumType(3),
	})
	schema.Enum(enumType2(1.2), map[string]float64{
		"this": float64(1.2),
		"is":   float64(3.2),
		"a":    float64(4.3),
		"map":  float64(5.3),
	})

	query := schema.Query()
	query.FieldFunc("inner", func(args struct {
		EnumField enumType
	}) enumType {
		return args.EnumField
	})
	query.FieldFunc("inner2", func(args struct {
		EnumField2 enumType2
	}) enumType2 {
		return args.EnumField2
	})

	query.FieldFunc("optional", func(args struct {
		EnumField *enumType
	}) enumType {
		if args.EnumField != nil {
			return *args.EnumField
		} else {
			return enumType(4)
		}
	})

	query.FieldFunc("pointerret", func(args struct {
		EnumField *enumType
	}) *enumType {
		return args.EnumField
	})

	builtSchema := schema.MustBuild()

	// Enum value as input argument and selection in a query
	q, err := graphql.Parse(`
		{
			inner(enumField: firstField)
		}
		`, nil)
	if err != nil {
		panic(err)
	}
	if err := graphql.ValidateQuery(context.Background(), builtSchema.Query, q.SelectionSet); err != nil {
		t.Error(err)
	}

	e := graphql.Executor{}
	val, err := e.Execute(context.Background(), builtSchema.Query, nil, q)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{
		"inner": "firstField",
	}, internal.AsJSON(val))

	// Same underlying type
	q, err = graphql.Parse(`
		{
			inner2(enumField2: this)
		}
		`, nil)
	if err != nil {
		panic(err)
	}
	if err := graphql.ValidateQuery(context.Background(), builtSchema.Query, q.SelectionSet); err != nil {
		t.Error(err)
	}

	val, err = e.Execute(context.Background(), builtSchema.Query, nil, q)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{
		"inner2": "this",
	}, internal.AsJSON(val))

	// Undefinded enum type
	q, err = graphql.Parse(`
		{
			inner(enumField: wrongField)
		}
		`, nil)
	if err != nil {
		panic(err)
	}
	if err := graphql.ValidateQuery(context.Background(), builtSchema.Query, q.SelectionSet); err == nil {
		t.Error("Parsed undefined enum type", err)
	}

	// Input is pointer to enum
	q, err = graphql.Parse(`
		{
			optional(enumField: firstField)
		}
		`, nil)
	if err != nil {
		panic(err)
	}
	if err := graphql.ValidateQuery(context.Background(), builtSchema.Query, q.SelectionSet); err != nil {
		t.Error(err)
	}

	val, err = e.Execute(context.Background(), builtSchema.Query, nil, q)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{
		"optional": "firstField",
	}, internal.AsJSON(val))

	// Output is pointer to enum
	q, err = graphql.Parse(`
		{
			pointerret(enumField: firstField)
		}
		`, nil)
	if err != nil {
		panic(err)
	}
	if err := graphql.ValidateQuery(context.Background(), builtSchema.Query, q.SelectionSet); err != nil {
		t.Error(err)
	}

	val, err = e.Execute(context.Background(), builtSchema.Query, nil, q)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{
		"pointerret": float64(1),
	}, internal.AsJSON(val))

}

func TestSkipDirectives(t *testing.T) {
	schema := schemabuilder.NewSchema()
	query := schema.Query()
	query.FieldFunc("value", func() string { return "s" })
	builtSchema := schema.MustBuild()

	execute := func(queryString string, vars map[string]interface{}) (interface{}, error) {
		q, err := graphql.Parse(queryString, vars)
		if err != nil {
			panic(err)
		}

		if err := graphql.ValidateQuery(context.Background(), builtSchema.Query, q.SelectionSet); err != nil {
			return nil, err
		}

		e := graphql.Executor{}
		return e.Execute(context.Background(), builtSchema.Query, nil, q)
	}

	// Variable skip
	result, err := execute(`
		query x {
			value @skip(if: $var)
		}`, map[string]interface{}{"var": true})
	if err != nil {
		t.Errorf("expected no err, received %s", err.Error())
	}
	if d := pretty.Compare(result, internal.ParseJSON("{}")); d != "" {
		t.Errorf("unexpected diff: %s", d)
	}

	result, err = execute(`
		query x {
			value @skip(if: $var)
		}`, map[string]interface{}{"var": false})
	if err != nil {
		t.Errorf("expected no err, received %s", err.Error())
	}
	if d := pretty.Compare(result, internal.ParseJSON(`{"value": "s"}`)); d != "" {
		t.Errorf("unexpected diff: %s", d)
	}

	// Variable include
	result, err = execute(`
		query x {
			value @include(if: $var)
		}`, map[string]interface{}{"var": false})
	if err != nil {
		t.Errorf("expected no err, received %s", err.Error())
	}
	if d := pretty.Compare(result, internal.ParseJSON("{}")); d != "" {
		t.Errorf("unexpected diff: %s", d)
	}

	result, err = execute(`
		query x {
			value @include(if: $var)
		}`, map[string]interface{}{"var": true})
	if err != nil {
		t.Errorf("expected no err, received %s", err.Error())
	}
	if d := pretty.Compare(result, internal.ParseJSON(`{"value": "s"}`)); d != "" {
		t.Errorf("unexpected diff: %s", d)
	}

	// Wrong type
	result, err = execute(`
		query x {
			value @skip(if: $var)
		}`, map[string]interface{}{"var": 5})
	if err == nil {
		t.Errorf("expected err, received nil")
	}
	if !strings.Contains(err.Error(), "expected type Boolean, found 5") {
		t.Errorf("expected err, received: %s", err.Error())
	}

	// Missing if
	result, err = execute(`
		query x {
			value @skip
		}`, nil)
	if err == nil {
		t.Errorf("expected err, received nil")
	}
	if !strings.Contains(err.Error(), "required argument not provided: if") {
		t.Errorf("expected err, received: %s", err.Error())
	}

	// Fragments
	result, err = execute(`
		query x {
			... on Query @skip(if: true) {
				value
			}
		}`, nil)
	if err != nil {
		t.Errorf("expected no err, received %s", err.Error())
	}
	if d := pretty.Compare(result, internal.ParseJSON("{}")); d != "" {
		t.Errorf("unexpected diff: %s", d)
	}

	result, err = execute(`
		query x {
			...X @skip(if: true)
		}
 		fragment X on Query {
			value
		}
`, nil)
	if err != nil {
		t.Errorf("expected no err, received %s", err.Error())
	}
	if d := pretty.Compare(result, internal.ParseJSON("{}")); d != "" {
		t.Errorf("unexpected diff: %s", d)
	}

	//When both skip and include are applied on same field
	result, err = execute(`
		query x {
			value @skip(if: $var) @include(if: $var)
		}`, map[string]interface{}{"var": true})
	if err != nil {
		t.Errorf("expected no err, received %s", err.Error())
	}
	if d := pretty.Compare(result, internal.ParseJSON("{}")); d != "" {
		t.Errorf("unexpected diff: %s", d)
	}

	result, err = execute(`
		query x {
			value @skip(if: $var) @include(if: $var)
		}`, map[string]interface{}{"var": false})
	if err != nil {
		t.Errorf("expected no err, received %s", err.Error())
	}
	if d := pretty.Compare(result, internal.ParseJSON("{}")); d != "" {
		t.Errorf("unexpected diff: %s", d)
	}

	result, err = execute(`
		query x {
			value @skip(if: $var1) @include(if: $var2)
		}`, map[string]interface{}{"var1": true, "var2": false})
	if err != nil {
		t.Errorf("expected no err, received %s", err.Error())
	}
	if d := pretty.Compare(result, internal.ParseJSON("{}")); d != "" {
		t.Errorf("unexpected diff: %s", d)
	}

	result, err = execute(`
		query x {
			value @skip(if: $var1) @include(if: $var2)
		}`, map[string]interface{}{"var1": false, "var2": true})
	if err != nil {
		t.Errorf("expected no err, received %s", err.Error())
	}
	if d := pretty.Compare(result, internal.ParseJSON(`{"value": "s"}`)); d != "" {
		t.Errorf("unexpected diff: %s", d)
	}
}
