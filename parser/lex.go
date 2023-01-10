package parser

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type item struct {
	typ itemType
	pos pos
	val string
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	}
	return fmt.Sprintf("%q", i.val)
}

type itemType int

type pos int

const (
	itemError itemType = iota
	itemEOF
	itemAnd
	itemArray
	itemBegin
	itemCase
	itemConst
	itemDiv
	itemDo
	itemDownto
	itemElse
	itemEnd
	itemFile
	itemFor
	itemFunction
	itemGoto
	itemIf
	itemIn
	itemLabel
	itemMod
	itemNil
	itemNot
	itemOf
	itemOr
	itemPacked
	itemProcedure
	itemProgram
	itemRecord
	itemRepeat
	itemSet
	itemThen
	itemTo
	itemTyp
	itemUntil
	itemVar
	itemWhile
	itemWith
	itemSign
	itemUnsignedDigitSequence
	itemAssignment
	itemColon
	itemGreaterEqual
	itemGreater
	itemLessEqual
	itemLess
	itemEqual
	itemNotEqual
	itemIdentifier
	itemSemicolon
	itemOpenParen
	itemCloseParen
	itemOpenBracket
	itemCloseBracket
	itemComma
	itemRange
	itemDot
	itemDoubleDot
	itemStringLiteral
	itemCaret
	itemMultiply
	itemFloatDivide
	itemForward
)

var key = map[string]itemType{
	"and":       itemAnd,
	"array":     itemArray,
	"begin":     itemBegin,
	"case":      itemCase,
	"const":     itemConst,
	"div":       itemDiv,
	"do":        itemDo,
	"downto":    itemDownto,
	"else":      itemElse,
	"end":       itemEnd,
	"file":      itemFile,
	"for":       itemFor,
	"function":  itemFunction,
	"goto":      itemGoto,
	"if":        itemIf,
	"in":        itemIn,
	"label":     itemLabel,
	"mod":       itemMod,
	"nil":       itemNil,
	"not":       itemNot,
	"of":        itemOf,
	"or":        itemOr,
	"packed":    itemPacked,
	"procedure": itemProcedure,
	"program":   itemProgram,
	"record":    itemRecord,
	"repeat":    itemRepeat,
	"set":       itemSet,
	"then":      itemThen,
	"to":        itemTo,
	"type":      itemTyp,
	"until":     itemUntil,
	"var":       itemVar,
	"while":     itemWhile,
	"with":      itemWith,
	"forward":   itemForward,
}

const eof = -1

type stateFn func(*lexer) stateFn

type lexer struct {
	name    string
	input   string
	state   stateFn
	pos     pos
	start   pos
	width   pos
	lastPos pos
	items   chan item
}

func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = pos(w)
	l.pos += l.width
	return r
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) emitIdentifier(s string) {
	l.items <- item{itemIdentifier, l.start, s}
	l.start = l.pos
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

func (l *lexer) lineNumber() int {
	return 1 + strings.Count(l.input[:l.lastPos], "\n")
}

func (l *lexer) columnInLine() int {
	bolPos := strings.LastIndex(l.input[:l.lastPos], "\n")
	return int(l.lastPos) - bolPos + 1
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, fmt.Sprintf(format, args...)}
	return nil
}

func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.pos
	return item
}

func lex(name, input string) *lexer {
	l := &lexer{
		name:  name,
		input: input,
		items: make(chan item),
	}
	go l.run()
	return l
}

func (l *lexer) run() {
	for l.state = lexText; l.state != nil; {
		l.state = l.state(l)
	}
}

