package parser

import (
	"errors"
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
	if t.packed != o.packed {
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
	fields       []*recordField
	variantField *recordVariantField
	packed       bool
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
	t.printFieldList(&buf, t)
	fmt.Fprint(&buf, "end")
	return buf.String()
}

func (tt *recordType) printFieldList(buf *strings.Builder, r *recordType) {
	for idx, f := range r.fields {
		if idx > 0 {
			buf.WriteString("; ")
		}
		buf.WriteString(f.String())
	}
	if r.variantField != nil {
		if len(r.fields) > 0 {
			buf.WriteString("; ")
		}
		buf.WriteString("case ")
		if r.variantField.tagField != "" {
			buf.WriteString(r.variantField.tagField + ": ")
		}
		buf.WriteString(r.variantField.typeName)
		buf.WriteString(" of ")
		for idx, variant := range r.variantField.variants {
			if idx > 0 {
				buf.WriteString(", ")
				for jdx, label := range variant.caseLabels {
					if jdx > 0 {
						buf.WriteString(", ")
					}
					buf.WriteString(label.String())
				}
				buf.WriteString(": (")
				r.printFieldList(buf, variant.fields)
				buf.WriteString(")")
			}
		}
	}
}

func (t *recordType) Equals(dt dataType) bool {
	o, ok := dt.(*recordType)
	if !ok {
		return false
	}

	if t.packed != o.packed || len(t.fields) != len(o.fields) {
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
	return ok && t.elementType.Equals(o.elementType) && t.packed == o.packed
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

type fileType struct {
	elementType dataType
}

func (t *fileType) Type() string {
	return fmt.Sprintf("file of %s", t.elementType.Type())
}

func (t *fileType) Equals(dt dataType) bool {
	o, ok := dt.(*fileType)
	return ok && t.elementType.Equals(o.elementType)
}

type constantLiteral interface {
	ConstantType() dataType
	Negate() (constantLiteral, error)
	String() string
}

type integerLiteral struct {
	Value int
}

func (l *integerLiteral) ConstantType() dataType {
	return &integerType{}
}

func (l *integerLiteral) Negate() (constantLiteral, error) {
	return &integerLiteral{Value: -l.Value}, nil
}

func (l *integerLiteral) String() string {
	return fmt.Sprintf("%d", l.Value)
}

type floatLiteral struct {
	minus       bool
	beforeComma string
	afterComma  string
	scaleFactor int
}

func (l *floatLiteral) ConstantType() dataType {
	return &realType{}
}

func (l *floatLiteral) Negate() (constantLiteral, error) {
	nl := &floatLiteral{}
	*nl = *l
	nl.minus = !nl.minus
	return nl, nil
}

func (l *floatLiteral) String() string {
	var buf strings.Builder
	if l.minus {
		buf.WriteByte('-')
	}
	buf.WriteString(l.beforeComma)
	buf.WriteByte('.')
	buf.WriteString(l.afterComma)
	if l.scaleFactor != 0 {
		buf.WriteString("e ")
		fmt.Fprintf(&buf, "%d", l.scaleFactor)
	}
	return buf.String()
}

type stringLiteral struct {
	Value string
}

func (l *stringLiteral) ConstantType() dataType {
	return &stringType{}
}

func (l *stringLiteral) Negate() (constantLiteral, error) {
	return nil, errors.New("can't negate string literal")
}

func (l *stringLiteral) String() string {
	var buf strings.Builder

	buf.WriteString("'")

	for _, r := range l.Value {
		buf.WriteRune(r)
		if r == '\'' {
			buf.WriteRune(r)
		}
	}

	buf.WriteString("'")

	return buf.String()
}
