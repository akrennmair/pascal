package parser

type LabelledStatement struct {
	label string
	Statement
}

func (s *LabelledStatement) Type() StatementType {
	return s.Type()
}

func (s *LabelledStatement) Label() *string {
	return &s.label
}

type GotoStatement struct {
	Target string
}

func (s *GotoStatement) Type() StatementType {
	return StatementGoto
}

func (s *GotoStatement) Label() *string {
	return nil
}

type CompoundStatement struct {
	Statements []Statement
}

func (s *CompoundStatement) Type() StatementType {
	return StatementCompoundStatement
}

func (s *CompoundStatement) Label() *string {
	return nil
}

type WhileStatement struct {
	Condition Expression
	Statement Statement
}

func (s *WhileStatement) Type() StatementType {
	return StatementWhile
}

func (s *WhileStatement) Label() *string {
	return nil
}

type RepeatStatement struct {
	Condition  Expression
	Statements []Statement
}

func (s *RepeatStatement) Type() StatementType {
	return StatementRepeat
}

func (s *RepeatStatement) Label() *string {
	return nil
}

type ForStatement struct {
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
	return nil
}

type IfStatement struct {
	Condition     Expression
	Statement     Statement
	ElseStatement Statement
}

func (s *IfStatement) Type() StatementType {
	return StatementIf
}

func (s *IfStatement) Label() *string {
	return nil
}

type AssignmentStatement struct {
	LeftExpr  Expression
	RightExpr Expression
}

func (s *AssignmentStatement) Type() StatementType {
	return StatementAssignment
}

func (s *AssignmentStatement) Label() *string {
	return nil
}

type ProcedureCallStatement struct {
	Name         string
	ActualParams []Expression
}

func (s *ProcedureCallStatement) Type() StatementType {
	return StatementProcedureCall
}

func (s *ProcedureCallStatement) Label() *string {
	return nil
}

type CaseStatement struct {
	Expr      Expression
	CaseLimbs []*CaseLimb
}

func (s *CaseStatement) Type() StatementType {
	return StatementCase
}

func (s *CaseStatement) Label() *string {
	return nil
}

type CaseLimb struct {
	Label     []ConstantLiteral
	Statement Statement
}

type WithStatement struct {
	RecordVariables []string
	Block           *Block
}

func (s *WithStatement) Type() StatementType {
	return StatementWith
}

func (s *WithStatement) Label() *string {
	return nil
}

type WriteStatement struct {
	AppendNewLine bool       // if true, writeln instead of write is meant.
	FileVar       Expression // file variable; can be nil.
	ActualParams  []Expression
}

func (s *WriteStatement) Type() StatementType {
	return StatementWrite
}

func (s *WriteStatement) Label() *string {
	return nil
}
