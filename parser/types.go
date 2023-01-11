package parser

import (
	"errors"
	"fmt"
	"strings"
)

type DataType interface {
	TypeString() string // TODO: rename to TypeString
	Equals(dt DataType) bool
	TypeName() string           // non-empty if type was looked up by name (not in type definition).
	Named(name string) DataType // produces a copy of the data type but with a name.
	Resolve(b *Block) error
}

// PointerType describes a type that is a pointer to another type.
type PointerType struct {
	// Name of the type. May be empty.
	Name string

	// Dereferenced type. If it is nil, it indicates that the associated value is
	// compatible with any other pointer type. This is used to represent the type
	// of the nil literal.
	Type_ DataType

	block *Block
}

func (t *PointerType) TypeString() string {
	if t.Name == "" && t.Type_ == nil {
		return "nil" // compatible with any type; strictly speaking, this is not syntactically correct in Pascal as a type.
	}

	if t.Name != "" { // if there is a name, print name (even if it has been resolved) to avoid infinite recursion.
		return fmt.Sprintf("^%s", t.Name)
	}

	return fmt.Sprintf("^%s", t.Type_.TypeString())
}

func (t *PointerType) Equals(dt DataType) bool {
	o, ok := dt.(*PointerType)
	if !ok {
		return false
	}

	if t.Type_ == nil || o.Type_ == nil { // means at least one of them is a nil pointer, and nil is compatible with any type.
		return true
	}

	return t.Type_.Equals(o.Type_)
}

func (t *PointerType) TypeName() string {
	return t.Type_.TypeName()
}

func (t *PointerType) Named(_ string) DataType {
	return t
}

func (t *PointerType) Resolve(b *Block) error {
	if t.Type_ == nil {
		if t.Name != "" {
			t.Type_ = b.findType(t.Name)
			if t.Type_ == nil {
				return fmt.Errorf("couldn't resolve type %q", t.Name)
			}
		} else {
			return fmt.Errorf("nameless pointer type")
		}
	}
	return nil
}

// SubrangeType describes a type that is a range with a lower and an upper boundary of an integral type.
type SubrangeType struct {
	LowerBound int
	UpperBound int
	Type_      DataType
	name       string
}

func (t *SubrangeType) TypeString() string {
	lb := fmt.Sprint(t.LowerBound)
	ub := fmt.Sprint(t.UpperBound)
	if et, ok := t.Type_.(*EnumType); ok {
		if t.LowerBound >= 0 && t.LowerBound < len(et.Identifiers) && t.UpperBound >= 0 && t.UpperBound < len(et.Identifiers) {
			lb = et.Identifiers[t.LowerBound]
			ub = et.Identifiers[t.UpperBound]
		}
	}
	return fmt.Sprintf("%s..%s", lb, ub)
}

func (t *SubrangeType) Equals(dt DataType) bool {
	o, ok := dt.(*SubrangeType)
	if !ok {
		return false
	}

	if t.Type_ != nil && !t.Type_.Equals(o.Type_) {
		return false
	}

	return t.LowerBound == o.LowerBound && t.UpperBound == o.UpperBound
}

func (t *SubrangeType) TypeName() string {
	return t.name
}

func (t *SubrangeType) Named(name string) DataType {
	nt := *t
	nt.name = name
	return &nt
}

func (t *SubrangeType) Resolve(_ *Block) error {
	return nil
}

// EnumType describes an enumerated type, consisting of a list of identifiers.
type EnumType struct {
	// List of identifiers. Their indexes are equal to their respective integer values.
	Identifiers []string
	name        string
}

func (t *EnumType) TypeString() string {
	if t.name != "" {
		return t.name
	}
	return fmt.Sprintf("(%s)", strings.Join(t.Identifiers, ", "))
}

func (t *EnumType) Equals(dt DataType) bool {
	o, ok := dt.(*EnumType)
	if !ok {
		return false
	}
	if len(t.Identifiers) != len(o.Identifiers) {
		return false
	}
	for idx := range t.Identifiers {
		if t.Identifiers[idx] != o.Identifiers[idx] {
			return false
		}
	}
	return true
}

