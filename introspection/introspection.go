package introspection

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/schemabuilder"
)

type introspection struct {
	types        map[string]graphql.Type
	query        graphql.Type
	mutation     graphql.Type
	subscription graphql.Type
}

type DirectiveLocation string

const (
	QUERY               DirectiveLocation = "QUERY"
	MUTATION                              = "MUTATION"
	FIELD                                 = "FIELD"
	FRAGMENT_DEFINITION                   = "FRAGMENT_DEFINITION"
	FRAGMENT_SPREAD                       = "FRAGMENT_SPREAD"
	INLINE_FRAGMENT                       = "INLINE_FRAGMENT"
	SUBSCRIPTION                          = "SUBSCRIPTION"
)

type TypeKind string

const (
	SCALAR       TypeKind = "SCALAR"
	OBJECT                = "OBJECT"
	INTERFACE             = "INTERFACE"
	UNION                 = "UNION"
	ENUM                  = "ENUM"
	INPUT_OBJECT          = "INPUT_OBJECT"
	LIST                  = "LIST"
	NON_NULL              = "NON_NULL"
)

type InputValue struct {
	Name         string
	Description  string
	Type         Type
	DefaultValue *string
}

func (s *introspection) registerInputValue(schema *schemabuilder.Schema) {
	obj := schema.Object("__InputValue", InputValue{})
	obj.FieldFunc("name", func(in InputValue) string {
		return in.Name
	})
	obj.FieldFunc("description", func(in InputValue) string {
		return in.Description
	})
	obj.FieldFunc("type", func(in InputValue) Type {
		return in.Type
	})
	obj.FieldFunc("defaultValue", func(in InputValue) *string {
		return in.DefaultValue
	})
}

type EnumValue struct {
	Name              string
	Description       string
	IsDeprecated      bool
	DeprecationReason string
}

func (s *introspection) registerEnumValue(schema *schemabuilder.Schema) {
	obj := schema.Object("__EnumValue", EnumValue{})
	obj.FieldFunc("name", func(in EnumValue) string {
		return in.Name
	})
	obj.FieldFunc("description", func(in EnumValue) string {
		return in.Description
	})
	obj.FieldFunc("isDeprecated", func(in EnumValue) bool {
		return in.IsDeprecated
	})
	obj.FieldFunc("deprecationReason", func(in EnumValue) string {
		return in.DeprecationReason
	})
}

type Directive struct {
	Name        string
	Description string
	Locations   []DirectiveLocation
	Args        []InputValue
}

func (s *introspection) registerDirective(schema *schemabuilder.Schema) {
	obj := schema.Object("__Directive", Directive{})
	obj.FieldFunc("name", func(in Directive) string {
		return in.Name
	})
	obj.FieldFunc("description", func(in Directive) string {
		return in.Description
	})
	obj.FieldFunc("locations", func(in Directive) []DirectiveLocation {
		return in.Locations
	})
	obj.FieldFunc("args", func(in Directive) []InputValue {
		return in.Args
	})

	// if err := schemabuilder.RegisterScalar(reflect.TypeOf(DirectiveLocation("")), "directiveLocation", func(value interface{}, dest reflect.Value) error {
	// 	asString, ok := value.(string)
	// 	if !ok {
	// 		return errors.New("not a string")
	// 	}
	// 	dest.Set(reflect.ValueOf(asString).Convert(dest.Type()))
	// 	return nil
	// }); err != nil {
	// 	panic(err)
	// }

	schema.Enum(DirectiveLocation("QUERY"), map[string]interface{}{
		"QUERY":               DirectiveLocation("QUERY"),
		"MUTATION":            DirectiveLocation("MUTATION"),
		"FIELD":               DirectiveLocation("FIELD"),
		"FRAGMENT_DEFINITION": DirectiveLocation("FRAGMENT_DEFINITION"),
		"FRAGMENT_SPREAD":     DirectiveLocation("FRAGMENT_SPREAD"),
		"INLINE_FRAGMENT":     DirectiveLocation("INLINE_FRAGMENT"),
		"SUBSCRIPTION":        DirectiveLocation("SUBSCRIPTION"),
	})
}

