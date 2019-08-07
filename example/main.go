package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

func init() {
	var typ = reflect.TypeOf(time.Time{})
	_ = schemabuilder.RegisterScalar(typ, "DateTime", func(value interface{}, dest reflect.Value) error {
		v, ok := value.(string)
		if !ok {
			return errors.New("invalid type expected string")
		}

		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}

		dest.Set(reflect.ValueOf(t))

		return nil
	})

}

type Server struct {
	Characters []*Character
}

type Character struct {
	Id          string
	Name        string
	Type        Type
	DateOfBirth time.Time
	Metadata    map[string]string
}

type Type int32

const (
	WIZARD Type = iota
	MUGGLE
	GOBLIN
	HOUSE_ELF
)

type CreateCharacterRequest struct {
	Name        string
	Type        Type
	DateOfBirth time.Time
	Metadata    map[string]string
}

func RegisterPayload(schema *schemabuilder.Schema) {
	payload := schema.Object("Character", Character{})
	payload.FieldFunc("id", func(ctx context.Context, in *Character) *schemabuilder.ID {
		return &schemabuilder.ID{Value: in.Id}
	})
	payload.FieldFunc("name", func(ctx context.Context, in *Character) string {
		return in.Name
	})
	payload.FieldFunc("type", func(ctx context.Context, in *Character) Type {
		return in.Type
	})
	payload.FieldFunc("dateOfBirth", func(ctx context.Context, in *Character) time.Time {
		return in.DateOfBirth
	})
	payload.FieldFunc("metadata", func(ctx context.Context, in *Character) (*schemabuilder.Map, error) {
		data, err := json.Marshal(in.Metadata)
		if err != nil {
			return nil, err
		}

		return &schemabuilder.Map{Value: string(data)}, nil
	})
}

func RegisterInput(schema *schemabuilder.Schema) {
	input := schema.InputObject("CreateCharacterRequest", CreateCharacterRequest{})
	input.FieldFunc("name", func(target *CreateCharacterRequest, source string) {
		target.Name = source
	})
	input.FieldFunc("type", func(target *CreateCharacterRequest, source Type) {
		target.Type = source
	})
	input.FieldFunc("dateOfBirth", func(target *CreateCharacterRequest, source time.Time) {
		target.DateOfBirth = source
	})
	input.FieldFunc("metadata", func(target *CreateCharacterRequest, source schemabuilder.Map) error {
		v := source.Value

		decodedValue, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return err
		}

		data := make(map[string]string)
		if err := json.Unmarshal(decodedValue, &data); err != nil {
			return err
		}

		target.Metadata = data
		return nil
	})
}

func RegisterEnum(schema *schemabuilder.Schema) {
	schema.Enum(Type(0), map[string]interface{}{
		"WIZARD":    Type(0),
		"MUGGLE":    Type(1),
		"GOBLIN":    Type(2),
		"HOUSE_ELF": Type(3),
	})
}

func (s *Server) RegisterOperations(schema *schemabuilder.Schema) {
	schema.Query().FieldFunc("character", func(ctx context.Context, args struct {
		Id *schemabuilder.ID
	}) *Character {
		for _, ch := range s.Characters {
			if ch.Id == args.Id.Value {
				return ch
			}
		}

		return nil
	})

	schema.Query().FieldFunc("characters", func(ctx context.Context, args struct{}) []*Character {
		return s.Characters
	})

	schema.Mutation().FieldFunc("createCharacter", func(ctx context.Context, args struct {
		Input *CreateCharacterRequest
	}) *Character {
		ch := &Character{
			Id:          uuid.Must(uuid.NewUUID()).String(),
			Name:        args.Input.Name,
			Type:        args.Input.Type,
			DateOfBirth: args.Input.DateOfBirth,
			Metadata:    args.Input.Metadata,
		}
		s.Characters = append(s.Characters, ch)

		return ch
	})
}

func main() {
	sb := schemabuilder.NewSchema()
	RegisterPayload(sb)
	RegisterInput(sb)
	RegisterEnum(sb)

	s := &Server{
		Characters: []*Character{{
			Id:          "015f13a5-cf9b-49d7-b457-6113bcf8fd56",
			Name:        "Harry Potter",
			Type:        WIZARD,
			DateOfBirth: time.Date(1980, time.July, 31, 0, 0, 0, 0, time.Local),
		}},
	}

	s.RegisterOperations(sb)
	schema, err := sb.Build()
	if err != nil {
		log.Fatalln(err)
	}

	introspection.AddIntrospectionToSchema(schema)

	http.Handle("/graphql", jaal.HTTPHandler(schema))
	log.Println("Running")
	if err := http.ListenAndServe(":9000", nil); err != nil {
		panic(err)
	}
}
