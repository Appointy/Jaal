package schemabuilder

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
)

//Object - an Object represents a Go type and set of methods to be converted into an Object in a GraphQL schema.
type Object struct {
	Name        string // Optional, defaults to Type's name.
	Description string
	Type        interface{}
	Methods     Methods // Deprecated, use FieldFunc instead.

	key string
}

// Key registers the key field on an object. The field should be specified by the name of the graphql field.
// For example, for an object User:
//   type struct User {
//	   UserKey int64
//   }
// The key will be registered as:
// object.Key("userKey")
func (s *Object) Key(f string) {
	s.key = f
}

// InputObject represents the input objects passed in queries,mutations and subscriptions
type InputObject struct {
	Name   string
	Type   interface{}
	Fields map[string]interface{}
}

// A Methods map represents the set of methods exposed on a Object.
type Methods map[string]*method

type method struct {
	MarkedNonNullable bool
	Fn                interface{}
}

// EnumMapping is a representation of an enum that includes both the mapping and reverse mapping.
type EnumMapping struct {
	Map        map[string]interface{}
	ReverseMap map[interface{}]string
}

// InterfaceObj is a representation of graphql interface
type InterfaceObj struct {
	Struct reflect.Type
	Type   interface{}
}

// Interface is a special marker struct that can be embedded into to denote that a type should be
// treated as a interface type by the schemabuilder
type Interface struct{}

// Union is a special marker struct that can be embedded into to denote
// that a type should be treated as a union type by the schemabuilder.
//
// For example, to denote that a return value that may be a *Asset or
// *Vehicle might look like:
//   type GatewayUnion struct {
//     schemabuilder.Union
//     *Asset
//     *Vehicle
//   }
//
// Fields returning a union type should expect to return this type as a
// one-hot struct, i.e. only Asset or Vehicle should be specified, but not both.
type Union struct{}

var unionType = reflect.TypeOf(Union{})

// FieldFunc exposes a field on an object. The function f can take a number of
// optional arguments:
// func([ctx context.Context], [o *Type], [args struct {}]) ([Result], [error])
//
// For example, for an object of type User, a fullName field might take just an
// instance of the object:
//    user.FieldFunc("fullName", func(u *User) string {
//       return u.FirstName + " " + u.LastName
//    })
//
// An addUser mutation field might take both a context and arguments:
//    mutation.FieldFunc("addUser", func(ctx context.Context, args struct{
//        FirstName string
//        LastName  string
//    }) (int, error) {
//        userID, err := db.AddUser(ctx, args.FirstName, args.LastName)
//        return userID, err
//    })
func (s *Object) FieldFunc(name string, f interface{}) {
	if s.Methods == nil {
		s.Methods = make(Methods)
	}

	m := &method{Fn: f}

	if _, ok := s.Methods[name]; ok {
		panic("duplicate method")
	}
	s.Methods[name] = m
}

// FieldFunc is used to expose the fields of an input object and determine the method to fill it
// type ServiceProvider struct {
// 	Id                   string
// 	FirstName            string
// }
// inputObj := schema.InputObject("serviceProvider", ServiceProvider{})
// inputObj.FieldFunc("id", func(target *ServiceProvider, source *schemabuilder.ID) {
// 	target.Id = source.Value
// })
// inputObj.FieldFunc("firstName", func(target *ServiceProvider, source *string) {
// 	target.FirstName = *source
// })
// The target variable of the function should be pointer
func (io *InputObject) FieldFunc(name string, function interface{}) {
	funcTyp := reflect.TypeOf(function)

	if funcTyp.NumIn() != 2 {
		panic(fmt.Errorf("can not register field %v on %v as number of input argument should be 2", name, io.Name))
	}

	sourceTyp := funcTyp.In(0)
	if sourceTyp.Kind() != reflect.Ptr {
		panic(fmt.Errorf("can not register %s on input object %s as the first argument of the function is not a pointer type", name, io.Name))
	}

	if funcTyp.NumOut() > 2 {
		panic(fmt.Errorf("can not register field %v on %v as number of output parameters should be less than 2", name, io.Name))
	}

	io.Fields[name] = function
}

