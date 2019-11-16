package graphql_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/jerrors"
	"go.appointy.com/jaal/schemabuilder"
)

func TestLazyExecution(t *testing.T) {
	s := getServer()
	sb := schemabuilder.NewSchema()

	registerBroomstick(sb)
	registerWand(sb)
	registerWizard(sb)
	s.registerQueries(sb)
	if err := s.registerConnections(sb); err != nil {
		t.Fatal(err)
	}

	builtSchema := sb.MustBuild()
	execute := func(queryString string, vars map[string]interface{}) (interface{}, error) {
		q, err := graphql.Parse(queryString, vars)
		if err != nil {
			panic(err)
		}

		if err := graphql.ValidateQuery(context.Background(), builtSchema.Query, q.SelectionSet); err != nil {
			return nil, err
		}

		e := graphql.Executor{}
		return e.Execute(context.Background(), builtSchema.Query, nil, q)
	}

	t.Run("Lazy execution of wand", func(t *testing.T) {
		const query = `{
							wizard(id: "w1") {
								name
								wand {
									core
									wood
								}
							}
						}`
		expected := map[string]interface{}{
			"wizard": map[string]interface{}{
				"name": "Harry Potter",
				"wand": map[string]interface{}{
					"core": "phoenix",
					"wood": "holly",
				},
			},
		}

		result, err := execute(query, nil)
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, expected, result)
	})
	t.Run("Lazy execution of wand and broomstick", func(t *testing.T) {
		const query = `{
							wizard(id: "w1") {
								name
								wand {
									core
									wood
								}
								broomstick {
									name
								}
							}
						}`
		expected := map[string]interface{}{
			"wizard": map[string]interface{}{
				"name": "Harry Potter",
				"wand": map[string]interface{}{
					"core": "phoenix",
					"wood": "holly",
				},
				"broomstick": map[string]interface{}{
					"name": "Firebolt",
				},
			},
		}

		result, err := execute(query, nil)
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, expected, result)
	})
	t.Run("Multiple iterations of lazy execution", func(t *testing.T) {
		const query = `{
							wizard(id: "w3") {
								name
								wand {
									core
									wood
								}
								spouse {
									name
									wand {
										core
										wood
									}
								}
							}
						}`
		expected := map[string]interface{}{
			"wizard": map[string]interface{}{
				"name": "Hermoine Granger",
				"wand": map[string]interface{}{
					"core": "dragon",
					"wood": "oak",
				},
				"spouse": map[string]interface{}{
					"name": "Ronald Weasley",
					"wand": map[string]interface{}{
						"core": "unicorn",
						"wood": "teak",
					},
				},
			},
		}

		result, err := execute(query, nil)
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, expected, result)
	})
	t.Run("List lazy execution", func(t *testing.T) {
		const query = `{
							wizards {
								name
								wand {
									core
									wood
								}
								broomstick {
									name
								}
								spouse {
									name
									wand {
										core
										wood
									}
								}
							}
						}`
		expected := map[string]interface{}{
			"wizards": []interface{}{
				map[string]interface{}{
					"name": "Harry Potter",
					"wand": map[string]interface{}{
						"core": "phoenix",
						"wood": "holly",
					},
					"broomstick": map[string]interface{}{
						"name": "Firebolt",
					},
					"spouse": nil,
				},
				map[string]interface{}{
					"name": "Draco Malfoy",
					"wand": map[string]interface{}{
						"core": "serpent",
						"wood": "willow",
					},
					"broomstick": map[string]interface{}{
						"name": "Nimbus 2001",
					},
					"spouse": nil,
				},
				map[string]interface{}{
					"name": "Hermoine Granger",
					"wand": map[string]interface{}{
						"core": "dragon",
						"wood": "oak",
					},
					"broomstick": nil,
					"spouse": map[string]interface{}{
						"name": "Ronald Weasley",
						"wand": map[string]interface{}{
							"core": "unicorn",
							"wood": "teak",
						},
					},
				},
				map[string]interface{}{
					"name": "Ronald Weasley",
					"wand": map[string]interface{}{
						"core": "unicorn",
						"wood": "teak",
					},
					"broomstick": map[string]interface{}{
						"name": "Cleansweep 7",
					},
					"spouse": map[string]interface{}{
						"name": "Hermoine Granger",
						"wand": map[string]interface{}{
							"core": "dragon",
							"wood": "oak",
						},
					},
				},
			},
		}

		result, err := execute(query, nil)
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, expected, result)
	})
	t.Run("Only lazy execution at each level", func(t *testing.T) {
		const query = `{
							lazyWizard(id: "w1") {
								name
								wand {
									core
									wood
								}
							}
						}`
		expected := map[string]interface{}{
			"lazyWizard": map[string]interface{}{
				"name": "Harry Potter",
				"wand": map[string]interface{}{
					"core": "phoenix",
					"wood": "holly",
				},
			},
		}

		result, err := execute(query, nil)
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, expected, result)
	})
	t.Run("Error in lazy execution", func(t *testing.T) {
		const query = `{
							lazyWizard(id: "w11") {
								name
								wand {
									core
									wood
								}
							}
						}`

		_, err := execute(query, nil)
		if err != nil {
			expected := &jerrors.Error{
				Message: "wizard not found",
				Paths:   []string{"lazyWizard"},
				Extensions: &jerrors.Extension{
					Code: "Unknown",
				},
			}
			assert.Equal(t, expected, err)
		}
	})
}

