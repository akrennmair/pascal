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

type GotoStatement struct {
	label  *string
	Target string
}

func (s *GotoStatement) Type() StatementType {
	return StatementGoto
}

func (s *GotoStatement) Label() *string {
	return s.label
}

type CompoundStatement struct {
	label      *string
	Statements []Statement
}

func (s *CompoundStatement) Type() StatementType {
	return StatementCompoundStatement
}

func (s *CompoundStatement) Label() *string {
	return s.label
}

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

type ForStatement struct {
	label       *string
	Name        string
	InitialExpr Expression
	FinalExpr   Expression
	Statement   Statement
	DownTo      bool
}

func (s *ForStatement) Type() StatementType {
	return StatementFor
}

func (s *ForStatement) Label() *string {
	return s.label
}

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

type CaseLimb struct {
	Label     []ConstantLiteral
	Statement Statement
}

type WithStatement struct {
	label           *string
	RecordVariables []string
	Block           *Block
}

func (s *WithStatement) Type() StatementType {
	return StatementWith
}

func (s *WithStatement) Label() *string {
	return s.label
}

type WriteStatement struct {
	label         *string
	AppendNewLine bool       // if true, writeln instead of write is meant.
	FileVar       Expression // file variable; can be nil.
	ActualParams  []Expression
}

func (s *WriteStatement) Type() StatementType {
	return StatementWrite
}

func (s *WriteStatement) Label() *string {
	return s.label
}