func (t *EnumType) TypeName() string {
	return t.name
}

func (t *EnumType) Named(name string) DataType {
	nt := *t
	nt.name = name
	return &nt
}

func (t *EnumType) Resolve(_ *Block) error {
	return nil
}

// ArrayType describes an array type. IndexTypes contains the types of the dimensions
// of the array, which must either be a subrange type or an enumerated type. ElementType
// describes the data type of an individual element of the array. The Packed flag indicates
// whether the array type is packed or not.
type ArrayType struct {
	IndexTypes  []DataType
	ElementType DataType
	Packed      bool
	name        string
}

func (t *ArrayType) TypeString() string {
	var buf strings.Builder

	if t.Packed {
		buf.WriteString("packed ")
	}

	buf.WriteString("array [")

	for idx, it := range t.IndexTypes {
		if idx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(it.TypeString())
	}

	buf.WriteString("] of ")
	buf.WriteString(t.ElementType.TypeString())
	return buf.String()
}

func (t *ArrayType) Equals(dt DataType) bool {
	o, ok := dt.(*ArrayType)
	if !ok {
		return false
	}
	if t.Packed != o.Packed {
		return false
	}
	if t.ElementType.TypeString() != o.ElementType.TypeString() {
		return false
	}
	if len(t.IndexTypes) != len(o.IndexTypes) {
		return false
	}
	for idx := range t.IndexTypes {
		if !t.IndexTypes[idx].Equals(o.IndexTypes[idx]) {
			return false
		}
	}
	return true
}

func (t *ArrayType) TypeName() string {
	return t.name
}

func (t *ArrayType) Named(name string) DataType {
	nt := *t
	nt.name = name
	return &nt
}

func (t *ArrayType) Resolve(b *Block) error {
	return t.ElementType.Resolve(b)
}

// RecordType describes a record type, consisting of 1 or more fixed record fields, and an optional variant field.
// If a variant field is present, the fixed fields are optional.
type RecordType struct {
	// Fixed record fields.
	Fields []*RecordField

	// If not nil, contains the variant field.
	VariantField *RecordVariantField

	// If true, the record type was declared as packed.
	Packed bool
	name   string
}

func (t *RecordType) findField(name string) *RecordField {
	for _, f := range t.Fields {
		if f.Identifier == name {
			return f
		}
	}

	if t.VariantField != nil {
		if t.VariantField.TagField == name {
			return &RecordField{
				Identifier: t.VariantField.TagField,
				Type:       t.VariantField.Type,
			}
		}

		for _, variant := range t.VariantField.Variants {
			if rf := variant.Fields.findField(name); rf != nil {
				return rf
			}
		}
	}

	return nil
}

func (t *RecordType) TypeString() string {
	var buf strings.Builder
	if t.Packed {
		buf.WriteString("packed ")
	}
	buf.WriteString("record ")
	t.printFieldList(&buf, t)
	buf.WriteString("end")
	return buf.String()
}

func (tt *RecordType) printFieldList(buf *strings.Builder, r *RecordType) {
	for idx, f := range r.Fields {
		if idx > 0 {
			buf.WriteString("; ")
		}
		buf.WriteString(f.String())
	}
	if r.VariantField != nil {
		if len(r.Fields) > 0 {
			buf.WriteString("; ")
		}
		buf.WriteString("case ")
		if r.VariantField.TagField != "" {
			buf.WriteString(r.VariantField.TagField + ": ")
		}
		buf.WriteString(r.VariantField.Type.TypeName())
		buf.WriteString(" of ")
		for idx, variant := range r.VariantField.Variants {
			if idx > 0 {
				buf.WriteString(", ")
				for jdx, label := range variant.CaseLabels {
					if jdx > 0 {
						buf.WriteString(", ")
					}
					buf.WriteString(label.String())
				}
				buf.WriteString(": (")
				r.printFieldList(buf, variant.Fields)
				buf.WriteString(")")
			}
		}
	}
}

