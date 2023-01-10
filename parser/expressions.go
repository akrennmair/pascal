package parser

import (
	"fmt"
	"strings"
)

// Expression very generally describes any expression in a Pascal program.
type Expression interface {
	// String returns a string representation of the expression for debugging purposes.
	String() string

	// Type returns the data type of the expression.
	Type() DataType

	// IsVariableExpr returns true if the expression is a variable expression, which means
	// that it can be used as a left expression in assignments.
	IsVariableExpr() bool

	// Reduce reduces nested expressions to the innermost single expression, as far as
	// possible. This is to remove overly complicated nesting of various expression types.
	Reduce() Expression
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

// RelationalOperator describes relational operators used in relational expressions.
type RelationalOperator string

const (
	OpEqual        RelationalOperator = "="
	OpNotEqual     RelationalOperator = "<>"
	OpLess         RelationalOperator = "<"
	OpLessEqual    RelationalOperator = "<="
	OpGreater      RelationalOperator = ">"
	OpGreaterEqual RelationalOperator = ">="
	OpIn           RelationalOperator = "in"
)

var itemTypeRelOps = map[itemType]RelationalOperator{
	itemEqual:        OpEqual,
	itemNotEqual:     OpNotEqual,
	itemLess:         OpLess,
	itemLessEqual:    OpLessEqual,
	itemGreater:      OpGreater,
	itemGreaterEqual: OpGreaterEqual,
	itemIn:           OpIn,
}

func itemTypeToRelationalOperator(typ itemType) RelationalOperator {
	op, ok := itemTypeRelOps[typ]
	if !ok {
		return RelationalOperator(fmt.Sprintf("INVALID(%d)", typ))
	}
	return op
}

// RelationalExpr expresses a relational expression in which a left expression
// is compared to a right expression. The resulting type of a relational expression
// is always boolean.
type RelationalExpr struct {
	Left     Expression
	Operator RelationalOperator
	Right    Expression
}

func (e *RelationalExpr) String() string {
	return fmt.Sprintf("relation<%s %s %s>", e.Left, e.Operator, e.Right)
}

func (e *RelationalExpr) Type() DataType {
	return booleanTypeDef.Type
}

func (e *RelationalExpr) IsVariableExpr() bool {
	return false
}

func (e *RelationalExpr) Reduce() Expression {
	return &RelationalExpr{
		Left:     e.Left.Reduce(),
		Operator: e.Operator,
		Right:    e.Right.Reduce(),
	}
}

func isAdditionOperator(typ itemType) bool {
	return typ == itemSign || typ == itemOr
}

// SimpleExpr describes a simple expression, which consists of an optional
// sign, a starting term expression, and an optional list of pairs of addition operators
// ("addition" in the loosest sense) and further term expressions.
type SimpleExpr struct {
	Sign  string
	First Expression
	Next  []*Addition
}

func (e *SimpleExpr) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "simple-expr:<%s %s", e.Sign, e.First)
	for _, add := range e.Next {
		fmt.Fprintf(&buf, " %s %s", add.Operator, add.Term)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *SimpleExpr) Type() DataType {
	return e.First.Type()
}

func (e *SimpleExpr) IsVariableExpr() bool {
	if e.Sign != "" || len(e.Next) > 0 {
		return false
	}

	return e.First.IsVariableExpr()
}

func (e *SimpleExpr) Reduce() Expression {
	if len(e.Next) == 0 && (e.Sign == "" || e.Sign == "+") {
		return e.First.Reduce()
	}
	ne := &SimpleExpr{
		Sign:  e.Sign,
		First: e.First.Reduce(),
	}

	for _, add := range e.Next {
		ne.Next = append(ne.Next, &Addition{
			Operator: add.Operator,
			Term:     add.Term.Reduce(),
		})
	}

	return ne
}

// AdditionOperator describes addition operators. "Addition" is used in a loose sense,
// is the preferred nomenclature in the Pascal EBNF, and primarily refers to its
// precedence.
type AdditionOperator string

const (
	OperatorAdd      AdditionOperator = "+"
	OperatorSubtract AdditionOperator = "-"
	OperatorOr       AdditionOperator = "or"
)

func tokenToAdditionOperator(t item) AdditionOperator {
	if t.typ == itemSign && (t.val == "+" || t.val == "-") {
		return AdditionOperator(t.val)
	}
	if t.typ == itemOr {
		return OperatorOr
	}
	return AdditionOperator(fmt.Sprintf("INVALID(%+v)", t))
}

// Addition describes an addition operator and a term used in a simple expression.
type Addition struct {
	Operator AdditionOperator
	Term     Expression
}

