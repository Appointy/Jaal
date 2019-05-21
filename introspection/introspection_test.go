package introspection_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.appointy.com/appointy/jaal/introspection"
	"go.appointy.com/appointy/jaal/schemabuilder"
)

type User struct {
	Name     string
	MaybeAge *int64
}

type Vehicle struct {
	Name  string
	Speed int64
}
type Asset struct {
	Name         string
	BatteryLevel int64
}

type Gateway struct {
	schemabuilder.Union

	*Vehicle
	*Asset
}

type enumType int32

func registerTypes(schema *schemabuilder.Schema) {
	obj := schema.Object("user", User{})
	obj.FieldFunc("name", func(in User) string {
		return in.Name
	})
	obj.FieldFunc("mayBeAge", func(in User) *int64 {
		return in.MaybeAge
	})

	obj = schema.Object("asset", Asset{})
	obj.FieldFunc("name", func(in Asset) string {
		return in.Name
	})
	obj.FieldFunc("batteryLevel", func(in Asset) int64 {
		return in.BatteryLevel
	})

	obj = schema.Object("vehicle", Vehicle{})
	obj.FieldFunc("name", func(in Vehicle) string {
		return in.Name
	})
	obj.FieldFunc("speed", func(in Vehicle) int64 {
		return in.Speed
	})

	inputObject := schema.InputObject("user", User{})
	inputObject.FieldFunc("name", func(in *User, name string) {
		in.Name = name
	})
	inputObject.FieldFunc("mayBeAge", func(in *User, age int64) {
		in.MaybeAge = &age
	})
}

func makeSchema() *schemabuilder.Schema {
	schema := schemabuilder.NewSchema()
	registerTypes(schema)

	user := schema.Object("user", User{})
	user.Key("name")
	var enumField enumType
	schema.Enum(enumField, map[string]enumType{
		"random":  enumType(3),
		"random1": enumType(2),
		"random2": enumType(1),
	})
	query := schema.Query()
	query.FieldFunc("me", func() User {
		return User{Name: "me"}
	})
	query.FieldFunc("noone", func() *User {
		return &User{Name: "me"}
	})
	query.FieldFunc("nullableUser", func() (*User, error) {
		return nil, nil
	})
	query.FieldFunc("usersConnection", func() ([]User, error) {
		return nil, nil
	})
	query.FieldFunc("usersConnectionPtr", func() ([]*User, error) {
		return nil, nil
	})

	query.FieldFunc("gateway", func() (*Gateway, error) {
		return nil, nil
	})

	// Add a non-null field after "noone" to test that caching
	// mechanism in schemabuilder chooses the correct type
	// for the return value.
	query.FieldFunc("viewer", func() (User, error) {
		return User{Name: "me"}, nil
	})

	user.FieldFunc("friends", func(u *User) []*User {
		return nil
	})
	user.FieldFunc("greet", func(args struct {
		Other     string
		Include   *User
		Enumfield enumType
		Optional  string `graphql:",optional"`
	}) string {
		return ""
	})

	mutation := schema.Mutation()
	mutation.FieldFunc("sayHi", func() {})

	return schema
}

func TestComputeSchemaJSON(t *testing.T) {
	schemaBuilderSchema := makeSchema()

	actualBytes, err := introspection.ComputeSchemaJSON(*schemaBuilderSchema)
	require.NoError(t, err)

	var actual map[string]interface{}
	json.Unmarshal(actualBytes, &actual)
}
