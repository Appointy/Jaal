package schemabuilder

import (
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.appointy.com/jaal/graphql"
)

// argField is a representation of an input parameter field for a function.  It
// must be a field on a struct and will have an associated "argParser" for
// reading an input JSON and filling the struct field.
type argField struct {
	field  reflect.StructField
	parser *argParser
}

// argParser is a struct that holds information for how to deserialize a JSON
// input into a particular go variable.
type argParser struct {
	FromJSON func(interface{}, reflect.Value) error
	Type     reflect.Type
}

// Parse is a convenience function that takes in JSON args and writes them into a new variable type for the argParser.
func (p *argParser) Parse(args interface{}) (interface{}, error) {
	if p == nil {
		return nilParseArguments(args)
	}
	parsed := reflect.New(p.Type).Elem()
	if err := p.FromJSON(args, parsed); err != nil {
		return nil, err
	}
	return parsed.Interface(), nil
}

// nilParseArguments is a default function for parsing args.  It expects to be
// called with nothing, and will return an error if called with non-empty args.
func nilParseArguments(args interface{}) (interface{}, error) {
	if args == nil {
		return nil, nil
	}
	if args, ok := args.(map[string]interface{}); !ok || len(args) != 0 {
		return nil, fmt.Errorf("unexpected args")
	}
	return nil, nil
}

// wrapPtrParser wraps an ArgParser with a helper that will convert the parsed type into a pointer type.
func wrapPtrParser(inner *argParser) *argParser {
	return &argParser{
		FromJSON: func(value interface{}, dest reflect.Value) error {
			if value == nil {
				// optional value
				return nil
			}

			ptr := reflect.New(inner.Type)
			if err := inner.FromJSON(value, ptr.Elem()); err != nil {
				return err
			}
			dest.Set(ptr)
			return nil
		},
		Type: reflect.PtrTo(inner.Type),
	}
}

// getEnumArgParser creates an arg parser for an Enum type.
func (sb *schemaBuilder) getEnumArgParser(typ reflect.Type) (*argParser, graphql.Type) {
	var values []string
	for mapping := range sb.enumMappings[typ].Map {
		values = append(values, mapping)
	}
	return &argParser{FromJSON: func(value interface{}, dest reflect.Value) error {
		asString, ok := value.(string)
		if !ok {
			return errors.New("not a string")
		}
		val, ok := sb.enumMappings[typ].Map[asString]
		if !ok {
			return fmt.Errorf("unknown enum value %v", asString)
		}
		dest.Set(reflect.ValueOf(val).Convert(dest.Type()))
		return nil
	}, Type: typ}, &graphql.Enum{Type: typ.Name(), Values: values, ReverseMap: sb.enumMappings[typ].ReverseMap}

}

// wrapWithZeroValue wraps an ArgParser with a helper that will convert non- provided parameters into the argParser's zero value (basically do nothing).
func wrapWithZeroValue(inner *argParser, fieldArgTyp graphql.Type) (*argParser, graphql.Type) {
	// Make sure the "fieldArgType" we expose in graphQL is a Nullable field.
	if f, ok := fieldArgTyp.(*graphql.NonNull); ok {
		fieldArgTyp = f.Type
	}
	return &argParser{
		FromJSON: func(value interface{}, dest reflect.Value) error {
			if value == nil {
				// optional value
				return nil
			}

			return inner.FromJSON(value, dest)
		},
		Type: inner.Type,
	}, fieldArgTyp
}

// getScalarArgParser creates an arg parser for a scalar type.
func getScalarArgParser(typ reflect.Type) (*argParser, graphql.Type, bool) {
	for match, argParser := range scalarArgParsers {
		if typesIdenticalOrScalarAliases(match, typ) {
			name, ok := getScalar(typ)
			if !ok {
				panic(typ)
			}

			if typ != argParser.Type {
				// The scalar may be a type alias here,
				// so we annotate the parser to output the
				// alias instead of the underlying type.
				newParser := *argParser
				newParser.Type = typ
				argParser = &newParser
			}

			return argParser, &graphql.Scalar{Type: name}, true
		}
	}
	return nil, nil, false
}