// UnmarshalFunc is used to unmarshal scalar value from JSON
type UnmarshalFunc func(value interface{}, dest reflect.Value) error

// RegisterScalar is used to register custom scalars.
//
// For example, to register a custom ID type,
// type ID struct {
// 		Value string
// }
//
// Implement JSON Marshalling
// func (id ID) MarshalJSON() ([]byte, error) {
//  return strconv.AppendQuote(nil, string(id.Value)), nil
// }
//
// Register unmarshal func
// func init() {
//	typ := reflect.TypeOf((*ID)(nil)).Elem()
//	if err := schemabuilder.RegisterScalar(typ, "ID", func(value interface{}, d reflect.Value) error {
//		v, ok := value.(string)
//		if !ok {
//			return errors.New("not a string type")
//		}
//
//		d.Field(0).SetString(v)
//		return nil
//	}); err != nil {
//		panic(err)
//	}
//}
func RegisterScalar(typ reflect.Type, name string, uf UnmarshalFunc) error {
	if typ.Kind() == reflect.Ptr {
		return errors.New("type should not be of pointer type")
	}

	if uf == nil {
		// Slow fail safe to avoid reflection code by package users
		if !reflect.PtrTo(typ).Implements(reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()) {
			return errors.New("either UnmarshalFunc should be provided or the provided type should implement json.Unmarshaler interface")
		}

		f, _ := reflect.PtrTo(typ).MethodByName("UnmarshalJSON")

		uf = func(value interface{}, dest reflect.Value) error {
			var x interface{}
			switch v := value.(type) {
			case []byte:
				x = v
			case string:
				x = []byte(v)
			case float64:
				x = []byte(strconv.FormatFloat(v, 'g', -1, 64))
			case int64:
				x = []byte(strconv.FormatInt(v, 10))
			case bool:
				if v {
					x = []byte{'t', 'r', 'u', 'e'}
				} else {
					x = []byte{'f', 'a', 'l', 's', 'e'}
				}
			default:
				return errors.New("unknown type")
			}

			if err := f.Func.Call([]reflect.Value{dest.Addr(), reflect.ValueOf(x)})[0].Interface(); err != nil {
				return err.(error)
			}

			return nil
		}
	}

	scalars[typ] = name
	scalarArgParsers[typ] = &argParser{
		FromJSON: uf,
	}

	return nil
}

// ID is the graphql ID scalar
type ID struct {
	Value string
}

// MarshalJSON implements JSON Marshalling used to generate the output
func (id ID) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, string(id.Value)), nil
}

// isScalarType checks whether a reflect.Type is scalar or not
func isScalarType(t reflect.Type) bool {
	_, ok := scalars[t]
	return ok
}

// typesIdenticalOrScalarAliases checks whether a & b are same scalar
func typesIdenticalOrScalarAliases(a, b reflect.Type) bool {
	return a == b || (a.Kind() == b.Kind() && (a.Kind() != reflect.Struct) && (a.Kind() != reflect.Map) && isScalarType(a))
}

//Timestamp handles the time
type Timestamp timestamp.Timestamp

// MarshalJSON implements JSON Marshalling used to generate the output
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, string(time.Unix(t.Seconds, int64(t.Nanos)).Format(time.RFC3339))), nil
}

//Map handles maps
type Map struct {
	Value string
}

// MarshalJSON implements JSON Marshalling used to generate the output
func (m Map) MarshalJSON() ([]byte, error) {
	v := base64.StdEncoding.EncodeToString([]byte(m.Value))
	d, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return d, nil
}

//Duration handles the duration
type Duration duration.Duration

// MarshalJSON implements JSON Marshalling used to generate the output
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Itoa(int(d.Seconds))), nil
}

//Bytes handles the duration
type Bytes struct {
	Value []byte
}

// MarshalJSON implements JSON Marshalling used to generate the output
func (b Bytes) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(b.Value)
	if err != nil {
		return nil, err
	}
	return data, nil
}