func (t *RecordType) Equals(dt DataType) bool {
	o, ok := dt.(*RecordType)
	if !ok {
		return false
	}

	if t.Packed != o.Packed || len(t.Fields) != len(o.Fields) {
		return false
	}

	for idx := range t.Fields {
		if t.Fields[idx].String() != o.Fields[idx].String() {
			return false
		}
	}

	return true
}

func (t *RecordType) TypeName() string {
	return t.name
}

func (t *RecordType) Named(name string) DataType {
	nt := *t
	nt.name = name
	return &nt
}

func (t *RecordType) Resolve(b *Block) error {
	for _, field := range t.Fields {
		if err := field.Type.Resolve(b); err != nil {
			return err
		}
	}

	if t.VariantField != nil {
		if err := t.VariantField.Type.Resolve(b); err != nil {
			return err
		}

		for _, variant := range t.VariantField.Variants {
			for _, field := range variant.Fields.Fields {
				if err := field.Type.Resolve(b); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// SetType describes a type that consists of a set of elements of a particular type.
type SetType struct {
	// The element type.
	ElementType DataType

	// If true, the set type was declared as packed.
	Packed bool

	name string
}

func (t *SetType) TypeString() string {
	packed := ""
	if t.Packed {
		packed = "packed "
	}
	return fmt.Sprintf("%sset of %s", packed, t.ElementType.TypeString())
}

func (t *SetType) Equals(dt DataType) bool {
	o, ok := dt.(*SetType)
	return ok && t.ElementType.Equals(o.ElementType) && t.Packed == o.Packed
}

func (t *SetType) TypeName() string {
	return t.name
}

func (t *SetType) Named(name string) DataType {
	nt := *t
	nt.name = name
	return &nt
}

func (t *SetType) Resolve(b *Block) error {
	return t.ElementType.Resolve(b)
}

// IntegerType describes the integer type.
type IntegerType struct {
	name string
}

func (t *IntegerType) TypeString() string {
	return "integer"
}

func (t *IntegerType) Equals(dt DataType) bool {
	_, ok := dt.(*IntegerType)
	return ok
}

func (t *IntegerType) TypeName() string {
	return t.name
}

func (t *IntegerType) Named(name string) DataType {
	nt := *t
	nt.name = name
	return &nt
}

func (t *IntegerType) Resolve(_ *Block) error {
	return nil
}

// StringType describes the string type.
type StringType struct {
	name string
}

func (t *StringType) TypeString() string {
	return "string"
}

func (t *StringType) Equals(dt DataType) bool {
	_, ok := dt.(*StringType)
	return ok
}

func (t *StringType) TypeName() string {
	return t.name
}

func (t *StringType) Named(name string) DataType {
	nt := *t
	nt.name = name
	return &nt
}

func (t *StringType) Resolve(_ *Block) error {
	return nil
}

// RealType describes the real type.
type RealType struct {
	name string
}

func (t *RealType) TypeString() string {
	return "real"
}

func (t *RealType) Equals(dt DataType) bool {
	_, ok := dt.(*RealType)
	return ok
}

func (t *RealType) TypeName() string {
	return t.name
}

func (t *RealType) Named(name string) DataType {
	nt := *t
	nt.name = name
	return &nt
}

func (t *RealType) Resolve(_ *Block) error {
	return nil
}

// FileType describes a file type. A file type consists of elements of a particular
// element type, which can be either read from the file or written to the file.
type FileType struct {
	// The element type.
	ElementType DataType

	// If true, indicates that the file type was declared as packed.
	Packed bool

	name string
}

func (t *FileType) TypeString() string {
	var buf strings.Builder
	if t.Packed {
		buf.WriteString("packed ")
	}
	buf.WriteString("file of ")
	buf.WriteString(t.ElementType.TypeString())
	return buf.String()
}

func (t *FileType) Equals(dt DataType) bool {
	o, ok := dt.(*FileType)
	return ok && t.ElementType.Equals(o.ElementType)
}

func (t *FileType) TypeName() string {
	return t.name
}

func (t *FileType) Named(name string) DataType {
	nt := *t
	nt.name = name
	return &nt
}

func (t *FileType) Resolve(b *Block) error {
	return t.ElementType.Resolve(b)
}

// TextType describes a text file.
type TextType struct {
	name string
}

func (t *TextType) TypeString() string {
	return "text"
}

func (t *TextType) Equals(dt DataType) bool {
	_, ok := dt.(*TextType)
	return ok
}

func (t *TextType) TypeName() string {
	return t.name
}

func (t *TextType) Named(name string) DataType {
	nt := *t
	nt.name = name
	return &nt
}

func (t *TextType) Resolve(b *Block) error {
	return nil
}

// ProcedureType describes a procedure by its formal parameters. This is only used
// as the type of formal parameters itself, i.e. in procedure and function declarations.
type ProcedureType struct {
	FormalParams []*FormalParameter
}

func (t *ProcedureType) TypeString() string {
	var buf strings.Builder
	buf.WriteString("(")

	for idx, param := range t.FormalParams {
		if idx > 0 {
			buf.WriteString("; ")
			buf.WriteString(param.String())
		}
	}

	buf.WriteString(")")

	return buf.String()
}

func (t *ProcedureType) Equals(dt DataType) bool {
	o, ok := dt.(*ProcedureType)
	if !ok {
		return false
	}

	if len(t.FormalParams) != len(o.FormalParams) {
		return false
	}

	for idx := range t.FormalParams {
		if !t.FormalParams[idx].Type.Equals(o.FormalParams[idx].Type) {
			return false
		}
	}

	return true
}

func (t *ProcedureType) TypeName() string {
	return ""
}

func (t *ProcedureType) Named(_ string) DataType {
	return t
}

func (t *ProcedureType) Resolve(b *Block) error {
	for _, param := range t.FormalParams {
		if err := param.Type.Resolve(b); err != nil {
			return err
		}
	}
	return nil
}

// FunctionType describes a function by its formal parameters and its return type.
// This is only used as the type of formal parameters itself, i.e. in procedure and
// function declarations.
type FunctionType struct {
	FormalParams []*FormalParameter
	ReturnType   DataType
}

func (t *FunctionType) TypeString() string {
	var buf strings.Builder
	buf.WriteString("(")

	for idx, param := range t.FormalParams {
		if idx > 0 {
			buf.WriteString("; ")
			buf.WriteString(param.String())
		}
	}

	buf.WriteString(") : ")

	buf.WriteString(t.ReturnType.TypeString())

	return buf.String()
}

func (t *FunctionType) Equals(dt DataType) bool {
	o, ok := dt.(*FunctionType)
	if !ok {
		return false
	}

	if !t.ReturnType.Equals(o.ReturnType) {
		return false
	}

	if len(t.FormalParams) != len(o.FormalParams) {
		return false
	}

	for idx := range t.FormalParams {
		if !t.FormalParams[idx].Type.Equals(o.FormalParams[idx].Type) {
			return false
		}
	}

	return true
}

func (t *FunctionType) TypeName() string {
	return ""
}

func (t *FunctionType) Named(_ string) DataType {
	return t
}

func (t *FunctionType) Resolve(b *Block) error {
	for _, param := range t.FormalParams {
		if err := param.Type.Resolve(b); err != nil {
			return err
		}
	}

	if err := t.ReturnType.Resolve(b); err != nil {
		return err
	}

	return nil
}

// ConstantLiteral very generally describes a constant literal.
type ConstantLiteral interface {
	ConstantType() DataType
	Negate() (ConstantLiteral, error)
	String() string
}

// IntegerLiteral describes an integer literal with a particular value.
type IntegerLiteral struct {
	// The literal's integer value.
	Value int
}

func (l *IntegerLiteral) ConstantType() DataType {
	return &IntegerType{}
}

func (l *IntegerLiteral) Negate() (ConstantLiteral, error) {
	return &IntegerLiteral{Value: -l.Value}, nil
}

func (l *IntegerLiteral) String() string {
	return fmt.Sprintf("%d", l.Value)
}

// RealLiteral describes a real literal with a particular value.
type RealLiteral struct {
	// If true, the real value is negative.
	Minus bool

	// Digits before the comma. May be empty.
	BeforeComma string

	// Digits after the comma. May be empty.
	AfterComma string

	// The scale factor. It indicates by which power of ten
	// the value represented before and after the comma needs
	// to be multiplied.
	ScaleFactor int
}

func (l *RealLiteral) ConstantType() DataType {
	return &RealType{}
}

func (l *RealLiteral) Negate() (ConstantLiteral, error) {
	nl := &RealLiteral{}
	*nl = *l
	nl.Minus = !nl.Minus
	return nl, nil
}

func (l *RealLiteral) String() string {
	var buf strings.Builder
	if l.Minus {
		buf.WriteByte('-')
	}
	buf.WriteString(l.BeforeComma)
	buf.WriteByte('.')
	buf.WriteString(l.AfterComma)
	if l.ScaleFactor != 0 {
		buf.WriteString("e ")
		fmt.Fprintf(&buf, "%d", l.ScaleFactor)
	}
	return buf.String()
}

// StringLiteral describes a string literal.
type StringLiteral struct {
	Value string
}

func (l *StringLiteral) ConstantType() DataType {
	return &StringType{}
}

func (l *StringLiteral) Negate() (ConstantLiteral, error) {
	return nil, errors.New("can't negate string literal")
}

func (l *StringLiteral) IsCharLiteral() bool {
	// TODO: solve this neater.
	// TODO: deduplicate, as the same code is also used in *stringExpr
	return len(l.Value) == 1 ||
		(len(l.Value) == 3 && l.Value[0] == '\'' && l.Value[2] == '\'') ||
		l.Value == "''''"
}

func (l *StringLiteral) String() string {
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

// CharLiteral describes a char literal
type CharLiteral struct {
	Value byte
}

func (l *CharLiteral) ConstantType() DataType {
	return charTypeDef.Type
}

func (l *CharLiteral) Negate() (ConstantLiteral, error) {
	return nil, errors.New("can't negate char literal")
}

func (l *CharLiteral) String() string {
	var buf strings.Builder

	buf.WriteString("'")
	buf.WriteByte(l.Value)
	buf.WriteString("'")

	return buf.String()
}

// EnumValueLiteral describes a literal of an enumerated type, both by
// the enumerated type's symbol and the integral value that is associated
// with it.
type EnumValueLiteral struct {
	Symbol string
	Value  int
	Type   DataType
}

func (l *EnumValueLiteral) ConstantType() DataType {
	return l.Type
}

func (l *EnumValueLiteral) Negate() (ConstantLiteral, error) {
	return nil, errors.New("can't negate enum value")
}

func (l *EnumValueLiteral) String() string {
	return l.Symbol
}

func typesCompatible(t1, t2 DataType) bool {
	if t1.Equals(t2) {
		return true
	}

	if isIntegerType(t1) && isIntegerType(t2) {
		return true
	}

	if (isIntegerType(t1) && isRealType(t2)) || (isRealType(t1) && isIntegerType(t2)) {
		return true
	}

	if srt, isSubrangeType := t1.(*SubrangeType); isSubrangeType {
		if typesCompatible(srt.Type_, t2) {
			return true
		}
	}

	if isSetType(t1) && isSetType(t2) {
		if typesCompatible(t1.(*SetType).ElementType, t2.(*SetType).ElementType) {
			return true
		}
	}

	// TODO: implement more cases of compatibility

	return false
}

func exprCompatible(t DataType, expr Expression) bool {
	if typesCompatible(t, expr.Type()) {
		return true
	}

	strLiteral, isStringLiteral := expr.(*StringExpr)

	if IsCharType(t) && (isStringLiteral && strLiteral.IsCharLiteral()) {
		return true
	}

	return false
}

func labelCompatibleWithType(label ConstantLiteral, typ DataType) bool {
	if typ.Equals(label.ConstantType()) {
		return true
	}

	sl, ok := label.(*StringLiteral)
	if ok && sl.IsCharLiteral() && typ.Equals(charTypeDef.Type) {
		return true
	}

	//fmt.Printf("label type %s, expression type is %s\n", label.ConstantType().Type(), typ.Type())

	return false
}

func typesCompatibleForAssignment(lt, rt DataType) bool {
	if lt.Equals(rt) {
		return true
	}

	if isSetType(lt) && isSetType(rt) {
		// if left and right side are both sets, and the right side has an empty type (b/c it's an empty literal)
		// then types are compatible for assignment. In addition, we assign the element type to the set type.
		if rt.(*SetType).ElementType == nil {
			rt.(*SetType).ElementType = lt.(*SetType).ElementType
			return true
		}

		if typesCompatible(lt.(*SetType).ElementType, rt.(*SetType).ElementType) {
			return true
		}
	}

	if rightSetType, ok := rt.(*SetType); isSetType(lt) && ok && rightSetType.ElementType == nil {
		rightSetType.ElementType = lt.(*SetType).ElementType
		return true
	}

	if isIntegerType(lt) && isIntegerType(rt) {
		return true
	}

	if isRealType(lt) && isIntegerType(rt) {
		return true
	}

	if isPackedCharArray(lt) && rt.Equals(&StringType{}) {
		return true
	}

	// TODO: implement more cases of compatibility

	return false
}

func isPackedCharArray(typ DataType) bool {
	arrType, ok := typ.(*ArrayType)
	if !ok {
		return false
	}

	if !arrType.Packed {
		return false
	}

	return arrType.Packed && len(arrType.IndexTypes) == 1 && IsCharType(arrType.ElementType)
}

func isIntegerType(dt DataType) bool {
	switch t := dt.(type) {
	case *IntegerType:
		return true
	case *SubrangeType:
		if t.Type_.Equals(&IntegerType{}) {
			return true
		}
	}

	return false
}

func isRealType(dt DataType) bool {
	_, ok := dt.(*RealType)
	return ok
}

func isSetType(dt DataType) bool {
	_, ok := dt.(*SetType)
	return ok
}

func isCharStringLiteralAssignment(b *Block, lexpr Expression, rexpr Expression) bool {
	se, isStringExpr := rexpr.(*StringExpr)

	var (
		sl              *StringLiteral
		isStringLiteral bool
	)

	sc, isStringConstant := rexpr.(*ConstantExpr)
	if isStringConstant {
		constDecl := b.findConstantDeclaration(sc.Name)
		if constDecl != nil {
			sl, isStringLiteral = constDecl.Value.(*StringLiteral)
		}
	}

	/*
		fmt.Printf("=====\n")
		fmt.Printf("lexpr = %s (%s)\n", spew.Sdump(lexpr), lexpr.Type().Type())
		fmt.Printf("rexpr = %s (%s) sl = %s isStringLiteral = %t\n", spew.Sdump(rexpr), rexpr.Type().Type(), sl, isStringLiteral)
		fmt.Printf("lexpr.IsVariabelExpr = %t\n", lexpr.IsVariableExpr())
		fmt.Printf("lexpr is char = %t\n", IsCharType(lexpr.Type()))
		fmt.Printf("rexpr is string = %t\n", rexpr.Type().Equals(getBuiltinType("string")))
	*/

	return lexpr.IsVariableExpr() &&
		IsCharType(lexpr.Type()) &&
		rexpr.Type().Equals(&StringType{}) &&
		((isStringExpr && se.IsCharLiteral()) || isStringLiteral && sl.IsCharLiteral())
}
