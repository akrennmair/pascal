package parser

import (
	"fmt"
	"strings"
)

type pointerType struct {
	name string
}

func (t *pointerType) Type() string {
	return fmt.Sprintf("^%s", t.name)
}

func (t *pointerType) Equals(dt dataType) bool {
	o, ok := dt.(*pointerType)
	return ok && (t.name == "" || o.name == "" || t.name == o.name)
}

type subrangeType struct {
	lowerBound int
	upperBound int
}

func (t *subrangeType) Type() string {
	return fmt.Sprintf("%d..%d", t.lowerBound, t.upperBound)
}

func (t *subrangeType) Equals(dt dataType) bool {
	o, ok := dt.(*subrangeType)
	return ok && t.lowerBound == o.lowerBound && t.upperBound == o.upperBound
}

type enumType struct {
	identifiers []string
}

func (t *enumType) Type() string {
	return fmt.Sprintf("(%s)", strings.Join(t.identifiers, ", "))
}

func (t *enumType) Equals(dt dataType) bool {
	o, ok := dt.(*enumType)
	if !ok {
		return false
	}
	if len(t.identifiers) != len(o.identifiers) {
		return false
	}
	for idx := range t.identifiers {
		if t.identifiers[idx] != o.identifiers[idx] {
			return false
		}
	}
	return true
}

type arrayType struct {
	indexTypes  []dataType
	elementType dataType
	packed      bool
}

func (t *arrayType) Type() string {
	var indexTypes []string
	for _, it := range t.indexTypes {
		indexTypes = append(indexTypes, it.Type())
	}
	return fmt.Sprintf("array [%s] of %s", strings.Join(indexTypes, ", "), t.elementType.Type())
}

func (t *arrayType) Equals(dt dataType) bool {
	o, ok := dt.(*arrayType)
	if !ok {
		return false
	}
	if t.elementType.Type() != o.elementType.Type() {
		return false
	}
	if len(t.indexTypes) != len(o.indexTypes) {
		return false
	}
	for idx := range t.indexTypes {
		if !t.indexTypes[idx].Equals(o.indexTypes[idx]) {
			return false
		}
	}
	return true
}

type recordType struct {
	fields []*recordField
	packed bool
}

func (t *recordType) findField(name string) *recordField {
	for _, f := range t.fields {
		for _, id := range f.Identifiers {
			if id == name {
				return f
			}
		}
	}
	return nil
}

func (t *recordType) Type() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "record ")
	for idx, f := range t.fields {
		if idx > 0 {
			fmt.Fprint(&buf, "; ")
		}
		fmt.Fprintf(&buf, "%s", f)
	}
	fmt.Fprint(&buf, "end")
	return buf.String()
}

func (t *recordType) Equals(dt dataType) bool {
	o, ok := dt.(*recordType)
	if !ok {
		return false
	}

	if len(t.fields) != len(o.fields) {
		return false
	}

	for idx := range t.fields {
		if t.fields[idx].String() != o.fields[idx].String() {
			return false
		}
	}

	return true
}

type setType struct {
	elementType dataType
	packed      bool
}

func (t *setType) Type() string {
	return fmt.Sprintf("set of %s", t.elementType.Type())
}

func (t *setType) Equals(dt dataType) bool {
	o, ok := dt.(*setType)
	return ok && t.elementType.Equals(o.elementType)
}

type integerType struct{}

func (t *integerType) Type() string {
	return "integer"
}

func (t *integerType) Equals(dt dataType) bool {
	_, ok := dt.(*integerType)
	return ok
}

type booleanType struct{}

func (t *booleanType) Type() string {
	return "boolean"
}

func (t *booleanType) Equals(dt dataType) bool {
	_, ok := dt.(*booleanType)
	return ok
}

type stringType struct{}

func (t *stringType) Type() string {
	return "string"
}

func (t *stringType) Equals(dt dataType) bool {
	_, ok := dt.(*stringType)
	return ok
}

type realType struct{}

func (t *realType) Type() string {
	return "real"
}

func (t *realType) Equals(dt dataType) bool {
	_, ok := dt.(*realType)
	return ok
}
