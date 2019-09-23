package graphql

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"go.appointy.com/jaal/internal"
)

type ComputationInput struct {
	Id                   string
	Query                string
	ParsedQuery          *Query
	Variables            map[string]interface{}
	Ctx                  context.Context
	Previous             interface{}
	IsInitialComputation bool
	Extensions           map[string]interface{}
}

type Executor struct {
}

var ErrNoUpdate = errors.New("no update")

func (e *Executor) Execute(ctx context.Context, typ Type, source interface{}, query *Query) (interface{}, error) {
	return e.execute(ctx, typ, source, query.SelectionSet)
}

func (e *Executor) execute(ctx context.Context, typ Type, source interface{}, selectionSet *SelectionSet) (interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch typ := typ.(type) {
	case *Scalar:
		if typ.Unwrapper != nil {
			return typ.Unwrapper(source)
		}
		return unwrap(source), nil
	case *Enum:
		val := unwrap(source)
		if mapVal, ok := typ.ReverseMap[val]; ok {
			return mapVal, nil
		}
		return nil, errors.New("enum is not valid")
	case *Union:
		return e.executeUnion(ctx, typ, source, selectionSet)
	case *Interface:
		return e.executeInterface(ctx, typ, source, selectionSet)
	case *Object:
		return e.executeObject(ctx, typ, source, selectionSet)
	case *List:
		return e.executeList(ctx, typ, source, selectionSet)
	case *NonNull:
		return e.execute(ctx, typ.Type, source, selectionSet)
	default:
		panic(typ)
	}
}

// unwrap will return the value associated with a pointer type, or nil if the pointer is nil
func unwrap(v interface{}) interface{} {
	i := reflect.ValueOf(v)
	for i.Kind() == reflect.Ptr && !i.IsNil() {
		i = i.Elem()
	}
	if i.Kind() == reflect.Invalid {
		return nil
	}
	return i.Interface()
}

func (e *Executor) executeUnion(ctx context.Context, typ *Union, source interface{}, selectionSet *SelectionSet) (interface{}, error) {
	value := reflect.ValueOf(source)
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return nil, nil
	}

	fields := make(map[string]interface{})
	for _, selection := range selectionSet.Selections {
		if selection.Name == "__typename" {
			fields[selection.Alias] = typ.Name
			continue
		}
	}

	// For every inline fragment spread, check if the current concrete type matches and execute that object.
	var possibleTypes []string
	for typString, graphqlTyp := range typ.Types {
		inner := reflect.ValueOf(source)
		if inner.Kind() == reflect.Ptr && inner.Elem().Kind() == reflect.Struct {
			inner = inner.Elem()
		}

		inner = inner.FieldByName(typString)
		if inner.IsNil() {
			continue
		}
		possibleTypes = append(possibleTypes, graphqlTyp.String())

		for _, fragment := range selectionSet.Fragments {
			if fragment.Fragment.On != typString {
				continue
			}
			resolved, err := e.executeObject(ctx, graphqlTyp, inner.Interface(), fragment.Fragment.SelectionSet)
			if err != nil {
				if err == ErrNoUpdate {
					return nil, err
				}
				return nil, internal.NestErrorPaths(err, typString)
			}

			for k, v := range resolved.(map[string]interface{}) {
				fields[k] = v
			}
		}
	}

	if len(possibleTypes) > 1 {
		return nil, fmt.Errorf("union type field should only return one value, but received: %s", strings.Join(possibleTypes, " "))
	}
	return fields, nil
}

// executeObject executes an object query
func (e *Executor) executeObject(ctx context.Context, typ *Object, source interface{}, selectionSet *SelectionSet) (interface{}, error) {
	value := reflect.ValueOf(source)
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return nil, nil
	}

	selections, err := Flatten(selectionSet)
	if err != nil {
		return nil, err
	}

	fields := make(map[string]interface{})

	// for every selection, resolve the value and store it in the output object
	for _, selection := range selections {
		if ok, err := shouldIncludeNode(selection.Directives); err != nil {
			if err == ErrNoUpdate {
				return nil, err
			}
			return nil, internal.NestErrorPaths(err, selection.Alias)
		} else if !ok {
			continue
		}

		if selection.Name == "__typename" {
			fields[selection.Alias] = typ.Name
			continue
		}

		field := typ.Fields[selection.Name]
		resolved, err := e.resolveAndExecute(ctx, field, source, selection)
		if err != nil {
			if err == ErrNoUpdate {
				return nil, err
			}
			return nil, internal.NestErrorPaths(err, selection.Alias)
		}
		fields[selection.Alias] = resolved
	}

	if typ.KeyField != nil {
		value, err := e.resolveAndExecute(ctx, typ.KeyField, source, &Selection{})
		if err != nil {
			return nil, internal.NestErrorPaths(err, "__key")
		}
		fields["__key"] = value
	}

	return fields, nil
}

