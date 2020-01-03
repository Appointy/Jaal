package schemabuilder

import (
	"fmt"
	"reflect"

	"go.appointy.com/jaal/graphql"
)

// Schema is a struct that can be used to build out a GraphQL schema.  Functions
// can be registered against the "Mutation", "Query" and "Subscription" objects in order to
// build out a full GraphQL schema.
type Schema struct {
	objects      map[string]*Object
	enumTypes    map[reflect.Type]*EnumMapping
	inputObjects map[string]*InputObject
}

// NewSchema creates a new schema.
func NewSchema() *Schema {
	schema := &Schema{
		objects:      make(map[string]*Object),
		inputObjects: make(map[string]*InputObject),
	}

	return schema
}

// Enum registers an enumType in the schema. The val should be any arbitrary value
// of the enumType to be used for reflection, and the enumMap should be
// the corresponding map of the enums.
//
// For example a enum could be declared as follows:
//   type enumType int32
//   const (
//	  one   enumType = 1
//	  two   enumType = 2
//	  three enumType = 3
//   )
//
// Then the Enum can be registered as:
//   s.Enum(enumType(1), map[string]interface{}{
//     "one":   enumType(1),
//     "two":   enumType(2),
//     "three": enumType(3),
//   })
func (s *Schema) Enum(val interface{}, enumMap interface{}) {
	typ := reflect.TypeOf(val)
	if s.enumTypes == nil {
		s.enumTypes = make(map[reflect.Type]*EnumMapping)
	}

	eMap, rMap := getEnumMap(enumMap, typ)
	s.enumTypes[typ] = &EnumMapping{Map: eMap, ReverseMap: rMap}
}

func getEnumMap(enumMap interface{}, typ reflect.Type) (map[string]interface{}, map[interface{}]string) {
	rMap := make(map[interface{}]string)
	eMap := make(map[string]interface{})
	v := reflect.ValueOf(enumMap)
	if v.Kind() == reflect.Map {
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			valInterface := val.Interface()
			if reflect.TypeOf(valInterface).Kind() != typ.Kind() {
				panic("types are not equal")
			}
			if key.Kind() == reflect.String {
				mapVal := reflect.ValueOf(valInterface).Convert(typ)
				eMap[key.String()] = mapVal.Interface()
			} else {
				panic("keys are not strings")
			}
		}
	} else {
		panic("enum function not passed a map")
	}

	for key, val := range eMap {
		rMap[val] = key
	}
	return eMap, rMap

}

// Object registers a struct as a GraphQL Object in our Schema.
// (https://facebook.github.io/graphql/June2018/#sec-Objects)
// We'll read the fields of the struct to determine it's basic "Fields" and
// we'll return an Object struct that we can use to register custom
// relationships and fields on the object.
func (s *Schema) Object(name string, typ interface{}) *Object {
	if object, ok := s.objects[name]; ok {
		if reflect.TypeOf(object.Type) != reflect.TypeOf(typ) {
			var t = reflect.TypeOf(object.Type)
			panic("re-registered object with different type, already registered type :" + fmt.Sprintf(" %s.%s", t.PkgPath(), t.Name()))
		}
		return object
	}
	object := &Object{
		Name: name,
		Type: typ,
	}
	s.objects[name] = object
	return object
}

// GetObject gets a registered object. It is used to create linkings between different objects
func (s *Schema) GetObject(name string, typ interface{}) (*Object, error) {
	object, ok := s.objects[name]
	if ok && reflect.TypeOf(object.Type) == reflect.TypeOf(typ) {
		return object, nil
	}

	return nil, fmt.Errorf("%v of type %v is not a registered Object on schema", name, typ)
}

// InputObject registers a struct as inout object which can be passed as an argument to a query or mutation
// We'll read through the fields of the struct and create argument parsers to fill the data from graphQL JSON input
func (s *Schema) InputObject(name string, typ interface{}) *InputObject {
	if inputObject, ok := s.inputObjects[name]; ok {
		if reflect.TypeOf(inputObject.Type) != reflect.TypeOf(typ) {
			var t = reflect.TypeOf(inputObject.Type)
			panic("re-registered input object with different type, already registered type :" + fmt.Sprintf(" %s.%s", t.PkgPath(), t.Name()))
		}
	}
	inputObject := &InputObject{
		Name:   name,
		Type:   typ,
		Fields: map[string]interface{}{},
	}
	s.inputObjects[name] = inputObject

	return inputObject
}

type query struct{}

// Query returns an Object struct that we can use to register all the top level
// graphql query functions we'd like to expose.
func (s *Schema) Query() *Object {
	return s.Object("Query", query{})
}

