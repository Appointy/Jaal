package pb

import (
	"context"

	duration "github.com/golang/protobuf/ptypes/duration"
	empty "github.com/golang/protobuf/ptypes/empty"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"go.appointy.com/appointy/jaal/schemabuilder"
)

func RegisterTypes(schema *schemabuilder.Schema) {
	registerClass(schema)
	registerCreateClassReq(schema)
	registerGetClassReq(schema)
	registerClassType(schema)
}

func registerClassType(schema *schemabuilder.Schema) {
	schema.Enum(ClassType(0), map[string]interface{}{
		"INVALID": ClassType(0),
		"REGULAR": ClassType(1),
		"SERIES":  ClassType(2),
	})
}

func registerClass(schema *schemabuilder.Schema) {
	obj := schema.Object("Class", Class{})
	obj.FieldFunc("id", func(ctx context.Context, in *Class) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("area", func(ctx context.Context, in *Class) float32 {
		return in.Area
	})
	obj.FieldFunc("strength", func(ctx context.Context, in *Class) int32 {
		return in.Strength
	})
	obj.FieldFunc("isDeleted", func(ctx context.Context, in *Class) bool {
		return in.IsDeleted
	})
	obj.FieldFunc("type", func(ctx context.Context, in *Class) ClassType {
		return in.Type
	})
	obj.FieldFunc("instructors", func(ctx context.Context, in *Class) []*ServiceProvider {
		return in.Instructors
	})
	obj.FieldFunc("metadata", func(ctx context.Context, in *Class) map[string]string {
		return in.Metadata
	})
	obj.FieldFunc("parent", func(ctx context.Context, in *Class) string {
		return in.Parent
	})
	obj.FieldFunc("charge", func(ctx context.Context, in *Class) *unionClassCharge {
		switch v := in.Charge.(type) {

		case *Class_PerInstance:
			return &unionClassCharge{
				Class_PerInstance: v,
			}
		case *Class_Lumpsum:
			return &unionClassCharge{
				Class_Lumpsum: v,
			}
		}
		return nil

		//Using fragments
	})
	obj.FieldFunc("startDate", func(ctx context.Context, in *Class) *timestamp.Timestamp {
		return in.StartDate
	})
	obj.FieldFunc("duration", func(ctx context.Context, in *Class) *duration.Duration {
		return in.Duration
	})
	obj.FieldFunc("empty", func(ctx context.Context, in *Class) *empty.Empty {
		return in.Empty
	})

	obj = schema.Object("ServiceProvider", ServiceProvider{})
	obj.FieldFunc("id", func(ctx context.Context, in *ServiceProvider) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("firstName", func(ctx context.Context, in *ServiceProvider) string {
		return in.FirstName
	})

	obj = schema.Object("Class_PerInstance", Class_PerInstance{})
	obj.FieldFunc("perInstance", func(ctx context.Context, in *Class_PerInstance) string {
		return in.PerInstance
	})

	obj = schema.Object("Class_Lumpsum", Class_Lumpsum{})
	obj.FieldFunc("lumpsum", func(ctx context.Context, in *Class_Lumpsum) int32 {
		return in.Lumpsum
	})
}

func registerCreateClassReq(schema *schemabuilder.Schema) {
	inputObj := schema.InputObject("createClassReq", CreateClassReq{})
	inputObj.FieldFunc("parent", func(target *CreateClassReq, source *string) {
		target.Parent = *source
	})
	inputObj.FieldFunc("class", func(target *CreateClassReq, source *Class) {
		target.Class = source
	})

	inputObj = schema.InputObject("class", Class{})
	inputObj.FieldFunc("id", func(target *Class, source *schemabuilder.ID) {
		target.Id = source.Value
	})
	inputObj.FieldFunc("area", func(target *Class, source *float32) {
		target.Area = *source
	})
	inputObj.FieldFunc("area", func(target *Class, source *float32) {
		target.Area = *source
	})
	inputObj.FieldFunc("strength", func(target *Class, source *int32) {
		target.Strength = *source
	})
	inputObj.FieldFunc("type", func(target *Class, source *ClassType) {
		target.Type = *source
	})
	inputObj.FieldFunc("instructors", func(target *Class, source []*ServiceProvider) {
		target.Instructors = source
	})
	inputObj.FieldFunc("metadata", func(target *Class, source *map[string]string) {
		target.Metadata = *source
	})
	inputObj.FieldFunc("parent", func(target *Class, source *string) {
		target.Parent = *source
	})
	inputObj.FieldFunc("classLumpsum", func(target *Class, source *Class_Lumpsum) {
		target.Charge = source
	})
	inputObj.FieldFunc("classPerInstance", func(target *Class, source *Class_PerInstance) {
		target.Charge = source
	})
	inputObj.FieldFunc("duration", func(target *Class, source *duration.Duration) {
		target.Duration = source
	})
	inputObj.FieldFunc("startDate", func(target *Class, source *timestamp.Timestamp) {
		target.StartDate = source
	})

	inputObj = schema.InputObject("serviceProvider", ServiceProvider{})
	inputObj.FieldFunc("id", func(target *ServiceProvider, source *schemabuilder.ID) {
		target.Id = source.Value
	})
	inputObj.FieldFunc("firstName", func(target *ServiceProvider, source *string) {
		target.FirstName = *source
	})

	inputObj = schema.InputObject("classLumpsum", Class_Lumpsum{})
	inputObj.FieldFunc("lumpsum", func(target *Class_Lumpsum, source *int32) {
		target.Lumpsum = *source
	})

	inputObj = schema.InputObject("classPerInstance", Class_PerInstance{})
	inputObj.FieldFunc("perInstance", func(target *Class_PerInstance, source *string) {
		target.PerInstance = *source
	})

}

func registerGetClassReq(schema *schemabuilder.Schema) {
	inputObj := schema.InputObject("getClassReq", GetClassReq{})
	inputObj.FieldFunc("id", func(target *GetClassReq, source *schemabuilder.ID) {
		target.Id = source.Value
	})
}

type unionClassCharge struct {
	schemabuilder.Union
	*Class_PerInstance
	*Class_Lumpsum
}

// {
//     "query": "mutation CreateClass{createClass(in:{parent:\"par_01DBCE889ZHX6KTGPZWGEBT90T\", class:{id:\"\",instructors:[{id:\"sp_123455\",firstName:\"Anuj\"}],metadata:\"e30=\",parent:\"\",area:10.00,strength:10,isDeleted:\"false\",type:\"REGULAR\",classLumpsum:{lumpsum:1000},classPerInstance:{perInstance:\"100per10instance\"}}}){id,instructors{id,firstName},charge{...on Class_Lumpsum{lumpsum} ...on Class_PerInstance{perInstance}}}}"
// }
