package router

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

func ReflectToStruct(v reflect.Value) (reflect.Value, reflect.Type) {
	if v.Kind() == reflect.Struct {
		return v, v.Type()
	} else if nd := (nextDown{}); nd.canNext(v) {
		return ReflectToStruct(nd.next(v))
	}
	panic("not a struct or a pointer to one")
}

func MustBeExported(sf reflect.StructField) {
	if !sf.IsExported() {
		panic(fmt.Sprintf("%s is not exported", sf.Name))
	}
}

type TagType interface {
	string | []string
}

func GetTag[T TagType](tag reflect.StructTag, key string, allowEmpty, required bool) T {
	v := tag.Get(key)
	if required {
		v = GetRequiredTag(tag, key)
	}
	return parseTag[T](v, checkEmptyTag(key, allowEmpty))
}

func parseTag[T TagType](v string, checkEmptyTag func(string) string) T {
	switch t := any(*new(T)).(type) {
	case string:
		return any(checkEmptyTag(v)).(T)
	case []string:
		return any(splitTag(checkEmptyTag(v), checkEmptyTag)).(T)
	default:
		panic(fmt.Sprintf("cannot convert tag to type %T", t))
	}
}

func splitTag(tag string, checkEmptyTag func(string) string) (a []string) {
	for _, e := range strings.Split(tag, ",") {
		a = append(a, checkEmptyTag(e))
	}
	return
}

func GetRequiredTag(tag reflect.StructTag, key string) string {
	if v, ok := tag.Lookup(key); ok {
		return strings.TrimSpace(v)
	}
	panic(fmt.Sprintf("must have %q tag", key))
}

func checkEmptyTag(key string, allowEmpty bool) func(string) string {
	return func(v string) string {
		v = strings.TrimSpace(v)
		if v == "" && !allowEmpty {
			panic(fmt.Sprintf("cannot have empty tag %q", key))
		}
		return v
	}
}

type (
	NextDirection interface {
		next(reflect.Value) reflect.Value
		canNext(reflect.Value) bool
	}
	nextUp   struct{}
	nextDown struct{}
)

func ReflectDown() NextDirection { return nextDown{} }
func ReflectUp() NextDirection   { return nextUp{} }

var reflectElemableKinds = []reflect.Kind{reflect.Interface, reflect.Pointer}

func (nextDown) next(sv reflect.Value) reflect.Value { return sv.Elem() }
func (nextDown) canNext(v reflect.Value) bool        { return slices.Contains(reflectElemableKinds, v.Kind()) }
func (nextUp) next(sv reflect.Value) reflect.Value   { return sv.Addr() }
func (nextUp) canNext(sv reflect.Value) bool         { return sv.CanAddr() }

func IterStruct(v reflect.Value, fn func(reflect.Value, reflect.Type, reflect.StructField)) {
	v, t := ReflectToStruct(v)
	for i := 0; i < v.NumField(); i++ {
		fn(v.Field(i).Addr(), v.Field(i).Addr().Type(), t.Field(i))
	}
}

func GetValue[T any](sv reflect.Value, retries int) (T, bool) {
	for _, e := range []NextDirection{ReflectDown(), ReflectUp()} {
		if v, ok := GetValueDirection[T](sv, retries, e); ok {
			return v, true
		}
	}
	return *new(T), false
}

func GetValueDirection[T any](sv reflect.Value, retries int, direction NextDirection) (t T, _ bool) {
	if retries <= 0 {
		return
	} else if sv.IsZero() {
		return
	} else if handler, ok := sv.Interface().(T); ok {
		return handler, true
	} else if direction.canNext(sv) {
		return GetValueDirection[T](direction.next(sv), retries-1, direction)
	}
	return
}
