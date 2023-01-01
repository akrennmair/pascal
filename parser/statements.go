package parser

type labelledStatement struct {
	label string
	statement
}

func (s *labelledStatement) Type() statementType {
	return s.Type()
}

func (s *labelledStatement) Label() *string {
	return &s.label
}

type statementGoto struct {
	target string
}

func (s *statementGoto) Type() statementType {
	return stmtGoto
}

func (s *statementGoto) Label() *string {
	return nil
}

type compoundStatement struct {
	statements []statement
}

func (s *compoundStatement) Type() statementType {
	return stmtCompoundStatement
}

func (s *compoundStatement) Label() *string {
	return nil
}

type whileStatement struct {
	condition expression
	stmt      statement
}

func (s *whileStatement) Type() statementType {
	return stmtWhile
}

func (s *whileStatement) Label() *string {
	return nil
}

type repeatStatement struct {
	condition expression
	stmts     []statement
}

func (s *repeatStatement) Type() statementType {
	return stmtRepeat
}

func (s *repeatStatement) Label() *string {
	return nil
}

type forStatement struct {
	name        string
	initialExpr expression
	finalExpr   expression
	body        statement
	down        bool
}

func (s *forStatement) Type() statementType {
	return stmtFor
}

func (s *forStatement) Label() *string {
	return nil
}

type ifStatement struct {
	condition expression
	body      statement
	elseBody  statement
}

func (s *ifStatement) Type() statementType {
	return stmtIf
}

func (s *ifStatement) Label() *string {
	return nil
}

type assignmentStatement struct {
	lexpr expression
	rexpr expression
}

func (s *assignmentStatement) Type() statementType {
	return stmtAssignment
}

func (s *assignmentStatement) Label() *string {
	return nil
}

type procedureCallStatement struct {
	name          string
	parameterList []expression
}

func (s *procedureCallStatement) Type() statementType {
	return stmtProcedureCall
}

func (s *procedureCallStatement) Label() *string {
	return nil
}

type caseStatement struct {
	expr      expression
	caseLimbs []*caseLimb
}

func (s *caseStatement) Type() statementType {
	return stmtCase
}

func (s *caseStatement) Label() *string {
	return nil
}

type caseLimb struct {
	labels []constantLiteral
	stmt   statement
}

type withStatement struct {
	recordVariables []string
	block           *block
}

func (s *withStatement) Type() statementType {
	return stmtWith
}

func (s *withStatement) Label() *string {
	return nil
}
