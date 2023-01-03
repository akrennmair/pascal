package parser

import (
	"errors"
	"fmt"
	"strings"
)

type pointerType struct {
	name string
	typ  dataType
}

func (t *pointerType) Type() string {
	if t.typ == nil {
		return "nil" // compatible with any type; strictly speaking, this is not syntactically correct in Pascal as a type.
	}

	if t.name != "" { // if there is a name, print name (even if it has been resolved) to avoid infinite recursion.
		return fmt.Sprintf("^%s", t.name)
	}

	return fmt.Sprintf("^%s", t.typ.Type())
}

func (t *pointerType) Equals(dt dataType) bool {
	o, ok := dt.(*pointerType)
	if !ok {
		return false
	}

	if t.typ == nil || o.typ == nil { // means at least one of them is a nil pointer, and nil is compatible with any type.
		return true
	}

	return t.typ.Equals(o.typ)
}

type subrangeType struct {
	lowerBound int
	upperBound int
	typ        dataType
}

func (t *subrangeType) Type() string {
	lb := fmt.Sprint(t.lowerBound)
	ub := fmt.Sprint(t.upperBound)
	if et, ok := t.typ.(*enumType); ok {
		if t.lowerBound >= 0 && t.lowerBound < len(et.identifiers) && t.upperBound >= 0 && t.upperBound < len(et.identifiers) {
			lb = et.identifiers[t.lowerBound]
			ub = et.identifiers[t.upperBound]
		}
	}
	return fmt.Sprintf("%s..%s", lb, ub)
}

func (t *subrangeType) Equals(dt dataType) bool {
	o, ok := dt.(*subrangeType)
	if !ok {
		return false
	}

	if t.typ != o.typ {
		return false
	}

	if t.typ != nil && !t.typ.Equals(o.typ) {
		return false
	}

	return t.lowerBound == o.lowerBound && t.upperBound == o.upperBound
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
	var buf strings.Builder

	if t.packed {
		buf.WriteString("packed ")
	}

	buf.WriteString("array [")

	for idx, it := range t.indexTypes {
		if idx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(it.Type())
	}

	buf.WriteString("] of ")
	buf.WriteString(t.elementType.Type())
	return buf.String()
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
	if t.packed {
		buf.WriteString("packed ")
	}
	buf.WriteString("record ")
	t.printFieldList(&buf, t)
	buf.WriteString("end")
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
	packed := ""
	if t.packed {
		packed = "packed "
	}
	return fmt.Sprintf("%sset of %s", packed, t.elementType.Type())
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

type charType struct{}

func (t *charType) Type() string {
	return "char"
}

func (t *charType) Equals(dt dataType) bool {
	_, ok := dt.(*charType)
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
	packed      bool
}

func (t *fileType) Type() string {
	var buf strings.Builder
	if t.packed {
		buf.WriteString("packed ")
	}
	buf.WriteString("file of ")
	buf.WriteString(t.elementType.Type())
	return buf.String()
}

func (t *fileType) Equals(dt dataType) bool {
	o, ok := dt.(*fileType)
	return ok && t.elementType.Equals(o.elementType)
}

type procedureType struct {
	params []*formalParameter
}

func (t *procedureType) Type() string {
	var buf strings.Builder
	buf.WriteString("(")

	for idx, param := range t.params {
		if idx > 0 {
			buf.WriteString("; ")
			buf.WriteString(param.String())
		}
	}

	buf.WriteString(")")

	return buf.String()
}

func (t *procedureType) Equals(dt dataType) bool {
	o, ok := dt.(*procedureType)
	if !ok {
		return false
	}

	if len(t.params) != len(o.params) {
		return false
	}

	for idx := range t.params {
		if !t.params[idx].Type.Equals(o.params[idx].Type) {
			return false
		}
	}

	return true
}

type functionType struct {
	params     []*formalParameter
	returnType dataType
}

func (t *functionType) Type() string {
	var buf strings.Builder
	buf.WriteString("(")

	for idx, param := range t.params {
		if idx > 0 {
			buf.WriteString("; ")
			buf.WriteString(param.String())
		}
	}

	buf.WriteString(") : ")

	buf.WriteString(t.returnType.Type())

	return buf.String()
}

func (t *functionType) Equals(dt dataType) bool {
	o, ok := dt.(*functionType)
	if !ok {
		return false
	}

	if !t.returnType.Equals(o.returnType) {
		return false
	}

	if len(t.params) != len(o.params) {
		return false
	}

	for idx := range t.params {
		if !t.params[idx].Type.Equals(o.params[idx].Type) {
			return false
		}
	}

	return true
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

func (l *stringLiteral) IsCharLiteral() bool {
	// TODO: solve this neater.
	// TODO: deduplicate, as the same code is also used in *stringExpr
	return len(l.Value) == 1 ||
		(len(l.Value) == 3 && l.Value[0] == '\'' && l.Value[2] == '\'') ||
		l.Value == "''''"
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

type enumValueLiteral struct {
	Symbol string
	Value  int
	Type   dataType
}

func (l *enumValueLiteral) ConstantType() dataType {
	return l.Type
}

func (l *enumValueLiteral) Negate() (constantLiteral, error) {
	return nil, errors.New("can't negate enum value")
}

func (l *enumValueLiteral) String() string {
	return l.Symbol
}

func typesCompatible(t1, t2 dataType) bool {
	if t1.Equals(t2) {
		return true
	}

	if isIntegerType(t1) && isIntegerType(t2) {
		return true
	}

	// TODO: implement more cases of compatibility

	return false
}

func isIntegerType(dt dataType) bool {
	switch dt.(type) {
	case *integerType:
		return true
	case *subrangeType:
		return true
	}

	return false
}

func isRealType(dt dataType) bool {
	if _, ok := dt.(*realType); ok {
		return true
	}

	return false
}

func isCharStringLiteralAssignment(b *block, lexpr expression, rexpr expression) bool {
	se, isStringExpr := rexpr.(*stringExpr)

	var (
		sl              *stringLiteral
		isStringLiteral bool
	)

	sc, isStringConstant := rexpr.(*constantExpr)
	if isStringConstant {
		constDecl := b.findConstantDeclaration(sc.name)
		if constDecl != nil {
			sl, isStringLiteral = constDecl.Value.(*stringLiteral)
		}
	}

	/*
		fmt.Printf("rexpr = %s sl = %s isStringLiteral = %t\n", spew.Sdump(rexpr), sl, isStringLiteral)
		fmt.Printf("lexpr.IsVariabelExpr = %t\n", lexpr.IsVariableExpr())
		fmt.Printf("lexpr is char = %t\n", lexpr.Type().Equals(&charType{}))
		fmt.Printf("rexpr is string = %t\n", rexpr.Type().Equals(getBuiltinType("string")))
	*/

	return lexpr.IsVariableExpr() &&
		lexpr.Type().Equals(&charType{}) &&
		rexpr.Type().Equals(&stringType{}) &&
		((isStringExpr && se.IsCharLiteral()) || isStringLiteral && sl.IsCharLiteral())
}
