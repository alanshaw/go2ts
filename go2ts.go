// Package go2ts converts golang types to Typescript declarations.
//
// Example:
// 	package main
//
// 	import (
// 		"reflect"
// 		"github.com/alanshaw/go2ts"
// 	)
//
// 	type User struct {
// 		Name string
// 	}
//
// 	func main () {
// 		c := go2ts.NewConverter()
//
// 		c.Convert(reflect.TypeOf("")) // string
// 		c.Convert(reflect.TypeOf(User{})) // { Name: string }
// 		c.Convert(reflect.TypeOf(func(string, int, bool) User { return nil })
// 		// (str: string, int: number, bool: boolean) => Promise<{ Name: string }>
//
// 		// Add custom type declarations
// 		c.AddTypes(map[reflect.Type]string{
// 			reflect.TypeOf(User{}): "User"
// 		})
//
// 		// Output now includes "User" instead of { Name: string }
// 		c.Convert(reflect.TypeOf(map[string]User{})) // { [k: string]: User }
// 	}
package go2ts

import (
	"fmt"
	"reflect"
	"strings"
)

var primitives = map[reflect.Type]string{
	reflect.TypeOf((*bool)(nil)).Elem():    "boolean",
	reflect.TypeOf((*int)(nil)).Elem():     "number",
	reflect.TypeOf((*int8)(nil)).Elem():    "number",
	reflect.TypeOf((*int16)(nil)).Elem():   "number",
	reflect.TypeOf((*int32)(nil)).Elem():   "number",
	reflect.TypeOf((*int64)(nil)).Elem():   "number",
	reflect.TypeOf((*uint)(nil)).Elem():    "number",
	reflect.TypeOf((*uint8)(nil)).Elem():   "number",
	reflect.TypeOf((*uint16)(nil)).Elem():  "number",
	reflect.TypeOf((*uint32)(nil)).Elem():  "number",
	reflect.TypeOf((*uint64)(nil)).Elem():  "number",
	reflect.TypeOf((*float32)(nil)).Elem(): "number",
	reflect.TypeOf((*float64)(nil)).Elem(): "number",
	reflect.TypeOf((*uintptr)(nil)).Elem(): "number",
	reflect.TypeOf((*string)(nil)).Elem():  "string",
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// FuncConf are configuration options that determine how a function is
// converted into a typescript declaration by the converter.
type FuncConf struct {
	// IsSync flags that the func is synchronous and returns values instead of a
	// Promise that resolves to the return values.
	IsSync bool
	// AlwaysArray will cause an array to be returned, even if there is only a
	// single return value. By default an array is only returned if there are 2+
	// return values.
	AlwaysArray bool
	// NoIgnoreContext will include a context.Context param in the typescript
	// function declaration if it is the first parameter. Default is to ignore it.
	NoIgnoreContext bool
	// IsMethod flags that the func is a method with a (ignored) receiver param
	// and causes the converter to output a class method declaration.
	IsMethod bool
	// MethodName is the name of the method, used when FuncConf.IsMethod is true.
	MethodName string
	// ParamNames overrides default or global parameter names.
	ParamNames []string
}

// Converter will convert a golang reflect.Type to Typescript type string.
type Converter struct {
	types      map[reflect.Type]string
	paramNames map[reflect.Type]string
	// OnConvert is called when a type is converted but NOT present in the types
	// table. It is safe (and expected) that Converter.AddTypes is called from
	// this handler so that discovered types can be included in a converted type.
	OnConvert func(reflect.Type, string)
	// ConfigureFunc is called for each function that is converted in order to set
	// configuration options for how the typescript declaration should appear.
	ConfigureFunc func(reflect.Type) FuncConf
}

// NewConverter creates a new converter instance with primitive types added.
func NewConverter() *Converter {
	c := Converter{
		types:      make(map[reflect.Type]string),
		paramNames: make(map[reflect.Type]string),
		OnConvert:  func(reflect.Type, string) {},
	}
	c.AddTypes(primitives)
	c.AddParamNames(paramNames)
	return &c
}

// AddTypes adds custom types.
func (c *Converter) AddTypes(customTypes map[reflect.Type]string) {
	for k, v := range customTypes {
		c.types[k] = v
	}
}

// AddParamNames adds custom function parameter names for types.
func (c *Converter) AddParamNames(customParamNames map[reflect.Type]string) {
	for k, v := range customParamNames {
		c.paramNames[k] = v
	}
}

// Convert takes a golang reflect.Type and returns a Typescript type string.
//
// Notes:
//
// chan is converted to AsyncIterable.
//
// Assumes functions/methods are async so return values are all Promise<T>
// and errors assumed to be thrown not returned.
//
// If a function returns multiple values they are returned as an array.
//
// Context in function params is ignored.
//
// Recursion is NOT supported.
//
// Interfaces are converted to any.
//
// struct methods are NOT converted, but Converter.ConfigureFunc can be 
// used to create method declarations.
func (c *Converter) Convert(t reflect.Type) (ts string) {
	ts, ok := c.types[t]
	if ok {
		return
	}

	kind := t.Kind()

	// Handle type aliases
	for t, s := range primitives {
		if t.Kind() == kind {
			ts = s
			return
		}
	}

	defer func() { c.OnConvert(t, ts) }()

	if kind == reflect.Ptr {
		ts = c.convert(t.Elem())
	} else if kind == reflect.Chan {
		ts = fmt.Sprintf("AsyncIterable<%s>", c.convert(t.Elem()))
	} else if kind == reflect.Func {
		ts = c.convertFunc(t)
	} else if kind == reflect.Struct {
		ts = c.convertStruct(t)
	} else if kind == reflect.Slice {
		ts = fmt.Sprintf("Array<%s>", c.convert(t.Elem()))
	} else if kind == reflect.Map {
		ts = fmt.Sprintf("{ [k: string]: %s }", c.convert(t.Elem()))
	} else if kind == reflect.Interface {
		ts = "any"
	} else {
		panic(fmt.Errorf("unhandled type: %v (%s)", t, t.Kind()))
	}
	return
}

func (c *Converter) convert(t reflect.Type) string {
	ts := c.Convert(t)
	// re-check against types: OnConvert may have called AddTypes
	uts, ok := c.types[t]
	if ok {
		return uts
	}
	return ts
}

// extractFunc extracts type inforamtion about a function.
func (c *Converter) extractFunc(t reflect.Type, fconf FuncConf) *funcInfo {
	finfo := funcInfo{Name: t.Name(), Returns: "void"}
	if t.NumOut() > 0 {
		var rets []string
		for i := 0; i < t.NumOut(); i++ {
			out := t.Out(i)
			if i == t.NumOut()-1 && out.Implements(errorType) {
				break // skip last param if error
			}
			rets = append(rets, c.convert(out))
		}

		// If only 1 value just return it, if more than 1 we need to wrap in array.
		if (len(rets) > 0 && fconf.AlwaysArray) || len(rets) > 1 {
			finfo.Returns = fmt.Sprintf("[%s]", strings.Join(rets, ", "))
		} else if len(rets) == 1 {
			finfo.Returns = rets[0]
		}
	}

	start := 0
	if fconf.IsMethod {
		start = 1 // first argument is receiver, so skip over this
	}
	for i := start; i < t.NumIn(); i++ {
		in := t.In(i)
		// skip context if method takes one
		if in.Name() == "Context" && !fconf.NoIgnoreContext {
			continue
		}
		var name string
		if len(fconf.ParamNames) > i {
			name = fconf.ParamNames[i]
		} else {
			name = c.paramName(in)
		}
		p := param{name, c.convert(in)}
		finfo.appendParam(p)
	}

	if !fconf.IsSync {
		finfo.Returns = fmt.Sprintf("Promise<%s>", finfo.Returns)
	}
	return &finfo
}

// paramName attempts to find a name for a function parameter.
func (c *Converter) paramName(t reflect.Type) string {
	name, ok := c.paramNames[t]
	if ok {
		return name
	}
	if t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		return c.paramName(t.Elem())
	}
	name = t.Name()
	if name == "" {
		name = "_"
	} else {
		if isUpper(name) {
			name = strings.ToLower(name)
		} else {
			name = strings.ToLower(name[0:1]) + name[1:]
		}
	}
	return name
}

