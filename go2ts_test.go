package go2ts

import (
	"context"
	"reflect"
	"testing"
)

func expect(t *testing.T, actual string, expected string) {
	t.Helper()
	if actual != expected {
		t.Fatalf("expected \"%s\" to equal \"%s\"", expected, actual)
	}
}

func typ(i interface{}) reflect.Type {
	return reflect.TypeOf(i)
}

type User struct{ Name string }
type Nested struct{ Owner User }

func (*Nested) Method(arg string) {}

func TestPrimitives(t *testing.T) {
	c := NewConverter()
	expect(t, c.Convert(typ("")), "string")
	expect(t, c.Convert(typ(138)), "number")
	expect(t, c.Convert(typ(int64(138))), "number")
	expect(t, c.Convert(typ(true)), "boolean")
}

func TestStructs(t *testing.T) {
	c := NewConverter()
	expect(t, c.Convert(typ(User{})), "{ Name: string }")
	expect(t, c.Convert(typ(&User{})), "{ Name: string }")
	expect(t, c.Convert(typ(Nested{})), "{ Owner: { Name: string } }")
}

func TestFuncs(t *testing.T) {
	c := NewConverter()
	// params
	expect(t, c.Convert(typ(func() {})), "() => Promise<void>")
	expect(t, c.Convert(typ(func(string, int, bool) {})), "(str: string, int: number, bool: boolean) => Promise<void>")
	expect(t, c.Convert(typ(func(struct{}) {})), "(_: {}) => Promise<void>")
	expect(t, c.Convert(typ(func(struct{ ID string }) {})), "(_: { ID: string }) => Promise<void>")
	expect(t, c.Convert(typ(func(User) {})), "(user: { Name: string }) => Promise<void>")
	expect(t, c.Convert(typ(func(*User) {})), "(user: { Name: string }) => Promise<void>")
	expect(t, c.Convert(typ(func(Nested) {})), "(nested: { Owner: { Name: string } }) => Promise<void>")
	// returns
	expect(t, c.Convert(typ(func() string { return "foo" })), "() => Promise<string>")
	expect(t, c.Convert(typ(func() User { return User{} })), "() => Promise<{ Name: string }>")
	expect(t, c.Convert(typ(func() *User { return nil })), "() => Promise<{ Name: string }>")
	expect(t, c.Convert(typ(func() error { return nil })), "() => Promise<void>")
	expect(t, c.Convert(typ(func() (string, error) { return "", nil })), "() => Promise<string>")
	expect(t, c.Convert(typ(func() (string, string, error) { return "", "", nil })), "() => Promise<[string, string]>")
	expect(t, c.Convert(typ(func() chan string { return nil })), "() => Promise<AsyncIterable<string>>")
	// ignore context
	expect(t, c.Convert(typ(func(context.Context, string) {})), "(str: string) => Promise<void>")
	// methods
	n := Nested{}
	m, _ := typ(&n).MethodByName("Method")
	c.ConfigureFunc = func(t reflect.Type) FuncConf { return FuncConf{IsMethod: true, MethodName: m.Name} }
	expect(t, c.Convert(m.Type), "Method (str: string): Promise<void>")
	// configurations
	c.ConfigureFunc = func(t reflect.Type) FuncConf { return FuncConf{ParamNames: []string{"foo", "bar", "baz"}} }
	expect(t, c.Convert(typ(func(string, int, bool) {})), "(foo: string, bar: number, baz: boolean) => Promise<void>")
	c.ConfigureFunc = func(t reflect.Type) FuncConf { return FuncConf{IsSync: true} }
	expect(t, c.Convert(typ(func() string { return "" })), "() => string")
	c.ConfigureFunc = func(t reflect.Type) FuncConf { return FuncConf{IsSync: true, AlwaysArray: true} }
	expect(t, c.Convert(typ(func() string { return "" })), "() => [string]")
	c.ConfigureFunc = func(t reflect.Type) FuncConf { return FuncConf{IsSync: true, NoIgnoreContext: true} }
	expect(t, c.Convert(typ(func(ctx context.Context) {})), "(context: any) => void")
}

func TestSlices(t *testing.T) {
	c := NewConverter()
	expect(t, c.Convert(typ([]string{})), "Array<string>")
	expect(t, c.Convert(typ([]*User{})), "Array<{ Name: string }>")
}

func TestArrays(t *testing.T) {
	c := NewConverter()
	expect(t, c.Convert(typ([2]string{"first", "second"})), "Array<string>")
}

func TestMaps(t *testing.T) {
	c := NewConverter()
	expect(t, c.Convert(typ(map[string]int{})), "{ [k: string]: number }")
	expect(t, c.Convert(typ(map[string]User{})), "{ [k: string]: { Name: string } }")
}