func lexText(l *lexer) stateFn {
	r := l.peek()
	switch {
	case r == ' ' || r == '\n' || r == '\r' || r == '\t':
		l.acceptRun("\r\n\t ")
		l.ignore()
		return lexText
	case r >= '0' && r <= '9':
		return lexUnsignedDigitSequence
	case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
		return lexIdentifier
	case r == '+' || r == '-':
		l.next()
		l.emit(itemSign)
		return lexText
	case r == '=' || r == '<' || r == '>':
		return lexRelationalOperator
	case r == ':':
		return lexColonOrAssignment
	case r == ';':
		l.next()
		l.emit(itemSemicolon)
		return lexText
	case r == '(':
		l.next()
		switch l.peek() {
		case '*':
			l.next()
			return lexDigraphComment
		case '.':
			l.next()
			l.emit(itemOpenBracket)
			return lexText
		}
		l.emit(itemOpenParen)
		return lexText
	case r == ')':
		l.next()
		l.emit(itemCloseParen)
		return lexText
	case r == '[':
		l.next()
		l.emit(itemOpenBracket)
		return lexText
	case r == ']':
		l.next()
		l.emit(itemCloseBracket)
		return lexText
	case r == ',':
		l.next()
		l.emit(itemComma)
		return lexText
	case r == '\'':
		return lexStringLiteral
	case r == '{':
		return lexComment
	case r == '*':
		l.next()
		l.emit(itemMultiply)
		return lexText
	case r == '/':
		l.next()
		l.emit(itemFloatDivide)
		return lexText
	case r == '.':
		l.next()
		r = l.peek()
		switch r {
		case '.':
			l.next()
			l.emit(itemDoubleDot)
		case ')':
			l.next()
			l.emit(itemCloseBracket)
		default:
			l.emit(itemDot)
		}
		return lexText
	case r == '^':
		l.next()
		l.emit(itemCaret)
		return lexText
	case r == '@':
		l.next()
		l.emit(itemCaret)
		return lexText
	case r == eof:
		l.emit(itemEOF)
		return nil
	}
	return l.errorf("unknown token: %s", l.input[l.pos:])
}

func lexUnsignedDigitSequence(l *lexer) stateFn {
	l.acceptRun("0123456789")
	l.emit(itemUnsignedDigitSequence)
	return lexText
}

func lexIdentifier(l *lexer) stateFn {
	l.acceptRun("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	ident := strings.ToLower(l.input[l.start:l.pos])
	if typ, found := key[ident]; found {
		l.emit(typ)
	} else {
		l.emitIdentifier(ident)
	}
	return lexText
}

func lexRelationalOperator(l *lexer) stateFn {
	r := l.next()
	switch r {
	case '=':
		l.emit(itemEqual)
	case '<':
		r = l.peek()
		switch r {
		case '=':
			l.next()
			l.emit(itemLessEqual)
		case '>':
			l.next()
			l.emit(itemNotEqual)
		default:
			l.emit(itemLess)
		}
	case '>':
		r = l.peek()
		if r == '=' {
			l.next()
			l.emit(itemGreaterEqual)
		} else {
			l.emit(itemGreater)
		}
	default:
		return l.errorf("unexpected %c", r)
	}
	return lexText
}

func lexColonOrAssignment(l *lexer) stateFn {
	r := l.next()
	if r = l.peek(); r == '=' {
		l.next()
		l.emit(itemAssignment)
	} else {
		l.emit(itemColon)
	}
	return lexText
}

func lexStringLiteral(l *lexer) stateFn {
	seenFinalQuote := false // this is only there in case the closing ' is the final character in the text to parse; mostly necessary for expression parsing testing.
	r := l.next()
	for r = l.next(); r != eof; r = l.next() {
		if r == '\'' { // if the current character is ', then we peek to the next one.
			r = l.peek()
			if r != '\'' { // if it also a ', then we just go to next one, otherwise we've hit the final ' of a string.
				seenFinalQuote = true
				break
			}
			l.next()
		}
	}

	if seenFinalQuote || r != eof {
		l.emit(itemStringLiteral)
	}
	return lexText
}

func lexComment(l *lexer) stateFn {
	l.next()
	for r := l.next(); r != eof && r != '}'; r = l.next() {
	}
	l.ignore()
	return lexText
}

func lexDigraphComment(l *lexer) stateFn {
	l.next()
	for r := l.next(); r != eof && !(r == '*' && l.peek() == ')'); r = l.next() {
	}
	l.next()
	l.ignore()
	return lexText
}
