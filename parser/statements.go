package parser

type StatementType int

const (
	StatementGoto StatementType = iota
	StatementAssignment
	StatementProcedureCall
	StatementCompoundStatement
	StatementWhile
	StatementRepeat
	StatementFor
	StatementIf
	StatementCase
	StatementWith
	StatementWrite
)

type Statement interface {
	Type() StatementType
	Label() *string
}

// GotoStatement describes a goto statement.
type GotoStatement struct {
	label *string

	// The target label where execution shall continue next.
	Target string
}

func (s *GotoStatement) Type() StatementType {
	return StatementGoto
}

func (s *GotoStatement) Label() *string {
	return s.label
}

// CompoundStatement describes a grouped list of statements.
type CompoundStatement struct {
	label *string

	// The list of statements contained in the compound statements.
	Statements []Statement
}

func (s *CompoundStatement) Type() StatementType {
	return StatementCompoundStatement
}

func (s *CompoundStatement) Label() *string {
	return s.label
}

// WhileStatement describes a looping statement that continues to execute
// the provided statement as long as the condition evaluates true. The statement
// is executed zero times or more, i.e. the condition is evaluated before the
// statement is executed for the first time.
type WhileStatement struct {
	label     *string
	Condition Expression
	Statement Statement
}

func (s *WhileStatement) Type() StatementType {
	return StatementWhile
}

func (s *WhileStatement) Label() *string {
	return s.label
}

// RepeatStatement describes a looping statement that continues to execute
// the provided statement sequence until the condition evaluates true. The
// statement sequence is executed one time or more, i.e. the condition
// is evaluated only after the statement sequence has been executed for the first
// time.
type RepeatStatement struct {
	label      *string
	Condition  Expression
	Statements []Statement
}

func (s *RepeatStatement) Type() StatementType {
	return StatementRepeat
}

func (s *RepeatStatement) Label() *string {
	return s.label
}

// ForStatement describes a looping statement that initializes a provided variable
// with an initial expression, then executes the provides the statement and increments
// or decrements the variable until it has reached the final expression. When the
// final expression is reached, the statement is executed for one last time.
type ForStatement struct {
	label *string

	// Variable name that is initialized with the initial expression.
	Name string

	// Initial expression.
	InitialExpr Expression

	// Final expression.
	FinalExpr Expression

	// Statement to execute.
	Statement Statement

	// If true, indicates that the variable is to be decremented rather than incremented.
	DownTo bool
}

func (s *ForStatement) Type() StatementType {
	return StatementFor
}

func (s *ForStatement) Label() *string {
	return s.label
}

// IfStatement describes a conditional statement. If the condition is true, the statement
// is executed. If an else statement is present and the condition is false, the else
// statement is executed.
type IfStatement struct {
	label         *string
	Condition     Expression
	Statement     Statement
	ElseStatement Statement
}

func (s *IfStatement) Type() StatementType {
	return StatementIf
}

func (s *IfStatement) Label() *string {
	return s.label
}

// AssignmentStatement describes an assignment of a value (the evaluated result from the expression
// on the right side of the assignment operator) to an assignable expression on the left
// side of the assignment operator.
type AssignmentStatement struct {
	label     *string
	LeftExpr  Expression
	RightExpr Expression
}

func (s *AssignmentStatement) Type() StatementType {
	return StatementAssignment
}

func (s *AssignmentStatement) Label() *string {
	return s.label
}

// ProcedureCallStatement describes a procedure call, including its name, the actual parameters
// provided, and, for validation purposes, the formal parameters of the procedure that is referenced.
type ProcedureCallStatement struct {
	label        *string
	Name         string
	ActualParams []Expression
	FormalParams []*FormalParameter
}

func (s *ProcedureCallStatement) Type() StatementType {
	return StatementProcedureCall
}

func (s *ProcedureCallStatement) Label() *string {
	return s.label
}

// CaseStatement describes a conditional statement. The provided expression is first evaluated, and
// depending on the value, the first case limb is chosen where the value matches any of the
// case labels. That case limb's statement is then executed. If no matching case limb can be found,
// then no statement is executed.
type CaseStatement struct {
	label     *string
	Expr      Expression
	CaseLimbs []*CaseLimb
}

func (s *CaseStatement) Type() StatementType {
	return StatementCase
}

func (s *CaseStatement) Label() *string {
	return s.label
}

// CaseLimb describes a case limb, which consists of a list of case labels (constants) and
// a statement.
type CaseLimb struct {
	Label     []ConstantLiteral
	Statement Statement
}

// WithStatement describes the with statement, which serves the purpose to denote fields of
// record variables by their field name instead of having to resort to the long-form notation
// within a statement.
type WithStatement struct {
	label *string

	// The record variables for which the field names can be used directly.
	RecordVariables []string

	// Block containing the record variables' fields declared as variables as well as the
	// statement where these are valid.
	Block *Block
}

func (s *WithStatement) Type() StatementType {
	return StatementWith
}

func (s *WithStatement) Label() *string {
	return s.label
}

// WriteStatement describes a write or writeln statement.
type WriteStatement struct {
	label *string

	// If true, writeln was called.
	AppendNewLine bool

	// The not nil, the first actual parameter was a file variable.
	FileVar Expression

	// The list of actual parameters. If the first parameter was a file variable,
	// it is not contained in this list.
	ActualParams []Expression
}

func (s *WriteStatement) Type() StatementType {
	return StatementWrite
}

func (s *WriteStatement) Label() *string {
	return s.label
}