type Schema struct {
	Types            []Type
	QueryType        *Type
	MutationType     *Type
	SubscriptionType *Type
	Directives       []Directive
}

func (s *introspection) registerSchema(schema *schemabuilder.Schema) {
	obj := schema.Object("__Schema", Schema{})
	obj.FieldFunc("types", func(in Schema) []Type {
		return in.Types
	})
	obj.FieldFunc("queryType", func(in Schema) *Type {
		return in.QueryType
	})
	obj.FieldFunc("mutationType", func(in Schema) *Type {
		return in.MutationType
	})
	obj.FieldFunc("subscriptionType", func(in Schema) *Type {
		return in.SubscriptionType
	})
	obj.FieldFunc("directives", func(in Schema) []Directive {
		return in.Directives
	})

}

type Type struct {
	Inner graphql.Type `json:"-"`
}

func (s *introspection) registerType(schema *schemabuilder.Schema) {
	object := schema.Object("__Type", Type{})
	object.FieldFunc("kind", func(t Type) TypeKind {
		switch t.Inner.(type) {
		case *graphql.Object:
			return OBJECT
		case *graphql.Union:
			return UNION
		case *graphql.Interface:
			return INTERFACE
		case *graphql.Scalar:
			return SCALAR
		case *graphql.Enum:
			return ENUM
		case *graphql.List:
			return LIST
		case *graphql.InputObject:
			return INPUT_OBJECT
		case *graphql.NonNull:
			return NON_NULL
		default:
			return ""
		}
	})

	object.FieldFunc("name", func(t Type) string {
		switch t := t.Inner.(type) {
		case *graphql.Object:
			return t.Name
		case *graphql.Union:
			return t.Name
		case *graphql.Interface:
			return t.Name
		case *graphql.Scalar:
			return t.Type
		case *graphql.Enum:
			return t.Type
		case *graphql.InputObject:
			return t.Name
		default:
			return ""
		}
	})

	object.FieldFunc("description", func(t Type) string {
		switch t := t.Inner.(type) {
		case *graphql.Object:
			return t.Description
		case *graphql.Union:
			return t.Description
		case *graphql.Interface:
			return t.Description
		default:
			return ""
		}
	})

	object.FieldFunc("interfaces", func(t Type) []Type {
		switch t := t.Inner.(type) {
		case *graphql.Object:
			types := make([]Type, 0, len(t.Interfaces))
			for _, typ := range t.Interfaces {
				types = append(types, Type{Inner: typ})
			}

			sort.Slice(types, func(i, j int) bool { return types[i].Inner.String() < types[j].Inner.String() })
			return types
		default:
			return nil
		}
	})
	object.FieldFunc("possibleTypes", func(t Type) []Type {
		switch t := t.Inner.(type) {
		case *graphql.Union:
			types := make([]Type, 0, len(t.Types))
			for _, typ := range t.Types {
				types = append(types, Type{Inner: typ})
			}

			sort.Slice(types, func(i, j int) bool { return types[i].Inner.String() < types[j].Inner.String() })
			return types
		case *graphql.Interface:
			types := make([]Type, 0, len(t.Types))
			for _, typ := range t.Types {
				types = append(types, Type{Inner: typ})
			}

			sort.Slice(types, func(i, j int) bool { return types[i].Inner.String() < types[j].Inner.String() })
			return types
		default:
			return nil
		}
	})

	object.FieldFunc("inputFields", func(t Type) []InputValue {
		var fields []InputValue

		switch t := t.Inner.(type) {
		case *graphql.InputObject:
			for name, f := range t.InputFields {
				fields = append(fields, InputValue{
					Name: name,
					Type: Type{Inner: f},
				})
			}
		}

		sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
		return fields
	})

	object.FieldFunc("fields", func(t Type, args struct {
		IncludeDeprecated *bool
	}) []field {
		var fields []field

		switch t := t.Inner.(type) {
		case *graphql.Object:
			for name, f := range t.Fields {
				var args []InputValue
				for name, a := range f.Args {
					args = append(args, InputValue{
						Name: name,
						Type: Type{Inner: a},
					})
				}
				sort.Slice(args, func(i, j int) bool { return args[i].Name < args[j].Name })

				fields = append(fields, field{
					Name: name,
					Type: Type{Inner: f.Type},
					Args: args,
				})
			}
		case *graphql.Interface:
			for name, f := range t.Fields {
				var args []InputValue
				for name, a := range f.Args {
					args = append(args, InputValue{
						Name: name,
						Type: Type{Inner: a},
					})
				}
				sort.Slice(args, func(i, j int) bool { return args[i].Name < args[j].Name })

				fields = append(fields, field{
					Name: name,
					Type: Type{Inner: f.Type},
					Args: args,
				})
			}
		}
		sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })

		return fields
	})

	object.FieldFunc("ofType", func(t Type) *Type {
		switch t := t.Inner.(type) {
		case *graphql.List:
			return &Type{Inner: t.Type}
		case *graphql.NonNull:
			return &Type{Inner: t.Type}
		default:
			return nil
		}
	})

	object.FieldFunc("enumValues", func(t Type, args struct {
		IncludeDeprecated *bool
	}) []EnumValue {

		switch t := t.Inner.(type) {
		case *graphql.Enum:
			var enumVals []EnumValue
			for k, v := range t.ReverseMap {
				val := fmt.Sprintf("%v", k)
				enumVals = append(enumVals,
					EnumValue{Name: v, Description: val, IsDeprecated: false, DeprecationReason: ""})
			}
			sort.Slice(enumVals, func(i, j int) bool { return enumVals[i].Name < enumVals[j].Name })
			return enumVals
		}
		return nil
	})
}

