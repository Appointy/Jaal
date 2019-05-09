package internal

import "reflect"

// IsScalarType checks whether a reflect.Type is scalar or not
func IsScalarType(t reflect.Type) bool {
	switch t.Kind() {
	case
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String:
		return true

	default:
		return false
	}
}

// TypesIdenticalOrScalarAliases checks whether a & b are same scalar
func TypesIdenticalOrScalarAliases(a, b reflect.Type) bool {
	return a == b || (a.Kind() == b.Kind() && IsScalarType(a))
}
