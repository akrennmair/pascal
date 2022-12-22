package parser

import (
	"fmt"
	"strings"
)

type expression interface {
	String() string
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

type minusExpr struct {
	expr *termExpr
}

func (e *minusExpr) String() string {
	return fmt.Sprintf("minus<%s>", e.expr)
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

type identifierExpr struct {
	name string
}

func (e *identifierExpr) String() string {
	return fmt.Sprintf("ident:<%s>", e.name)
}

type integerExpr struct {
	val int64
}

func (e *integerExpr) String() string {
	return fmt.Sprintf("int:<%d>", e.val)
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

type stringExpr struct {
	str string
}

func (e *stringExpr) String() string {
	return fmt.Sprintf("str:<%q>", e.str)
}

type nilExpr struct{}

func (e *nilExpr) String() string {
	return "nil"
}

type notExpr struct {
	expr factorExpr
}

func (e *notExpr) String() string {
	return fmt.Sprintf("not:<%s>", e.expr)
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

type subExpr struct {
	expr expression
}

func (e *subExpr) String() string {
	return fmt.Sprintf("sub-expr:<%s>", e.expr)
}

type indexedVariableExpr struct {
	name  string
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

type functionCallExpr struct {
	name   string
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

type fieldDesignatorExpr struct {
	name  string
	field string
}

func (e *fieldDesignatorExpr) String() string {
	return fmt.Sprintf("field-designator-expr:<%s.%s>", e.name, e.field)
}