type field struct {
	Name              string
	Description       string
	Args              []InputValue
	Type              Type
	IsDeprecated      bool
	DeprecationReason string
}

func (s *introspection) registerField(schema *schemabuilder.Schema) {
	obj := schema.Object("__Field", field{})
	obj.FieldFunc("name", func(in field) string {
		return in.Name
	})
	obj.FieldFunc("description", func(in field) string {
		return in.Description
	})
	obj.FieldFunc("type", func(in field) Type {
		return in.Type
	})
	obj.FieldFunc("args", func(in field) []InputValue {
		return in.Args
	})
	obj.FieldFunc("isDeprecated", func(in field) bool {
		return in.IsDeprecated
	})
	obj.FieldFunc("deprecationReason", func(in field) string {
		return in.DeprecationReason
	})
}

func collectTypes(typ graphql.Type, types map[string]graphql.Type) {
	switch typ := typ.(type) {
	case *graphql.Object:
		if _, ok := types[typ.Name]; ok {
			return
		}
		types[typ.Name] = typ

		for _, field := range typ.Fields {
			collectTypes(field.Type, types)

			for _, arg := range field.Args {
				collectTypes(arg, types)
			}
		}

	case *graphql.Union:
		if _, ok := types[typ.Name]; ok {
			return
		}
		types[typ.Name] = typ
		for _, graphqlTyp := range typ.Types {
			collectTypes(graphqlTyp, types)
		}

	case *graphql.Interface:
		if _, ok := types[typ.Name]; ok {
			return
		}
		types[typ.Name] = typ

		for _, field := range typ.Fields {
			collectTypes(field.Type, types)

			for _, arg := range field.Args {
				collectTypes(arg, types)
			}
		}
		for _, object := range typ.Types {
			collectTypes(object, types)
		}

	case *graphql.List:
		collectTypes(typ.Type, types)

	case *graphql.Scalar:
		if _, ok := types[typ.Type]; ok {
			return
		}
		types[typ.Type] = typ

	case *graphql.Enum:
		if _, ok := types[typ.Type]; ok {
			return
		}
		types[typ.Type] = typ

	case *graphql.InputObject:
		if _, ok := types[typ.Name]; ok {
			return
		}
		types[typ.Name] = typ

		for _, field := range typ.InputFields {
			collectTypes(field, types)
		}

	case *graphql.NonNull:
		collectTypes(typ.Type, types)
	}
}