// TermExpr describes a term, which consists of a factor and an optional list of
// multiplication operators ("multiplication" in the loosest sense) and terms.
type TermExpr struct {
	First Expression
	Next  []*Multiplication
}

func (e *TermExpr) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "term-expr:<%s", e.First)
	for _, mul := range e.Next {
		fmt.Fprintf(&buf, " %s %s", mul.Operator, mul.Factor)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *TermExpr) Type() DataType {
	return e.First.Type()
}

func (e *TermExpr) IsVariableExpr() bool {
	if len(e.Next) > 0 {
		return false
	}

	return e.First.IsVariableExpr()
}

func (e *TermExpr) Reduce() Expression {
	if len(e.Next) == 0 {
		return e.First.Reduce()
	}

	ne := &TermExpr{
		First: e.First.Reduce(),
	}

	for _, mul := range ne.Next {
		ne.Next = append(ne.Next, &Multiplication{
			Operator: mul.Operator,
			Factor:   mul.Factor.Reduce(),
		})
	}
	return e
}

// Multipliciation describes a multiplication operator and a factor used in a term.
type Multiplication struct {
	Operator MultiplicationOperator
	Factor   Expression
}

// MultiplicationOperator describes a multiplication operator. "Multiplication" is used
// here in a loose sense, as it is the preferred nomenclature of the Pascal EBNF, and
// primarily refers to its precedence.
type MultiplicationOperator string

const (
	OperatorMultiply    MultiplicationOperator = "*"
	OperatorFloatDivide MultiplicationOperator = "/"
	OperatorDivide      MultiplicationOperator = "div"
	OperatorModulo      MultiplicationOperator = "mod"
	OperatorAnd         MultiplicationOperator = "and"
)

var itemTypMulOp = map[itemType]MultiplicationOperator{
	itemMultiply:    OperatorMultiply,
	itemFloatDivide: OperatorFloatDivide,
	itemDiv:         OperatorDivide,
	itemMod:         OperatorModulo,
	itemAnd:         OperatorAnd,
}

