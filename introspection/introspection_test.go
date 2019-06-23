package introspection_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"go.appointy.com/jaal/graphql"

	"github.com/stretchr/testify/require"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
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

func TestIntrospectionForInterface(t *testing.T) {
	s := node{
		customers: []Customer{
			{
				Id:   "cus_01DBF6E5CE9JY03HP3XGAVRAAC",
				Name: "Anuj",
			},
		},
		providers: []Provider{
			{
				Id:    "pro_01DBF6E5CE9JY03HP3XGMTCFR7",
				Email: "anuj.g@appointy.com",
			},
		},
	}
	builder := schemabuilder.NewSchema()
	s.registerNodeInterface(builder)
	schema := builder.MustBuild()
	introspection.AddIntrospectionToSchema(schema)
	e := graphql.Executor{}

	tests := []struct {
		name           string
		query          string
		expectedResult interface{}
	}{
		{
			name: "Test __Schema",
			query: `
				{
					__schema{
						types{
							name
							kind
						}
						mutationType{
							name
							kind
							fields{
								name
							}
						}
						subscriptionType{
							name
							kind
						}
						directives{
							name
						}
					}
				}
			`,
			expectedResult: map[string]interface{}{
				"__schema": map[string]interface{}{
					"types": []interface{}{
						map[string]interface{}{
							"name": "Customer",
							"kind": "OBJECT",
						},
						map[string]interface{}{
							"name": "ID",
							"kind": "SCALAR",
						},
						map[string]interface{}{
							"kind": "OBJECT",
							"name": "Mutation",
						},
						map[string]interface{}{
							"name": "Node",
							"kind": "INTERFACE",
						},
						map[string]interface{}{
							"name": "NodeInput",
							"kind": "INPUT_OBJECT",
						},
						map[string]interface{}{
							"name": "OneOf",
							"kind": "UNION",
						},
						map[string]interface{}{
							"name": "Provider",
							"kind": "OBJECT",
						},
						map[string]interface{}{
							"name": "ProviderType",
							"kind": "ENUM",
						},
						map[string]interface{}{
							"name": "Query",
							"kind": "OBJECT",
						},
						map[string]interface{}{
							"name": "String",
							"kind": "SCALAR",
						},
						map[string]interface{}{
							"kind": "OBJECT",
							"name": "Subscription",
						},
					},
					"mutationType": map[string]interface{}{
						"name":   "Mutation",
						"kind":   "OBJECT",
						"fields": []interface{}{},
					},
					"subscriptionType": map[string]interface{}{
						"name": "Subscription",
						"kind": "OBJECT",
					},
					"directives": []interface{}{
						map[string]interface{}{
							"name": "include",
						},
						map[string]interface{}{
							"name": "skip",
						},
					},
				},
			},
		},
		{
			name: "Test Query __Type",
			query: `
				{
					__schema{
						queryType{
							name
							kind
							interfaces{
								name
							}
							possibleTypes{
								name
							}
							enumValues{
								name
							}
							inputFields{
								name
							}
							ofType{
								name
								kind
							}
							fields{
								name
								args{
									name
									type{
										name
										kind
									}
								}
								type{
									name
									kind
								}
							}
						}
					}
				}
			`,
			expectedResult: map[string]interface{}{
				"__schema": map[string]interface{}{
					"queryType": map[string]interface{}{
						"name":          "Query",
						"kind":          "OBJECT",
						"interfaces":    []interface{}{},
						"possibleTypes": []interface{}{},
						"enumValues":    []interface{}{},
						"inputFields":   []interface{}{},
						"ofType":        nil,
						"fields": []interface{}{
							map[string]interface{}{
								"args": []interface{}{
									map[string]interface{}{
										"name": "in",
										"type": map[string]interface{}{
											"kind": "INPUT_OBJECT",
											"name": "NodeInput",
										},
									},
								},
								"name": "node",
								"type": map[string]interface{}{
									"kind": "INTERFACE",
									"name": "Node",
								},
							},
							map[string]interface{}{
								"args": []interface{}{
									map[string]interface{}{
										"name": "in",
										"type": map[string]interface{}{
											"kind": "INPUT_OBJECT",
											"name": "NodeInput",
										},
									},
								},
								"name": "oneOf",
								"type": map[string]interface{}{
									"kind": "UNION",
									"name": "OneOf",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Test Interface __Type",
			query: `
				{
					__type(name:"Node"){
						name
						kind
						fields{
							name
							args{
								name
							}
							type{
								kind
								ofType{
									name
									kind
								}
							}
						}
						possibleTypes{
							name
							kind
						}
						interfaces{
							name
						}
						enumValues{
							name
						}
						inputFields{
							name
						}
						ofType{
							name
							kind
						}
					}
				}
			`,
			expectedResult: map[string]interface{}{
				"__type": map[string]interface{}{
					"fields": []interface{}{
						map[string]interface{}{
							"args": []interface{}{},
							"name": "id",
							"type": map[string]interface{}{
								"kind": "NON_NULL",
								"ofType": map[string]interface{}{
									"kind": "SCALAR",
									"name": "ID",
								},
							},
						},
					},
					"kind": "INTERFACE",
					"name": "Node",
					"possibleTypes": []interface{}{
						map[string]interface{}{
							"kind": "OBJECT",
							"name": "Customer",
						},
						map[string]interface{}{
							"kind": "OBJECT",
							"name": "Provider",
						},
					},
					"interfaces":  []interface{}{},
					"enumValues":  []interface{}{},
					"inputFields": []interface{}{},
					"ofType":      nil,
				},
			},
		},
		{
			name: "Test Object __Type",
			query: `
				{
					__type(name:"Customer"){
						name
						kind
						fields{
							name
							args{
								name
							}
							type{
								kind
								ofType{
									name
									kind
									ofType{
										name
										kind
										ofType{
											name
											kind
										}
									}
								}
							}
						}
						interfaces{
							name
							kind
						}
						possibleTypes{
							name
						}
						enumValues{
							name
						}
						inputFields{
							name
						}
						ofType{
							name
							kind
						}
					}
				}
			`,
			expectedResult: map[string]interface{}{
				"__type": map[string]interface{}{
					"fields": []interface{}{
						map[string]interface{}{
							"args": []interface{}{},
							"name": "firstName",
							"type": map[string]interface{}{
								"kind": "NON_NULL",
								"ofType": map[string]interface{}{
									"kind":   "SCALAR",
									"name":   "String",
									"ofType": nil,
								},
							},
						},
						map[string]interface{}{
							"args": []interface{}{},
							"name": "id",
							"type": map[string]interface{}{
								"kind": "NON_NULL",
								"ofType": map[string]interface{}{
									"kind":   "SCALAR",
									"name":   "ID",
									"ofType": nil,
								},
							},
						},
						map[string]interface{}{
							"args": []interface{}{},
							"name": "phoneNumbers",
							"type": map[string]interface{}{
								"kind": "NON_NULL",
								"ofType": map[string]interface{}{
									"kind": "LIST",
									"name": "",
									"ofType": map[string]interface{}{
										"kind": "NON_NULL",
										"name": "",
										"ofType": map[string]interface{}{
											"kind": "SCALAR",
											"name": "String",
										},
									},
								},
							},
						},
					},
					"interfaces": []interface{}{
						map[string]interface{}{
							"kind": "INTERFACE",
							"name": "Node",
						},
					},
					"kind":          "OBJECT",
					"name":          "Customer",
					"possibleTypes": []interface{}{},
					"enumValues":    []interface{}{},
					"inputFields":   []interface{}{},
					"ofType":        nil,
				},
			},
		},
		{
			name: "Test InputObject __Type",
			query: `
				{
					__type(name:"NodeInput"){
						name
						kind
						inputFields{
							name
							type{
								kind
								ofType{
									name
									kind
								}
							}
						}
						interfaces{
							name
						}
						possibleTypes{
							name
						}
						enumValues{
							name
						}
						fields{
							name
						}
						ofType{
							name
							kind
						}
					}
				}
			`,
			expectedResult: map[string]interface{}{
				"__type": map[string]interface{}{
					"inputFields": []interface{}{
						map[string]interface{}{
							"name": "id",
							"type": map[string]interface{}{
								"kind":   "SCALAR",
								"ofType": nil,
							},
						},
					},
					"kind":          "INPUT_OBJECT",
					"name":          "NodeInput",
					"interfaces":    []interface{}{},
					"possibleTypes": []interface{}{},
					"enumValues":    []interface{}{},
					"fields":        []interface{}{},
					"ofType":        nil,
				},
			},
		},
		{
			name: "Test Object __Type",
			query: `
				{
					__type(name:"Provider"){
						name
						kind
						fields{
							name
							args{
								name
							}
							type{
								kind
								ofType{
									name
									kind
									ofType{
										name
										kind
										ofType{
											name
											kind
										}
									}
								}
							}
						}
						interfaces{
							name
							kind
						}
						possibleTypes{
							name
						}
						enumValues{
							name
						}
						inputFields{
							name
						}
						ofType{
							name
							kind
						}
					}
				}
			`,
			expectedResult: map[string]interface{}{
				"__type": map[string]interface{}{
					"fields": []interface{}{
						map[string]interface{}{
							"args": []interface{}{},
							"name": "email",
							"type": map[string]interface{}{
								"kind": "NON_NULL",
								"ofType": map[string]interface{}{
									"kind":   "SCALAR",
									"name":   "String",
									"ofType": nil,
								},
							},
						},
						map[string]interface{}{
							"args": []interface{}{},
							"name": "id",
							"type": map[string]interface{}{
								"kind": "NON_NULL",
								"ofType": map[string]interface{}{
									"kind":   "SCALAR",
									"name":   "ID",
									"ofType": nil,
								},
							},
						},
						map[string]interface{}{
							"args": []interface{}{},
							"name": "providerType",
							"type": map[string]interface{}{
								"kind": "NON_NULL",
								"ofType": map[string]interface{}{
									"kind":   "ENUM",
									"name":   "ProviderType",
									"ofType": nil,
								},
							},
						},
					},
					"interfaces": []interface{}{
						map[string]interface{}{
							"kind": "INTERFACE",
							"name": "Node",
						},
					},
					"kind":          "OBJECT",
					"name":          "Provider",
					"possibleTypes": []interface{}{},
					"enumValues":    []interface{}{},
					"inputFields":   []interface{}{},
					"ofType":        nil,
				},
			},
		},
		{
			name: "Test ENUM __Type",
			query: `
				{
					__type(name:"ProviderType"){
						name
						kind
						fields{
							name
						}
						interfaces{
							name
							kind
						}
						possibleTypes{
							name
						}
						enumValues{
							name
						}
						inputFields{
							name
						}
						ofType{
							name
							kind
						}
					}
				}
			`,
			expectedResult: map[string]interface{}{
				"__type": map[string]interface{}{
					"fields":        []interface{}{},
					"interfaces":    []interface{}{},
					"kind":          "ENUM",
					"name":          "ProviderType",
					"possibleTypes": []interface{}{},
					"enumValues": []interface{}{
						map[string]string{
							"name": "EMPLOYEE",
						},
						map[string]string{
							"name": "VENDOR",
						},
					},
					"inputFields": []interface{}{},
					"ofType":      nil,
				},
			},
		},
		{
			name: "Test UNION __Type",
			query: `
				{
					__type(name:"OneOf"){
						name
						kind
						fields{
							name
						}
						possibleTypes{
							name
							kind
						}
						interfaces{
							name
						}
						enumValues{
							name
						}
						inputFields{
							name
						}
						ofType{
							name
							kind
						}
					}
				}
			`,
			expectedResult: map[string]interface{}{
				"__type": map[string]interface{}{
					"fields": []interface{}{},
					"kind":   "UNION",
					"name":   "OneOf",
					"possibleTypes": []interface{}{
						map[string]interface{}{
							"kind": "OBJECT",
							"name": "Customer",
						},
						map[string]interface{}{
							"kind": "OBJECT",
							"name": "Provider",
						},
					},
					"interfaces":  []interface{}{},
					"enumValues":  []interface{}{},
					"inputFields": []interface{}{},
					"ofType":      nil,
				},
			},
		},
		{
			name: "Test Directives",
			query: `
				{
					__schema{
						directives{
							name
							description
							locations
							args{
								name
								description
								type{
									name
									kind
									description
									fields{
										name
									}
									interfaces{
										name
									}
									possibleTypes{
										name
									}
									enumValues{
										name
									}
									inputFields{
										name
									}
								}
								defaultValue
							}
						}
					}
				}
			`,
			expectedResult: map[string]interface{}{
				"__schema": map[string]interface{}{
					"directives": []interface{}{
						map[string]interface{}{
							"name":        "include",
							"description": "Directs the executor to include this field or fragment only when the `if` argument is true.",
							"locations": []interface{}{
								"FIELD",
								"FRAGMENT_SPREAD",
								"INLINE_FRAGMENT",
							},
							"args": []interface{}{
								map[string]interface{}{
									"name":         "if",
									"description":  "Included when true.",
									"defaultValue": nil,
									"type": map[string]interface{}{
										"name":          "Boolean",
										"kind":          "SCALAR",
										"description":   "",
										"fields":        []interface{}{},
										"interfaces":    []interface{}{},
										"possibleTypes": []interface{}{},
										"enumValues":    []interface{}{},
										"inputFields":   []interface{}{},
									},
								},
							},
						},
						map[string]interface{}{
							"name":        "skip",
							"description": "Directs the executor to skip this field or fragment only when the `if` argument is true.",
							"locations": []interface{}{
								"FIELD",
								"FRAGMENT_SPREAD",
								"INLINE_FRAGMENT",
							},
							"args": []interface{}{
								map[string]interface{}{
									"name":         "if",
									"description":  "Skipped when true.",
									"defaultValue": nil,
									"type": map[string]interface{}{
										"name":          "Boolean",
										"kind":          "SCALAR",
										"description":   "",
										"fields":        []interface{}{},
										"interfaces":    []interface{}{},
										"possibleTypes": []interface{}{},
										"enumValues":    []interface{}{},
										"inputFields":   []interface{}{},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := graphql.Parse(tt.query, nil)
			if err != nil {
				t.Fatal(err)
			}

			if err := graphql.ValidateQuery(context.Background(), schema.Query, query.SelectionSet); err != nil {
				t.Fatal(err)
			}

			result, err := e.Execute(context.Background(), schema.Query, nil, query)
			if err != nil {
				t.Fatal(err)
			}

			a, _ := json.Marshal(result)
			b, _ := json.Marshal(tt.expectedResult)

			if !reflect.DeepEqual(a, b) {
				t.Fatalf("%v Failed\ngot : \n%v\n\n\nactual : \n%v", tt.name, string(a), string(b))
			}

		})
	}
}

type Customer struct {
	Id           string
	Name         string
	PhoneNumbers []string
}

type Provider struct {
	Id    string
	Email string
	Type  ProviderType
}

type ProviderType int32

const (
	ProviderType_VENDOR   ProviderType = 0
	ProviderType_EMPLOYEE ProviderType = 1
)

type NodeInput struct {
	Id string
}

type Node struct {
	schemabuilder.Interface
	*Customer
	*Provider
}

type OneOf struct {
	schemabuilder.Union
	*Customer
	*Provider
}

type node struct {
	customers []Customer
	providers []Provider
}

func (s *node) registerNodeInterface(schema *schemabuilder.Schema) {
	schema.Query().FieldFunc("node", func(ctx context.Context, args struct {
		In NodeInput
	}) *Node {

		if strings.Contains(args.In.Id, "cus") {
			for _, cus := range s.customers {
				if cus.Id == args.In.Id {
					return &Node{
						Customer: &cus,
					}
				}
			}
		}

		for _, pro := range s.providers {
			if pro.Id == args.In.Id {
				return &Node{
					Provider: &pro,
				}
			}
		}

		return &Node{
			Customer: &Customer{Id: args.In.Id},
			Provider: &Provider{Id: args.In.Id},
		}
	})

	schema.Query().FieldFunc("oneOf", func(ctx context.Context, args struct {
		In NodeInput
	}) *OneOf {

		for _, cus := range s.customers {
			if cus.Id == args.In.Id {
				return &OneOf{
					Customer: &cus,
				}
			}
		}

		for _, pro := range s.providers {
			if pro.Id == args.In.Id {
				return &OneOf{
					Provider: &pro,
				}
			}
		}

		return nil
	})

	s.registerCustomer(schema)
	s.registerProvider(schema)
	s.registerNodeInput(schema)
	s.registerEnumType(schema)
	schema.Mutation()
}

func (s *node) registerNodeInput(schema *schemabuilder.Schema) {
	input := schema.InputObject("NodeInput", NodeInput{})
	input.FieldFunc("id", func(target *NodeInput, source *schemabuilder.ID) {
		target.Id = source.Value
	})
}

func (s *node) registerCustomer(schema *schemabuilder.Schema) {
	obj := schema.Object("Customer", Customer{})
	obj.FieldFunc("id", func(in *Customer) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("firstName", func(in *Customer) string {
		return in.Name
	})
	obj.FieldFunc("phoneNumbers", func(in *Customer) []string {
		return in.PhoneNumbers
	})
}

func (s *node) registerProvider(schema *schemabuilder.Schema) {
	obj := schema.Object("Provider", Provider{})
	obj.FieldFunc("id", func(in *Provider) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	obj.FieldFunc("email", func(in *Provider) string {
		return in.Email
	})
	obj.FieldFunc("providerType", func(in *Provider) ProviderType {
		return in.Type
	})
}

func (s *node) registerEnumType(schema *schemabuilder.Schema) {
	schema.Enum(ProviderType(0), map[string]interface{}{
		"VENDOR":   ProviderType(0),
		"EMPLOYEE": ProviderType(1),
	})
}