// scalarArgParsers are the static arg parsers that we can use for all scalar & static types.
var scalarArgParsers = map[reflect.Type]*argParser{
	reflect.TypeOf(bool(false)): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asBool, ok := value.(bool)
			if !ok {
				if value == nil {
					asBool = false
				} else {
					return errors.New("not a bool")
				}
			}
			dest.Set(reflect.ValueOf(asBool).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(float64(0)): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asFloat, ok := value.(float64)
			if !ok {
				if value == nil {
					asFloat = 0
				} else {
					return errors.New("not a number")
				}
			}
			dest.Set(reflect.ValueOf(asFloat).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(float32(0)): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asFloat, ok := value.(float64)
			if !ok {
				if value == nil {
					asFloat = 0
				} else {
					return errors.New("not a number")
				}
			}
			dest.Set(reflect.ValueOf(float32(asFloat)).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(int64(0)): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asFloat, ok := value.(float64)
			if !ok {
				if value == nil {
					asFloat = 0
				} else {
					return errors.New("not a number")
				}
			}
			dest.Set(reflect.ValueOf(int64(asFloat)).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(int32(0)): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asFloat, ok := value.(float64)
			if !ok {
				if value == nil {
					asFloat = 0
				} else {
					return errors.New("not a number")
				}
			}
			dest.Set(reflect.ValueOf(int32(asFloat)).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(int16(0)): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asFloat, ok := value.(float64)
			if !ok {
				if value == nil {
					asFloat = 0
				} else {
					return errors.New("not a number")
				}
			}
			dest.Set(reflect.ValueOf(int16(asFloat)).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(int8(0)): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asFloat, ok := value.(float64)
			if !ok {
				if value == nil {
					asFloat = 0
				} else {
					return errors.New("not a number")
				}
			}
			dest.Set(reflect.ValueOf(int8(asFloat)).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(uint64(0)): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asFloat, ok := value.(float64)
			if !ok {
				if value == nil {
					asFloat = 0
				} else {
					return errors.New("not a number")
				}
			}
			dest.Set(reflect.ValueOf(int64(asFloat)).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(uint32(0)): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asFloat, ok := value.(float64)
			if !ok {
				if value == nil {
					asFloat = 0
				} else {
					return errors.New("not a number")
				}
			}
			dest.Set(reflect.ValueOf(uint32(asFloat)).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(uint16(0)): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asFloat, ok := value.(float64)
			if !ok {
				if value == nil {
					asFloat = 0
				} else {
					return errors.New("not a number")
				}
			}
			dest.Set(reflect.ValueOf(uint16(asFloat)).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(uint8(0)): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asFloat, ok := value.(float64)
			if !ok {
				if value == nil {
					asFloat = 0
				} else {
					return errors.New("not a number")
				}
			}
			dest.Set(reflect.ValueOf(uint8(asFloat)).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(string("")): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asString, ok := value.(string)
			if !ok {
				if value == nil {
					asString = ""
				} else {
					return errors.New("not a string")
				}
			}
			dest.Set(reflect.ValueOf(asString).Convert(dest.Type()))
			return nil
		},
	},
	reflect.TypeOf(ID{Value: ""}): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			v, ok := value.(string)
			if !ok {
				if value == nil {
					v = ""
				} else {
					return errors.New("not a string")
				}
			}

			dest.Field(0).SetString(v)
			return nil
		},
	},
	reflect.TypeOf(Map{Value: ""}): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			v, ok := value.(string)
			if !ok {
				if value == nil {
					v = ""
				} else {
					return errors.New("not a string")
				}
			}

			dest.Field(0).SetString(v)
			return nil
		},
	},
	reflect.TypeOf(Timestamp(timestamp.Timestamp{})): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			v, ok := value.(string)
			if !ok {
				return errors.New("invalid type expected string")
			}

			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return err
			}

			dest.Field(0).SetInt(int64(t.Unix()))
			dest.Field(1).SetInt(int64(t.Nanosecond()))
			return nil
		},
	},
	reflect.TypeOf(Duration(duration.Duration{})): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			v, ok := value.(float64)
			if !ok {
				return errors.New("invalid type expected number")
			}

			dest.Field(0).SetInt(int64(v))
			dest.Field(1).SetInt(int64(0))
			return nil
		},
	},
	reflect.TypeOf(Bytes{Value: []byte{}}): {
		FromJSON: func(value interface{}, dest reflect.Value) error {
			v, ok := value.(string)
			if !ok {
				return errors.New("invalid type expected string")
			}

			decodedValue, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return err
			}

			dest.Field(0).Set(reflect.ValueOf(decodedValue))
			return nil
		},
	},
}
