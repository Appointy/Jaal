package graphql

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
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
			if fragment.On != typString {
				continue
			}
			resolved, err := e.executeObject(ctx, graphqlTyp, inner.Interface(), fragment.SelectionSet)
			if err != nil {
				return nil, fmt.Errorf("%v - %v", typString, err)
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

	selections := Flatten(selectionSet)

	fields := make(map[string]interface{})

	// for every selection, resolve the value and store it in the output object
	for _, selection := range selections {
		if selection.Name == "__typename" {
			fields[selection.Alias] = typ.Name
			continue
		}

		field := typ.Fields[selection.Name]
		resolved, err := e.resolveAndExecute(ctx, field, source, selection)
		if err != nil {
			return nil, fmt.Errorf("%v - %v", selection.Alias, err)
		}
		fields[selection.Alias] = resolved
	}

	if typ.KeyField != nil {
		value, err := e.resolveAndExecute(ctx, typ.KeyField, source, &Selection{})
		if err != nil {
			return nil, fmt.Errorf("%v - %v", "__key", err)
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
			return nil, fmt.Errorf("%v - %v", fmt.Sprint(i), err)
		}
		items[i] = resolved
	}

	return items, nil
}
