package graphql

import (
	"context"
	"fmt"
)

// ValidateQuery checks that the given selectionSet matches the schema typ, and parses the args in selectionSet
func ValidateQuery(ctx context.Context, typ Type, selectionSet *SelectionSet) error {
	switch typ := typ.(type) {
	case *Scalar:
		if selectionSet != nil {
			return fmt.Errorf("scalar field must have no selections")
		}
		return nil
	case *Enum:
		if selectionSet != nil {
			return fmt.Errorf("enum field must have no selections")
		}
		return nil
	case *Union:
		if selectionSet == nil {
			return fmt.Errorf("object field must have selections")
		}

		for _, fragment := range selectionSet.Fragments {
			for typString, graphqlTyp := range typ.Types {
				if fragment.Fragment.On != typString {
					continue
				}
				if err := ValidateQuery(ctx, graphqlTyp, fragment.Fragment.SelectionSet); err != nil {
					return err
				}
			}
		}
		for _, selection := range selectionSet.Selections {
			if selection.Name == "__typename" {
				if !isNilArgs(selection.Args) {
					return fmt.Errorf(`error parsing args for "__typename": no args expected`)
				}
				if selection.SelectionSet != nil {
					return fmt.Errorf(`scalar field "__typename" must have no selection`)
				}
				for _, fragment := range selectionSet.Fragments {
					fragment.Fragment.SelectionSet.Selections = append(fragment.Fragment.SelectionSet.Selections, selection)
				}
				continue
			}
			return fmt.Errorf(`unknown field "%s"`, selection.Name)
		}
		return nil

	case *Interface:
		if selectionSet == nil {
			return fmt.Errorf("object field must have selections")
		}
		for _, fragment := range selectionSet.Fragments {
			for typString, graphqlTyp := range typ.Types {
				if fragment.Fragment.On != typString {
					continue
				}
				if err := ValidateQuery(ctx, graphqlTyp, fragment.Fragment.SelectionSet); err != nil {
					return err
				}
			}
		}
		for _, selection := range selectionSet.Selections {
			if selection.Name == "__typename" {
				if !isNilArgs(selection.Args) {
					return fmt.Errorf(`error parsing args for "__typename": no args expected`)
				}
				if selection.SelectionSet != nil {
					return fmt.Errorf(`scalar field "__typename" must have no selection`)
				}
				continue
			}
			field, ok := typ.Fields[selection.Name]
			if !ok {
				return fmt.Errorf(`unknown field "%s"`, selection.Name)
			}

			if !selection.parsed {
				parsed, err := field.ParseArguments(selection.Args)
				if err != nil {
					return fmt.Errorf(`error parsing args for "%s": %s`, selection.Name, err)
				}
				selection.Args = parsed
				selection.parsed = true
			}
			if err := ValidateQuery(ctx, field.Type, selection.SelectionSet); err != nil {
				return err
			}
		}

		return nil
	case *Object:
		if selectionSet == nil {
			return fmt.Errorf("object field must have selections")
		}
		for _, selection := range selectionSet.Selections {
			if selection.Name == "__typename" {
				if !isNilArgs(selection.Args) {
					return fmt.Errorf(`error parsing args for "__typename": no args expected`)
				}
				if selection.SelectionSet != nil {
					return fmt.Errorf(`scalar field "__typename" must have no selection`)
				}
				continue
			}

			field, ok := typ.Fields[selection.Name]
			if !ok {
				return fmt.Errorf(`unknown field "%s"`, selection.Name)
			}

			// Only parse args once for a given selection.
			if !selection.parsed {
				parsed, err := field.ParseArguments(selection.Args)
				if err != nil {
					return fmt.Errorf(`error parsing args for "%s": %s`, selection.Name, err)
				}
				selection.Args = parsed
				selection.parsed = true
			}

			if err := ValidateQuery(ctx, field.Type, selection.SelectionSet); err != nil {
				return err
			}
		}
		for _, fragment := range selectionSet.Fragments {
			if err := ValidateQuery(ctx, typ, fragment.Fragment.SelectionSet); err != nil {
				return err
			}
		}
		return nil

	case *List:
		return ValidateQuery(ctx, typ.Type, selectionSet)

	case *NonNull:
		return ValidateQuery(ctx, typ.Type, selectionSet)

	default:
		panic("unknown type kind")
	}
}

func isNilArgs(args interface{}) bool {
	m, ok := args.(map[string]interface{})
	return args == nil || (ok && len(m) == 0)
}