type wand struct {
	Id   string
	Core string
	Wood string
}

type broomstick struct {
	Id   string
	Name string
}

type wizard struct {
	Id           string
	Name         string
	WandId       string
	BroomstickId string
	SpouseId     string
}

type server struct {
	wands       []*wand
	broomsticks []*broomstick
	wizards     []*wizard
}

func registerWand(schema *schemabuilder.Schema) {
	payload := schema.Object("Wand", wand{})
	payload.FieldFunc("id", func(ctx context.Context, in *wand) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	payload.FieldFunc("core", func(ctx context.Context, in *wand) string {
		return in.Core
	})
	payload.FieldFunc("wood", func(ctx context.Context, in *wand) string {
		return in.Wood
	})
}

func registerBroomstick(schema *schemabuilder.Schema) {
	payload := schema.Object("Broomstick", broomstick{})
	payload.FieldFunc("id", func(ctx context.Context, in *broomstick) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	payload.FieldFunc("name", func(ctx context.Context, in *broomstick) string {
		return in.Name
	})
}

func registerWizard(schema *schemabuilder.Schema) {
	payload := schema.Object("Wizard", wizard{})
	payload.FieldFunc("id", func(ctx context.Context, in *wizard) schemabuilder.ID {
		return schemabuilder.ID{Value: in.Id}
	})
	payload.FieldFunc("name", func(ctx context.Context, in *wizard) string {
		return in.Name
	})
	payload.FieldFunc("wandId", func(ctx context.Context, in *wizard) string {
		return in.WandId
	})
	payload.FieldFunc("broomstickId", func(ctx context.Context, in *wizard) string {
		return in.BroomstickId
	})
	payload.FieldFunc("spouseId", func(ctx context.Context, in *wizard) string {
		return in.SpouseId
	})
}

func (s *server) registerQueries(sb *schemabuilder.Schema) {
	sb.Query().FieldFunc("wizard", func(ctx context.Context, args struct{ Id schemabuilder.ID }) (*wizard, error) {
		for _, w := range s.wizards {
			if w.Id == args.Id.Value {
				return w, nil
			}
		}

		return nil, errors.New("wizard not found")
	})
	sb.Query().FieldFunc("wizards", func(ctx context.Context, args struct{}) ([]*wizard, error) {
		return s.wizards, nil
	})
	sb.Query().FieldFunc("lazyWizard", func(ctx context.Context, args struct{ Id schemabuilder.ID }) func() (*wizard, error) {
		response := make(chan *wizard)
		errCh := make(chan error)

		go func() {
			for _, w := range s.wizards {
				if w.Id == args.Id.Value {
					response <- w
					return
				}
			}

			errCh <- errors.New("wizard not found")
		}()

		return func() (*wizard, error) {
			select {
			case value := <-response:
				return value, nil
			case value := <-errCh:
				return nil, value
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	})
}

func (s *server) registerConnections(sb *schemabuilder.Schema) error {
	object, err := sb.GetObject("Wizard", wizard{})
	if err != nil {
		return err
	}
	object.FieldFunc("wand", func(ctx context.Context, in *wizard) func() (*wand, error) {
		response := make(chan *wand)
		errCh := make(chan error)

		go func() {
			if in.WandId == "" {
				response <- nil
				return
			}

			for _, w := range s.wands {
				if w.Id == in.WandId {
					response <- w
					return
				}
			}

			errCh <- errors.New("wand not found")
		}()

		return func() (*wand, error) {
			select {
			case value := <-response:
				return value, nil
			case value := <-errCh:
				return nil, value
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	})
	object.FieldFunc("broomstick", func(ctx context.Context, in *wizard) func() (*broomstick, error) {
		response := make(chan *broomstick)
		errCh := make(chan error)

		go func() {
			if in.BroomstickId == "" {
				response <- nil
				return
			}

			for _, w := range s.broomsticks {
				if w.Id == in.BroomstickId {
					response <- w
					return
				}
			}

			errCh <- errors.New("broomstick not found")
		}()

		return func() (*broomstick, error) {
			select {
			case value := <-response:
				return value, nil
			case value := <-errCh:
				return nil, value
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	})
	object.FieldFunc("spouse", func(ctx context.Context, in *wizard) func() (*wizard, error) {
		response := make(chan *wizard)
		errCh := make(chan error)

		go func() {
			if in.SpouseId == "" {
				response <- nil
				return
			}

			for _, w := range s.wizards {
				if w.Id == in.SpouseId {
					response <- w
					return
				}
			}

			errCh <- errors.New("spouse not found")
		}()

		return func() (*wizard, error) {
			select {
			case value := <-response:
				return value, nil
			case value := <-errCh:
				return nil, value
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	})

	return nil
}

func getServer() *server {
	return &server{
		wands: []*wand{
			{
				Id:   "wand1",
				Core: "dragon",
				Wood: "oak",
			},
			{
				Id:   "wand2",
				Core: "phoenix",
				Wood: "holly",
			},
			{
				Id:   "wand3",
				Core: "serpent",
				Wood: "willow",
			},
			{
				Id:   "wand4",
				Core: "unicorn",
				Wood: "teak",
			},
		},
		broomsticks: []*broomstick{
			{
				Id:   "bs1",
				Name: "Nimbus 2000",
			},
			{
				Id:   "bs2",
				Name: "Nimbus 2001",
			},
			{
				Id:   "bs3",
				Name: "Firebolt",
			},
			{
				Id:   "bs4",
				Name: "Cleansweep 7",
			},
		},
		wizards: []*wizard{
			{
				Id:           "w1",
				Name:         "Harry Potter",
				WandId:       "wand2",
				BroomstickId: "bs3",
			},
			{
				Id:           "w2",
				Name:         "Draco Malfoy",
				WandId:       "wand3",
				BroomstickId: "bs2",
			},
			{
				Id:           "w3",
				Name:         "Hermoine Granger",
				WandId:       "wand1",
				BroomstickId: "",
				SpouseId:     "w4",
			},
			{
				Id:           "w4",
				Name:         "Ronald Weasley",
				WandId:       "wand4",
				BroomstickId: "bs4",
				SpouseId:     "w3",
			},
		},
	}
}