// ConvertFunc converts a function type to a typescript declaration.
func (c *Converter) convertFunc(t reflect.Type) string {
	var fconf FuncConf
	if c.ConfigureFunc != nil {
		fconf = c.ConfigureFunc(t)
	}
	finfo := c.extractFunc(t, fconf)
	var params []string
	for _, p := range finfo.Params {
		params = append(params, fmt.Sprintf("%s: %s", p.Name, p.Type))
	}
	if fconf.IsMethod {
		return fmt.Sprintf("%s (%s): %s", fconf.MethodName, strings.Join(params, ", "), finfo.Returns)
	}
	return fmt.Sprintf("(%s) => %s", strings.Join(params, ", "), finfo.Returns)
}

// extractStruct extracts typescript type information about a struct.
func (c *Converter) extractStruct(t reflect.Type) *structInfo {
	sinfo := structInfo{Name: t.Name()}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !isUpper(f.Name[0:1]) {
			continue
		}
		sinfo.Fields = append(sinfo.Fields, field{Name: f.Name, Type: c.convert(f.Type)})
	}
	return &sinfo
}

// convertStruct converts a struct to a typescript declaration.
func (c *Converter) convertStruct(t reflect.Type) string {
	sinfo := c.extractStruct(t)
	if len(sinfo.Fields) == 0 {
		return "{}"
	}
	var fields []string
	for _, f := range sinfo.Fields {
		fields = append(fields, fmt.Sprintf("%s: %s", f.Name, f.Type))
	}
	return fmt.Sprintf("{ %s }", strings.Join(fields, ", "))
}
