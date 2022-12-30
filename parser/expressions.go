package parser

import (
	"fmt"
	"strings"
)

type expression interface {
	String() string
	Type() dataType
	IsVariableExpr() bool
	// TODO
}

func isRelationalOperator(typ itemType) bool {
	return typ == itemEqual ||
		typ == itemNotEqual ||
		typ == itemLess ||
		typ == itemLessEqual ||
		typ == itemGreater ||
		typ == itemGreaterEqual ||
		typ == itemIn
}

type relationalOperator string

const (
	opEqual        relationalOperator = "="
	opNotEqual     relationalOperator = "<>"
	opLess         relationalOperator = "<"
	opLessEqual    relationalOperator = "<="
	opGreater      relationalOperator = ">"
	opGreaterEqual relationalOperator = ">="
	opIn           relationalOperator = "in"
)

var itemTypeRelOps = map[itemType]relationalOperator{
	itemEqual:        opEqual,
	itemNotEqual:     opNotEqual,
	itemLess:         opLess,
	itemLessEqual:    opLessEqual,
	itemGreater:      opGreater,
	itemGreaterEqual: opGreaterEqual,
	itemIn:           opIn,
}

func itemTypeToRelationalOperator(typ itemType) relationalOperator {
	op, ok := itemTypeRelOps[typ]
	if !ok {
		return relationalOperator(fmt.Sprintf("INVALID(%d)", typ))
	}
	return op
}

type relationalExpr struct {
	left     *simpleExpression
	operator relationalOperator
	right    *simpleExpression
}

func (e *relationalExpr) String() string {
	return fmt.Sprintf("relation<%s %s %s>", e.left, e.operator, e.right)
}

func (e *relationalExpr) Type() dataType {
	return &booleanType{}
}

func (e *relationalExpr) IsVariableExpr() bool {
	return false
}

type minusExpr struct {
	expr *termExpr
}

func (e *minusExpr) String() string {
	return fmt.Sprintf("minus<%s>", e.expr)
}

func (e *minusExpr) Type() dataType {
	return e.expr.Type()
}

func (e *minusExpr) IsVariableExpr() bool {
	return false
}

func isAdditionOperator(typ itemType) bool {
	return typ == itemSign || typ == itemOr
}

type simpleExpression struct {
	sign  string
	first *termExpr
	next  []*addition
}

func (e *simpleExpression) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "simple-expr:<%s %s", e.sign, e.first)
	for _, add := range e.next {
		fmt.Fprintf(&buf, " %s %s", add.operator, add.term)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *simpleExpression) Type() dataType {
	return e.first.Type()
}

func (e *simpleExpression) IsVariableExpr() bool {
	if e.sign != "" || len(e.next) > 0 {
		return false
	}

	return e.first.IsVariableExpr()
}

type additionOperator string

const (
	opAdd      additionOperator = "+"
	opSubtract additionOperator = "-"
	opOr       additionOperator = "or"
)

func tokenToAdditionOperator(t item) additionOperator {
	if t.typ == itemSign && (t.val == "+" || t.val == "-") {
		return additionOperator(t.val)
	}
	if t.typ == itemOr {
		return opOr
	}
	return additionOperator(fmt.Sprintf("INVALID(%+v)", t))
}

type addition struct {
	operator additionOperator
	term     *termExpr
}

type termExpr struct {
	first factorExpr
	next  []*multiplication
}

func (e *termExpr) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "term-expr:<%s", e.first)
	for _, mul := range e.next {
		fmt.Fprintf(&buf, " %s %s", mul.operator, mul.factor)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *termExpr) Type() dataType {
	return e.first.Type()
}

func (e *termExpr) IsVariableExpr() bool {
	if len(e.next) > 0 {
		return false
	}

	return e.first.IsVariableExpr()
}

type multiplication struct {
	operator multiplicationOperator
	factor   factorExpr
}

type multiplicationOperator string

const (
	opMultiply    multiplicationOperator = "*"
	opFloatDivide multiplicationOperator = "/"
	opDivide      multiplicationOperator = "div"
	opModulo      multiplicationOperator = "mod"
	opAnd         multiplicationOperator = "and"
)

var itemTypMulOp = map[itemType]multiplicationOperator{
	itemMultiply:    opMultiply,
	itemFloatDivide: opFloatDivide,
	itemDiv:         opDivide,
	itemMod:         opModulo,
	itemAnd:         opAnd,
}

func itemTypeToMultiplicationOperator(typ itemType) multiplicationOperator {
	op, ok := itemTypMulOp[typ]
	if !ok {
		return multiplicationOperator(fmt.Sprintf("INVALID(%+v)", typ))
	}
	return op
}

func isMultiplicationOperator(typ itemType) bool {
	return typ == itemMultiply ||
		typ == itemFloatDivide ||
		typ == itemDiv ||
		typ == itemMod ||
		typ == itemAnd
}

type factorExpr interface {
	expression
}

type constantExpr struct {
	name string
	typ  dataType
}

func (e *constantExpr) String() string {
	return fmt.Sprintf("constant:<%s>", e.name)
}