type mutation struct{}

// Mutation returns an Object struct that we can use to register all the top level
// graphql mutation functions we'd like to expose.
func (s *Schema) Mutation() *Object {
	return s.Object("Mutation", mutation{})
}

type Subscription struct {
	Payload []byte
}

// Subscription returns an Object struct that we can use to register all the top level
// graphql subscription functions we'd like to expose.
func (s *Schema) Subscription() *Object {
	return s.Object("Subscription", Subscription{})
}

// Build takes the schema we have built on our Query, Mutation and Subscription starting points and builds a full graphql.Schema
// We can use graphql.Schema to execute and run queries. Essentially we read through all the methods we've attached to our
// Query, Mutation and Subscription Objects and ensure that those functions are returning other Objects that we can resolve in our GraphQL graph.
func (s *Schema) Build() (*graphql.Schema, error) {
	sb := &schemaBuilder{
		types:        make(map[reflect.Type]graphql.Type),
		objects:      make(map[reflect.Type]*Object),
		enumMappings: s.enumTypes,
		typeCache:    make(map[reflect.Type]cachedType, 0),
		inputObjects: make(map[reflect.Type]*InputObject, 0),
	}

	for _, object := range s.objects {
		typ := reflect.TypeOf(object.Type)
		if typ.Kind() != reflect.Struct {
			return nil, fmt.Errorf("object.Type should be a struct, not %s", typ.String())
		}

		if _, ok := sb.objects[typ]; ok {
			return nil, fmt.Errorf("duplicate object for %s", typ.String())
		}

		sb.objects[typ] = object
	}

	for _, inputObject := range s.inputObjects {
		typ := reflect.TypeOf(inputObject.Type)
		if typ.Kind() != reflect.Struct {
			return nil, fmt.Errorf("inputObject.Type should be a struct, not %s", typ.String())
		}

		if _, ok := sb.inputObjects[typ]; ok {
			return nil, fmt.Errorf("duplicate inputObject for %s", typ.String())
		}

		sb.inputObjects[typ] = inputObject
	}

	queryTyp, err := sb.getType(reflect.TypeOf(&query{}))
	if err != nil {
		return nil, err
	}
	mutationTyp, err := sb.getType(reflect.TypeOf(&mutation{}))
	if err != nil {
		return nil, err
	}
	subscriptionTyp, err := sb.getType(reflect.TypeOf(&Subscription{}))
	if err != nil {
		return nil, err
	}
	return &graphql.Schema{
		Query:        queryTyp,
		Mutation:     mutationTyp,
		Subscription: subscriptionTyp,
	}, nil
}

//MustBuild builds a schema and panics if an error occurs.
func (s *Schema) MustBuild() *graphql.Schema {
	built, err := s.Build()
	if err != nil {
		panic(err)
	}
	return built
}

// Clone creates a deep copy of schema and panics if it fails
func (s *Schema) Clone() *Schema {
	copy := Schema{
		objects:      make(map[string]*Object, len(s.objects)),
		inputObjects: make(map[string]*InputObject, len(s.inputObjects)),
		enumTypes:    make(map[reflect.Type]*EnumMapping, len(s.enumTypes)),
	}

	for key, value := range s.objects {
		copy.objects[key] = copyObject(value)
	}

	for key, value := range s.inputObjects {
		copy.inputObjects[key] = copyInputObject(value)
	}

	for key, value := range s.enumTypes {
		copy.enumTypes[key] = copyEnumMappings(value)
	}

	return &copy
}

func copyObject(object *Object) *Object {
	copy := &Object{
		Name:        object.Name,
		Description: object.Description,
		Type:        object.Type,
		Methods:     make(Methods, len(object.Methods)),
	}

	for name, m := range object.Methods {
		copy.Methods[name] = &method{
			MarkedNonNullable: m.MarkedNonNullable,
			Fn:                m.Fn,
		}
	}

	return copy
}

func copyInputObject(input *InputObject) *InputObject {
	copy := &InputObject{
		Name:   input.Name,
		Type:   input.Type,
		Fields: make(map[string]interface{}),
	}

	for name, field := range input.Fields {
		copy.Fields[name] = field
	}

	return copy
}

func copyEnumMappings(mapping *EnumMapping) *EnumMapping {
	enum := &EnumMapping{
		Map:        make(map[string]interface{}, len(mapping.Map)),
		ReverseMap: make(map[interface{}]string, len(mapping.ReverseMap)),
	}

	for key, value := range mapping.Map {
		enum.Map[key] = value
	}

	for key, value := range mapping.ReverseMap {
		enum.ReverseMap[key] = value
	}

	return enum
}