var includeDirective = Directive{
	Description: "Directs the executor to include this field or fragment only when the `if` argument is true.",
	Locations: []DirectiveLocation{
		FIELD,
		FRAGMENT_SPREAD,
		INLINE_FRAGMENT,
	},
	Name: "include",
	Args: []InputValue{
		InputValue{
			Name:        "if",
			Type:        Type{Inner: &graphql.Scalar{Type: "Boolean"}},
			Description: "Included when true.",
		},
	},
}

var skipDirective = Directive{
	Description: "Directs the executor to skip this field or fragment only when the `if` argument is true.",
	Locations: []DirectiveLocation{
		FIELD,
		FRAGMENT_SPREAD,
		INLINE_FRAGMENT,
	},
	Name: "skip",
	Args: []InputValue{
		InputValue{
			Name:        "if",
			Type:        Type{Inner: &graphql.Scalar{Type: "Boolean"}},
			Description: "Skipped when true.",
		},
	},
}

func (s *introspection) registerQuery(schema *schemabuilder.Schema) {
	object := schema.Query()

	object.FieldFunc("__schema", func() *Schema {
		var types []Type

		for _, typ := range s.types {
			types = append(types, Type{Inner: typ})
		}
		sort.Slice(types, func(i, j int) bool { return types[i].Inner.String() < types[j].Inner.String() })

		return &Schema{
			Types:            types,
			QueryType:        &Type{Inner: s.query},
			MutationType:     &Type{Inner: s.mutation},
			SubscriptionType: &Type{Inner: s.subscription},
			Directives:       []Directive{includeDirective, skipDirective},
		}
	})

	object.FieldFunc("__type", func(args struct{ Name string }) *Type {
		if typ, ok := s.types[args.Name]; ok {
			return &Type{Inner: typ}
		}
		return nil
	})
}

func (s *introspection) registerMutation(schema *schemabuilder.Schema) {
	schema.Mutation()
}

func (s *introspection) registerSubscription(schema *schemabuilder.Schema) {
	schema.Subscription()
}

func (s *introspection) schema() *graphql.Schema {
	schema := schemabuilder.NewSchema()
	s.registerDirective(schema)
	s.registerEnumValue(schema)
	s.registerField(schema)
	s.registerInputValue(schema)
	s.registerSubscription(schema)
	s.registerMutation(schema)
	s.registerQuery(schema)
	s.registerSchema(schema)
	s.registerType(schema)

	return schema.MustBuild()
}

// AddIntrospectionToSchema adds the introspection fields to existing schema
func AddIntrospectionToSchema(schema *graphql.Schema) {
	types := make(map[string]graphql.Type)
	collectTypes(schema.Query, types)
	collectTypes(schema.Mutation, types)
	collectTypes(schema.Subscription, types)
	is := &introspection{
		types:        types,
		query:        schema.Query,
		mutation:     schema.Mutation,
		subscription: schema.Subscription,
	}
	isSchema := is.schema()

	query := schema.Query.(*graphql.Object)

	isQuery := isSchema.Query.(*graphql.Object)
	for k, v := range query.Fields {
		isQuery.Fields[k] = v
	}

	schema.Query = isQuery
}

// ComputeSchemaJSON returns the result of executing a GraphQL introspection
// query.
func ComputeSchemaJSON(schemaBuilderSchema schemabuilder.Schema) ([]byte, error) {
	schema := schemaBuilderSchema.MustBuild()
	AddIntrospectionToSchema(schema)

	query, err := graphql.Parse(introspectionQuery, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	if err := graphql.ValidateQuery(context.Background(), schema.Query, query.SelectionSet); err != nil {
		return nil, err
	}

	executor := graphql.Executor{}
	value, err := executor.Execute(context.Background(), schema.Query, nil, query)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(value, "", "  ")
}
