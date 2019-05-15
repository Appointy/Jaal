package gtypes

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.appointy.com/appointy/jaal/schemabuilder"
)

//Schema is used to build the graphql schema
var Schema = schemabuilder.NewSchema()

func init() {
	RegisterWellKnownTypes()
}

//RegisterWellKnownTypes registers the commonly used scalars
func RegisterWellKnownTypes() {
	RegisterDuration()
	RegisterTimestamp()
	RegisterEmpty()
	RegisterStringStringMap()
}

// RegisterStringStringMap registers the map[string]string as a scalar
func RegisterStringStringMap() {
	typ := reflect.TypeOf(map[string]string{})
	schemabuilder.RegisterScalar(typ, "Metadata", func(value interface{}, target reflect.Value) error {
		v, ok := value.(string)
		if !ok {
			return errors.New("invalid type expected a string")
		}

		unq, err := unquote(v)
		if err != nil {
			return err
		}

		data := make(map[string]string, 10)
		if err := json.Unmarshal([]byte(unq), &data); err != nil {
			return err
		}

		target.Set(reflect.ValueOf(data))
		return nil
	})
}

// RegisterEmpty registers empty as an scalar type
func RegisterEmpty() {
	typ := reflect.TypeOf((*empty.Empty)(nil)).Elem()
	schemabuilder.RegisterScalar(typ, "Empty", func(value interface{}, target reflect.Value) error {
		return nil
	})
}

// RegisterDuration registers duration as an scalar type
func RegisterDuration() {
	typ := reflect.TypeOf((*duration.Duration)(nil)).Elem()
	schemabuilder.RegisterScalar(typ, "Duration", func(value interface{}, target reflect.Value) error {
		v, ok := value.(string)
		if !ok {
			return errors.New("invalid type expected a string")
		}

		unq, err := unquote(v)
		if err != nil {
			return err
		}

		d, err := time.ParseDuration(unq)
		if err != nil {
			return fmt.Errorf("bad Duration: %v", err)
		}

		ns := d.Nanoseconds()
		s := ns / 1e9
		ns %= 1e9
		target.Field(0).SetInt(s)
		target.Field(1).SetInt(ns)

		return nil
	})
}

// RegisterTimestamp registers timestamp as an scalar type
func RegisterTimestamp() {
	typ := reflect.TypeOf((*timestamp.Timestamp)(nil)).Elem()
	schemabuilder.RegisterScalar(typ, "Timestamp", func(value interface{}, target reflect.Value) error {
		v, ok := value.(string)
		if !ok {
			return errors.New("invalid type expected a string")
		}

		unq, err := unquote(v)
		if err != nil {
			return err
		}

		t, err := time.Parse(time.RFC3339Nano, unq)
		if err != nil {
			return fmt.Errorf("bad Timestamp: %v", err)
		}

		target.Field(0).SetInt(t.Unix())
		target.Field(1).SetInt(int64(t.Nanosecond()))
		return nil
	})
}

func unquote(s string) (string, error) {
	var ret string
	err := json.Unmarshal([]byte(s), &ret)
	return ret, err
}