func (e *constantExpr) Type() dataType {
	return e.typ
}

func (e *constantExpr) IsVariableExpr() bool {
	return false
}

type variableExpr struct {
	name string
	typ  dataType
}

func (e *variableExpr) String() string {
	return fmt.Sprintf("variable:<%s>", e.name)
}

func (e *variableExpr) Type() dataType {
	return e.typ
}

func (e *variableExpr) IsVariableExpr() bool {
	return true
}

type integerExpr struct {
	val int64
}

func (e *integerExpr) String() string {
	return fmt.Sprintf("int:<%d>", e.val)
}

func (e *integerExpr) Type() dataType {
	return &integerType{}
}

func (e *integerExpr) IsVariableExpr() bool {
	return false
}

type floatExpr struct {
	minus       bool
	beforeComma string
	afterComma  string
	scaleFactor int
}

func (e *floatExpr) String() string {
	sign := ""
	if e.minus {
		sign = "-"
	}
	return fmt.Sprintf("float:<%s%s.%se%d>", sign, e.beforeComma, e.afterComma, e.scaleFactor)
}

func (e *floatExpr) Type() dataType {
	return &realType{}
}

func (e *floatExpr) IsVariableExpr() bool {
	return false
}

type stringExpr struct {
	str string
}

func (e *stringExpr) String() string {
	return fmt.Sprintf("str:<%q>", e.str)
}

func (e *stringExpr) Type() dataType {
	return &stringType{}
}

func (e *stringExpr) IsVariableExpr() bool {
	return false
}

type nilExpr struct{}

func (e *nilExpr) String() string {
	return "nil"
}

func (e *nilExpr) Type() dataType {
	return &pointerType{
		name: "", // empty name indicates that it's compatible with any pointer type
	}
}

func (e *nilExpr) IsVariableExpr() bool {
	return false
}

type notExpr struct {
	expr factorExpr
}

func (e *notExpr) String() string {
	return fmt.Sprintf("not:<%s>", e.expr)
}

func (e *notExpr) Type() dataType {
	return e.expr.Type()
}

func (e *notExpr) IsVariableExpr() bool {
	return false
}

type setExpr struct {
	elements []expression
}

func (e *setExpr) String() string {
	var buf strings.Builder
	fmt.Fprint(&buf, "set-expr:<")
	for idx, elem := range e.elements {
		if idx > 0 {
			fmt.Fprint(&buf, " ")
		}
		fmt.Fprintf(&buf, "%s", elem)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *setExpr) Type() dataType {
	return &setType{
		elementType: e.elements[0].Type(),
	}
}

func (e *setExpr) IsVariableExpr() bool {
	return false
}

type subExpr struct {
	expr expression
}

func (e *subExpr) String() string {
	return fmt.Sprintf("sub-expr:<%s>", e.expr)
}

func (e *subExpr) Type() dataType {
	return e.expr.Type()
}

func (e *subExpr) IsVariableExpr() bool {
	return e.expr.IsVariableExpr() // TODO: check whether this is correct.
}

type indexedVariableExpr struct {
	name  string
	typ   dataType
	exprs []expression
}

func (e *indexedVariableExpr) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "indexed-variable-expr:<%s[", e.name)
	for idx, expr := range e.exprs {
		if idx > 0 {
			fmt.Fprint(&buf, ", ")
		}
		fmt.Fprintf(&buf, "%s", expr)
	}
	fmt.Fprint(&buf, "]>")
	return buf.String()
}

func (e *indexedVariableExpr) Type() dataType {
	return e.typ
}

func (e *indexedVariableExpr) IsVariableExpr() bool {
	return true
}

type functionCallExpr struct {
	name   string
	typ    dataType
	params []expression
}

func (e *functionCallExpr) String() string {
	var buf strings.Builder

	fmt.Fprintf(&buf, "function-call-expr:<%s(", e.name)
	for idx, param := range e.params {
		if idx > 0 {
			fmt.Fprintf(&buf, ", ")
		}
		fmt.Fprint(&buf, param)
	}
	fmt.Fprint(&buf, ")>")
	return buf.String()
}

func (e *functionCallExpr) Type() dataType {
	return e.typ
}

func (e *functionCallExpr) IsVariableExpr() bool {
	return false
}

type fieldDesignatorExpr struct {
	name  string
	field string
	typ   dataType
}

func (e *fieldDesignatorExpr) String() string {
	return fmt.Sprintf("field-designator-expr:<%s.%s>", e.name, e.field)
}

func (e *fieldDesignatorExpr) Type() dataType {
	return e.typ
}

func (e *fieldDesignatorExpr) IsVariableExpr() bool {
	return true
}

type enumValueExpr struct {
	symbol string
	value  int
	typ    dataType
}

func (e *enumValueExpr) String() string {
	return fmt.Sprintf("enum-value-expr:<%s %d of type %s>", e.symbol, e.value, e.typ.Type())
}

func (e *enumValueExpr) Type() dataType {
	return e.typ
}

func (e *enumValueExpr) IsVariableExpr() bool {
	return true
}
