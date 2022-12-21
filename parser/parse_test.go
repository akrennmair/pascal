package parser

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestParser(t *testing.T) {
	testData := []string{
		"program test; begin end.",
		"program test; label 1; begin end.",
		"program test; label 1, 2, 3; begin end.",
		"program test; const foo = 1; begin end.",
		"program test; const foo = 1; bar = 2; quux = 3; begin end.",
		"program test; type bar = int; begin end.",
		"program test; type foo = ^integer; begin end.",
		"program test; type foo = ( bar ); begin end.",
		"program test; type foo = ( bar, baz, quux ); begin end.",
		"program test; type foo = set of integer; begin end.",
		"program test; type foo = set of ^integer; begin end.",
		"program test; type foo = set of ( bar, baz, quux ); begin end.",
		"program test; type foo = file of integer; bar = file of set of integer; begin end.",
		"program test; type foo = packed set of integer; begin end.",
		"program test; var foo : integer; begin end.",
		"program test; var foo, bar, baz : set of integer; quux, bla : real; begin end.",
		`program test; procedure foo; begin end; begin end.`,
		`program test; procedure foo; var bar : integer; begin end; begin end.`,
		`program test; procedure foo(a : integer); begin end; begin end.`,
		`program test; procedure foo(a, b : integer); begin end; begin end.`,
		`program test; procedure foo(a, b : integer; c, d : ^integer; e : ( bla, oops )); begin end; begin end.`,
		`program test; procedure foo(var a, b : integer); begin end; begin end.`,
		`program test; function foo : integer; begin end; begin end.`,
		`program test; function foo(a : integer) : integer; begin end; begin end.`,
		`program test; function foo(a, b : real; var c, d : integer) : string; begin end; begin end.`,
		`program test; label 23, 42; begin 23: goto 42; 42: goto 23 end.`,
		`program test; var a, b : integer; begin a := 3; b := a end.`,
		`program test; begin while a do begin a := 3 end; end.`,
		`program test; var i : integer; begin for i := 0 to 23 do writeln(i) end.`,
	}

	for idx, testEntry := range testData {
		p, err := parse(fmt.Sprintf("test_%d.pas", idx), testEntry)
		if err != nil {
			t.Errorf("%d. parse failed: %v", idx, err)
		}
		t.Logf("%d. p = %s", idx, spew.Sdump(*p))
	}
}
