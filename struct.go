package go2ts

// structInfo is exported information about a golang func.
type structInfo struct {
	Name   string
	Fields []field
	// TODO: methods?
}

// field is a struct field.
type field struct {
	Name string
	Type string
}