func (e *Executor) resolveAndExecute(ctx context.Context, field *Field, source interface{}, selection *Selection) (interface{}, error) {
	value, err := safeExecuteResolver(ctx, field, source, selection.Args, selection.SelectionSet)
	if err != nil {
		return nil, err
	}
	return e.execute(ctx, field.Type, value, selection.SelectionSet)
}

func safeExecuteResolver(ctx context.Context, field *Field, source, args interface{}, selectionSet *SelectionSet) (result interface{}, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			result, err = nil, fmt.Errorf("graphql: panic: %v\n%s", panicErr, buf)
		}
	}()
	return field.Resolve(ctx, source, args, selectionSet)
}

var emptyList = []interface{}{}

// executeList executes a set query
func (e *Executor) executeList(ctx context.Context, typ *List, source interface{}, selectionSet *SelectionSet) (interface{}, error) {
	if reflect.ValueOf(source).IsNil() {
		return emptyList, nil
	}

	// iterate over arbitrary slice types using reflect
	slice := reflect.ValueOf(source)
	items := make([]interface{}, slice.Len())

	// resolve every element in the slice
	for i := 0; i < slice.Len(); i++ {
		value := slice.Index(i)
		resolved, err := e.execute(ctx, typ.Type, value.Interface(), selectionSet)
		if err != nil {
			if err == ErrNoUpdate {
				return nil, err
			}
			return nil, internal.NestErrorPaths(err, fmt.Sprint(i))
		}
		items[i] = resolved
	}

	return items, nil
}

// executeInterface resolves an interface query
func (e *Executor) executeInterface(ctx context.Context, typ *Interface, source interface{}, selectionSet *SelectionSet) (interface{}, error) {
	value := reflect.ValueOf(source)
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return nil, nil
	}
	fields := make(map[string]interface{})
	var possibleTypes []string
	for typString, graphqlTyp := range typ.Types {
		inner := reflect.ValueOf(source)
		if inner.Kind() == reflect.Ptr && inner.Elem().Kind() == reflect.Struct {
			inner = inner.Elem()
		}
		inner = inner.FieldByName(typString)
		if inner.IsNil() {
			continue
		}
		possibleTypes = append(possibleTypes, graphqlTyp.String())
		selections, err := Flatten(selectionSet)
		if err != nil {
			return nil, err
		}
		// for every selection, resolve the value and store it in the output object
		for _, selection := range selections {
			if selection.Name == "__typename" {
				fields[selection.Alias] = graphqlTyp.Name
				continue
			}
			field, ok := graphqlTyp.Fields[selection.Name]
			if !ok {
				continue
			}
			value := reflect.ValueOf(source).Elem()
			value = value.FieldByName(typString)
			resolved, err := e.resolveAndExecute(ctx, field, value.Interface(), selection)
			if err != nil {
				if err == ErrNoUpdate {
					return nil, err
				}
				return nil, internal.NestErrorPaths(err, selection.Alias)
			}
			fields[selection.Alias] = resolved
		}
	}
	// if len(possibleTypes) > 1 {
	// 	return nil, fmt.Errorf("interface type field should only return one value, but received: %s", strings.Join(possibleTypes, " "))
	// }
	return fields, nil
}

func findDirectiveWithName(directives []*Directive, name string) *Directive {
	for _, directive := range directives {
		if directive.Name == name {
			return directive
		}
	}
	return nil
}

func shouldIncludeNode(directives []*Directive) (bool, error) {
	parseIf := func(d *Directive) (bool, error) {
		args := d.Args.(map[string]interface{})
		if args["if"] == nil {
			return false, fmt.Errorf("required argument not provided: if")
		}

		if _, ok := args["if"].(bool); !ok {
			return false, fmt.Errorf("expected type Boolean, found %v", args["if"])
		}

		return args["if"].(bool), nil
	}

	skipDirective := findDirectiveWithName(directives, "skip")
	if skipDirective != nil {
		b, err := parseIf(skipDirective)
		if err != nil {
			return false, err
		}
		if b {
			return false, nil
		}
	}

	includeDirective := findDirectiveWithName(directives, "include")
	if includeDirective != nil {
		return parseIf(includeDirective)
	}

	return true, nil
}
