# Jaal - Develop spec compliant GraphQL servers

Jaal is a go framework for building spec compliant GraphQL servers. Jaal has support for all the graphql scalar types and builds the schema from registered objects using reflection. Jaal is inspired from Thunder by Samsara.

## Features

* In-built support for graphQL scalars
* In-built support for maps
* Custom Scalar registration
* Input, Payload and enum registrations
* Custom field registration on objects
* Interface Support
* Union Support
* In build include and skip directives
* Protocol buffers API generation

## Getting Started

The following example depicts how to build a simple graphQL server using jaal.

``` Go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/google/uuid"
    "go.appointy.com/jaal"
    "go.appointy.com/jaal/introspection"
    "go.appointy.com/jaal/schemabuilder"
)

type Server struct {
    Characters []*Character
}

type Character struct {
    Id   string
    Name string
    Type Type
}

type Type int32

const (
    WIZARD Type = iota
    MUGGLE
    GOBLIN
    HOUSE_ELF
)

type CreateCharacterRequest struct {
    Name string
    Type Type
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
}

func RegisterInput(schema *schemabuilder.Schema) {
    input := schema.InputObject("CreateCharacterRequest", CreateCharacterRequest{})
    input.FieldFunc("name", func(target *Character, source string) {
        target.Name = source
    })
    input.FieldFunc("type", func(target *Character, source Type) {
        target.Type = source
    })
}

func RegisterEnum(schema *schemabuilder.Schema) {
    schema.Enum(Type(0), map[string]interface{}{
        "WIZARD":    Type(0),
        "MUGGLE":    Type(0),
        "GOBLIN":    Type(0),
        "HOUSE_ELF": Type(0),
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
            Id:   uuid.Must(uuid.NewUUID()).String(),
            Name: args.Input.Name,
            Type: args.Input.Type,
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
            Id:   "015f13a5-cf9b-49d7-b457-6113bcf8fd56",
            Name: "Harry Potter",
            Type: WIZARD,
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
```

In the above example, we registered all the operations, inputs & payloads on the schema. We also registered the fields we wanted to expose on the schema. FIeld registration allows us to control the way in which a field is exposed. Here we exposed the field Id of Character as the graphQL scalar ID.

## Custom Scalar Registration

```Go
typ := reflect.TypeOf(time.Time{})
schemabuilder.RegisterScalar(typ, "DateTime", func(value interface{}, dest reflect.Value) error {
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
```

## Interface Registration

```Go
type server struct {
    dragons []Dragon
    snakes  []Snake
}

type Dragon struct {
    Id           string
    Name         string
    NumberOfLegs int32
}

type Snake struct {
    Id             string
    Name           string
    LengthInMetres float32
}

type MagicalCreature struct {
    schemabuilder.Interface
    *Dragon
    *Snake
}

func (s *server) RegisterInterface(schema *schemabuilder.Schema) {
    schema.Query().FieldFunc("magicalCreature", func(ctx context.Context, args struct {
        Id *schemabuilder.ID
    }) *MagicalCreature {
        for _, d := range s.dragons {
            if d.Id == args.Id.Value {
                return &MagicalCreature{
                    Dragon: &d,
                }
            }
        }

        for _, sn := range s.snakes {
            if sn.Id == args.Id.Value {
                return &MagicalCreature{
                    Snake: &sn,
                }
            }
        }

        return nil
    })
}

func RegisterPayloads(schema *schemabuilder.Schema) {
	payload := schema.Object("Dragon", Dragon{})
	payload.FieldFunc("id", func(ctx context.Context, in *Dragon) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	payload.FieldFunc("name", func(ctx context.Context, in *Dragon) string {
		return in.Name
	})
	payload.FieldFunc("numberOfLegs", func(ctx context.Context, in *Dragon) int32 {
		return in.NumberOfLegs
	})

	payload = schema.Object("Snake", Snake{})
	payload.FieldFunc("id", func(ctx context.Context, in *Snake) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	payload.FieldFunc("name", func(ctx context.Context, in *Snake) string {
		return in.Name
	})
	payload.FieldFunc("lengthInMetres", func(ctx context.Context, in *Snake) float32 {
		return in.LengthInMetres
	})
}

func main() {
	s := server{
		dragons: []Dragon{
			{
				Id:           "01d823a8-fdcd-4d03-957c-7ca898e2e5fd",
				Name:         "Norbert",
				NumberOfLegs: 2,
			},
		},
		snakes: []Snake{
			{
				Id:             "2631a997-7a73-45b2-a2fc-ae665a383da1",
				Name:           "Nagini",
				LengthInMetres: 1.23,
			},
		},
	}

	sb := schemabuilder.NewSchema()
	RegisterPayloads(sb)
	s.RegisterInterface(sb)

	schema := sb.MustBuild()
	introspection.AddIntrospectionToSchema(schema)

	http.Handle("/graphql", jaal.HTTPHandler(schema))
	fmt.Println("Running")
	if err := http.ListenAndServe(":9000", nil); err != nil {
		panic(err)
	}
}
```

The object schemabuilder.Interface acts as a special marker. It indicates that the type is to be registered as an interface. Jaal automatically registers the common fields(Id, Name) of the objects(Dragon & Snake) as the fields of interface (MagicalCreature). While defining a struct for interface, one must remember that all the fields of that struct are anonymous.

## Union Registration

The above example can be converted to a union by replacing schemabuilder.Interface with schemabuilder.Union and RegisterInterface() by RegisterUnion().

```Go
type MagicalCreature struct {
    schemabuilder.Union
    *Dragon
    *Snake
}

func (s *server) RegisterUnion(schema *schemabuilder.Schema) {
    schema.Query().FieldFunc("magicalCreature", func(ctx context.Context, args struct {
        Id *schemabuilder.ID
    }) *MagicalCreature {
        for _, d := range s.dragons {
            if d.Id == args.Id.Value {
                return &MagicalCreature{
                    Dragon: &d,
                }
            }
        }

        for _, sn := range s.snakes {
            if sn.Id == args.Id.Value {
                return &MagicalCreature{
                    Snake: &sn,
                }
            }
        }

        return nil
    })
}
```

## protoc-gen-jaal - Develop relay compliant GraphQL servers

[protoc-gen-jaal](https://github.com/appointy/protoc-gen-jaal) is a protoc plugin which is used to generate jaal APIs. The server built from these APIs is graphQL spec compliant as well as relay compliant. It also handles oneOf by registering it as a Union on the schema.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for reporting bugs and issues, the process for submitting pull requests to us, and roadmap. This project has adopted [Contributor Covenant Code of Conduct](code-of-conduct.md).

## Contributors

* Souvik Mandal (mandalsouvik76@gmail.com) - Implemented protoc-gen-jaal for creating jaal APIs.
* Bitan Paul (bitanpaul1@gmail.com) - Implemented relay compliant graphql subscriptions.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

## Acknowledgments

* **Samsara** - *Initial work* - [Thunder](https://github.com/samsarahq/thunder)