func itemTypeToMultiplicationOperator(typ itemType) MultiplicationOperator {
	op, ok := itemTypMulOp[typ]
	if !ok {
		return MultiplicationOperator(fmt.Sprintf("INVALID(%+v)", typ))
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

// ConstantExpr describes an expression that refers to a constant (defined elsewhere in the program)
// by its name and the type it represents.
type ConstantExpr struct {
	Name  string
	Type_ DataType
}

func (e *ConstantExpr) String() string {
	return fmt.Sprintf("constant:<%s : %s>", e.Name, e.Type_.Type())
}

func (e *ConstantExpr) Type() DataType {
	return e.Type_
}

func (e *ConstantExpr) IsVariableExpr() bool {
	return false
}

func (e *ConstantExpr) Reduce() Expression {
	return e
}

// VariableExpr describes an expression that refers to a variable or formal parameter (defined
// elsewhere in the program) by its name and the type it represents.
type VariableExpr struct {
	Name          string
	Type_         DataType
	VarDecl       *Variable
	ParamDecl     *FormalParameter
	IsReturnValue bool
}

func (e *VariableExpr) String() string {
	return fmt.Sprintf("variable:<%s : %s>", e.Name, e.Type_.Type())
}

func (e *VariableExpr) Type() DataType {
	return e.Type_
}

func (e *VariableExpr) IsVariableExpr() bool {
	return true
}

func (e *VariableExpr) Reduce() Expression {
	return e
}

// IntegerExpr describes a literal of type integer, as an expression.
type IntegerExpr struct {
	Value int64
}

func (e *IntegerExpr) String() string {
	return fmt.Sprintf("int:<%d>", e.Value)
}

func (e *IntegerExpr) Type() DataType {
	return &IntegerType{}
}

func (e *IntegerExpr) IsVariableExpr() bool {
	return false
}

func (e *IntegerExpr) Reduce() Expression {
	return e
}

// RealExpr describes a literal of type real, as an expression.
type RealExpr struct {
	Minus       bool
	BeforeComma string
	AfterComma  string
	ScaleFactor int
}

func (e *RealExpr) String() string {
	sign := ""
	if e.Minus {
		sign = "-"
	}
	return fmt.Sprintf("real:<%s%s.%se%d>", sign, e.BeforeComma, e.AfterComma, e.ScaleFactor)
}

func (e *RealExpr) Type() DataType {
	return &RealType{}
}

func (e *RealExpr) IsVariableExpr() bool {
	return false
}

func (e *RealExpr) Reduce() Expression {
	return e
}

// StringExpr describes a literal of type string, as an expression.
type StringExpr struct {
	Value string
}

func (e *StringExpr) String() string {
	return fmt.Sprintf("str:<%q>", e.Value)
}

func (e *StringExpr) Type() DataType {
	return &StringType{}
}

func (e *StringExpr) IsVariableExpr() bool {
	return false
}

func (e *StringExpr) Reduce() Expression {
	return e
}

type CharExpr struct {
	Value byte
}

func (e *CharExpr) String() string {
	return fmt.Sprintf("char:<'%c'>", e.Value)
}

func (e *CharExpr) Type() DataType {
	return charTypeDef.Type
}

func (e *CharExpr) IsVariableExpr() bool {
	return false
}

func (e *CharExpr) Reduce() Expression {
	return e
}

func (e *StringExpr) IsCharLiteral() bool {
	// TODO: solve this neater.
	return len(e.Value) == 1 ||
		(len(e.Value) == 3 && e.Value[0] == '\'' && e.Value[2] == '\'') ||
		e.Value == "''''"
}

// NilExpr describes the nil pointer, as an expression.
type NilExpr struct{}

func (e *NilExpr) String() string {
	return "nil"
}

func (e *NilExpr) Type() DataType {
	return &PointerType{Type_: nil} // nil means it's compatible with any type
}

func (e *NilExpr) IsVariableExpr() bool {
	return false
}

func (e *NilExpr) Reduce() Expression {
	return e
}

// NotExpr describes a NOT expression which negates another boolean expression.
type NotExpr struct {
	Expr Expression
}

func (e *NotExpr) String() string {
	return fmt.Sprintf("not:<%s>", e.Expr)
}

func (e *NotExpr) Type() DataType {
	return e.Expr.Type()
}

func (e *NotExpr) IsVariableExpr() bool {
	return false
}

func (e *NotExpr) Reduce() Expression {
	return e
}

// SetExpr describes a set literal, as an expression.
type SetExpr struct {
	Elements []Expression
}

func (e *SetExpr) String() string {
	var buf strings.Builder
	fmt.Fprint(&buf, "set-expr:<")
	for idx, elem := range e.Elements {
		if idx > 0 {
			fmt.Fprint(&buf, " ")
		}
		fmt.Fprintf(&buf, "%s", elem)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *SetExpr) Type() DataType {
	return &SetType{
		ElementType: e.Elements[0].Type(),
	}
}

func (e *SetExpr) IsVariableExpr() bool {
	return false
}

func (e *SetExpr) Reduce() Expression {
	return e
}

// SubExpr describes an expression that is surrounded by "(" and ")".
type SubExpr struct {
	Expr Expression
}

func (e *SubExpr) String() string {
	return fmt.Sprintf("sub-expr:<%s>", e.Expr)
}

func (e *SubExpr) Type() DataType {
	return e.Expr.Type()
}

func (e *SubExpr) IsVariableExpr() bool {
	return e.Expr.IsVariableExpr() // TODO: check whether this is correct.
}

func (e *SubExpr) Reduce() Expression {
	return e.Expr.Reduce()
}

// IndexedVariableExpr describes an indexed access of an element of an expression (which is of an array type).
// The type describes the returned data type, while IndexExprs contains the expressions for the array's
// dimensions.
type IndexedVariableExpr struct {
	Expr       Expression // an expression of type *arrayType
	Type_      DataType
	IndexExprs []Expression
}

func (e *IndexedVariableExpr) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "indexed-variable-expr:<(%s)[", e.Expr.String())
	for idx, expr := range e.IndexExprs {
		if idx > 0 {
			fmt.Fprint(&buf, ", ")
		}
		fmt.Fprintf(&buf, "%s", expr)
	}
	fmt.Fprint(&buf, "]>")
	return buf.String()
}

func (e *IndexedVariableExpr) Type() DataType {
	return e.Type_
}

func (e *IndexedVariableExpr) IsVariableExpr() bool {
	return true
}

func (e *IndexedVariableExpr) Reduce() Expression {
	ne := &IndexedVariableExpr{
		Expr:  e.Expr.Reduce(),
		Type_: e.Type_,
	}
	for _, ie := range e.IndexExprs {
		ne.IndexExprs = append(ne.IndexExprs, ie.Reduce())
	}
	return ne
}

// FunctionCallExpr describes a function being called by name (defined elsewhere in
// the program or as a system function), with the actual parameters
// as expressions. The data type of the expression is the function's return type.
type FunctionCallExpr struct {
	Name         string
	Type_        DataType
	ActualParams []Expression
	FormalParams []*FormalParameter
}

func (e *FunctionCallExpr) String() string {
	var buf strings.Builder

	fmt.Fprintf(&buf, "function-call-expr:<%s(", e.Name)
	for idx, param := range e.ActualParams {
		if idx > 0 {
			fmt.Fprintf(&buf, ", ")
		}
		fmt.Fprint(&buf, param)
	}
	fmt.Fprint(&buf, ")>")
	return buf.String()
}

func (e *FunctionCallExpr) Type() DataType {
	return e.Type_
}

func (e *FunctionCallExpr) IsVariableExpr() bool {
	return false
}

func (e *FunctionCallExpr) Reduce() Expression {
	ne := &FunctionCallExpr{
		Name:  e.Name,
		Type_: e.Type_,
	}

	for _, pe := range e.ActualParams {
		ne.ActualParams = append(ne.ActualParams, pe.Reduce())
	}

	return ne
}

// FieldDesignatorExpr describes an expression where a record type's field is accessed.
// Expr is an expression of a record type, Field contains the field name, while the type
// is the field's type and thus the expression's type.
type FieldDesignatorExpr struct {
	Expr  Expression
	Field string
	Type_ DataType
}

func (e *FieldDesignatorExpr) String() string {
	return fmt.Sprintf("field-designator-expr:<%s.%s>", e.Expr, e.Field)
}

func (e *FieldDesignatorExpr) Type() DataType {
	return e.Type_
}

func (e *FieldDesignatorExpr) IsVariableExpr() bool {
	return true
}

func (e *FieldDesignatorExpr) Reduce() Expression {
	return &FieldDesignatorExpr{
		Expr:  e.Expr.Reduce(),
		Field: e.Field,
		Type_: e.Type_,
	}
}

// EnumValueExpr describes an enum value, with the enum value's name, its integer value,
// and the enum data type it is of.
type EnumValueExpr struct {
	Name  string
	Value int
	Type_ DataType
}

func (e *EnumValueExpr) String() string {
	return fmt.Sprintf("enum-value-expr:<%s %d of type %s>", e.Name, e.Value, e.Type_.Type())
}

func (e *EnumValueExpr) Type() DataType {
	return e.Type_
}

func (e *EnumValueExpr) IsVariableExpr() bool {
	return true
}

func (e *EnumValueExpr) Reduce() Expression {
	return e
}

// DerefExpr describes a dereferencing expression, where a pointer is dereferenced to access
// the memory it points to, either for reading or writing purposes.
type DerefExpr struct {
	Expr Expression
}

func (e *DerefExpr) String() string {
	return fmt.Sprintf("deref-expr:<%s>", e.Expr)
}

func (e *DerefExpr) Type() DataType {
	t, ok := e.Expr.Type().(*PointerType)
	if !ok {
		panic("derefExpr was created with expression not of pointer type")
	}
	return t.Type_
}

func (e *DerefExpr) IsVariableExpr() bool {
	return e.Expr.IsVariableExpr()
}

func (e *DerefExpr) Reduce() Expression {
	return &DerefExpr{
		Expr: e.Expr.Reduce(),
	}
}

// FormatExpr is solely used to format actual parameters to the write and writeln procedures.
// Expr is what is to be written, the optional Width expression describes the overall width
// that is used to write the expression, and the decimal places expression indicates how
// many decimal places shall be shown if Expr's type is a real.
type FormatExpr struct {
	Expr          Expression
	Width         Expression
	DecimalPlaces Expression
}

func (e *FormatExpr) String() string {
	var buf strings.Builder

	buf.WriteString("format-expr:<")
	buf.WriteString(e.Expr.String())
	if e.Width != nil {
		buf.WriteString(":")
		buf.WriteString(e.Width.String())
	}
	if e.DecimalPlaces != nil {
		buf.WriteString(":")
		buf.WriteString(e.DecimalPlaces.String())
	}
	buf.WriteString(">")
	return buf.String()
}

func (e *FormatExpr) Type() DataType {
	return e.Expr.Type()
}

func (e *FormatExpr) IsVariableExpr() bool {
	return e.Expr.IsVariableExpr()
}

func (e *FormatExpr) Reduce() Expression {
	return e
}

func decodeStringLiteral(s string) string {
	if s[0] == '\'' {
		s = s[1:]
	}
	if s[len(s)-1] == '\'' {
		s = s[:len(s)-1]
	}

	var (
		buf                 strings.Builder
		skipNextSingleQuote bool
	)

	for _, r := range s {
		if skipNextSingleQuote {
			skipNextSingleQuote = false
			if r == '\'' {
				continue
			}
		}
		buf.WriteRune(r)
		if r == '\'' {
			skipNextSingleQuote = true
		}
	}

	out := buf.String()

	return out
}
