package gtypes

import (
	"context"
	"reflect"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/iancoleman/strcase"
	"go.appointy.com/jaal/schemabuilder"
	"google.golang.org/genproto/protobuf/field_mask"
)

//Schema is used to build the graphql schema
var Schema = schemabuilder.NewSchema()

func init() {
	RegisterEmpty()
	RegisterInputFieldMask()
	RegisterPayloadFieldMask()
}

// RegisterEmpty registers empty as an scalar type
func RegisterEmpty() {
	typ := reflect.TypeOf(empty.Empty{})
	schemabuilder.RegisterScalar(typ, "Empty", func(value interface{}, target reflect.Value) error {
		return nil
	})
}

// RegisterInputFieldMask registers FieldMask as GraphQL Input
func RegisterInputFieldMask() {
	input := Schema.InputObject("FieldMaskInput", field_mask.FieldMask{})
	input.FieldFunc("paths", func(target *field_mask.FieldMask, source []string) {
		target.Paths = source
	})

}

// RegisterPayloadFieldMask registers FieldMask as GraphQL Input
func RegisterPayloadFieldMask() {
	payload := Schema.Object("FieldMask", field_mask.FieldMask{})
	payload.FieldFunc("paths", func(ctx context.Context, in *field_mask.FieldMask) []string {
		return in.Paths
	})
}

// ModifyFieldMask modifies the paths of recieved field mask to snake case
func ModifyFieldMask(mask *field_mask.FieldMask) *field_mask.FieldMask {
	modified := &field_mask.FieldMask{Paths: []string{}}

	for _, path := range mask.GetPaths() {
		modified.Paths = append(modified.Paths, strcase.ToSnake(path))
	}

	return modified
}
