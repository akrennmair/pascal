package parser

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestParser(t *testing.T) {
	testData := []struct {
		name string
		code string
	}{
		{"empty", "program test; begin end."},
		{"label declaration", "program test; label 1; begin end."},
		{"multiple label declarations", "program test; label 1, 2, 3; begin end."},
		{"const declaration", "program test; const foo = 1; begin end."},
		{"multiple const declarations", "program test; const foo = 1; bar = 2; quux = 3; begin end."},
		{"type declaration alias", "program test; type bar = int; begin end."},
		{"type declaration pointer", "program test; type foo = ^integer; begin end."},
		{"type declaration enumerated", "program test; type foo = ( bar ); begin end."},
		{"type declaration enumerated 2", "program test; type foo = ( bar, baz, quux ); begin end."},
		{"type declaration set", "program test; type foo = set of integer; begin end."},
		{"type declaration set of pointers", "program test; type foo = set of ^integer; begin end."},
		{"type declaration set of enumerated", "program test; type foo = set of ( bar, baz, quux ); begin end."},
		{"type declaration files", "program test; type foo = file of integer; bar = file of set of integer; begin end."},
		{"type declaration packed set", "program test; type foo = packed set of integer; begin end."},
		{"type declaration integer variable", "program test; var foo : integer; begin end."},
		{"type declaration multiple sets of integers, and two reals", "program test; var foo, bar, baz : set of integer; quux, bla : real; begin end."},
		{"empty procedure", `program test; procedure foo; begin end; begin end.`},
		{"procedure with variables", `program test; procedure foo; var bar : integer; begin end; begin end.`},
		{"procedure with 1 parameter", `program test; procedure foo(a : integer); begin end; begin end.`},
		{"procedure with 2 parameters", `program test; procedure foo(a, b : integer); begin end; begin end.`},
		{"procedure with multiple parameters, including enum", `program test; procedure foo(a, b : integer; c, d : ^integer; e : ( bla, oops )); begin end; begin end.`},
		{"procedure with var parameters", `program test; procedure foo(var a, b : integer); begin end; begin end.`},
		{"function returning integer", `program test; function foo : integer; begin end; begin end.`},
		{"function with 1 parameter", `program test; function foo(a : integer) : integer; begin end; begin end.`},
		{"function with multiple parameters", `program test; function foo(a, b : real; var c, d : integer) : string; begin end; begin end.`},
		{"program with label declarations and gotos", `program test; label 23, 42; begin 23: goto 42; 42: goto 23 end.`},
		{"program with assignments", `program test; var a, b : integer; begin a := 3; b := a end.`},
		{"while loop with begin end block", `program test;
		begin
			while a do
			begin
				a := 3
			end
		end.`},
		{"for loop", `program test;
		var i : integer;
		begin
			for i := 0 to 23 do
				writeln(i)
		end.`},
		{"repeat until", `program test;
		begin
			repeat
				writeln(23)
			until a
		end.`},
		{"simple if", `program test;
		begin
			if a then
				writeln(23)
		end.`},
		{"if else", `program test;
		begin
			if a then
				writeln(23)
			else
				writeln(42)
		end.`},
		{"if else with begin end", `program test;
		begin
			if a then
			begin
				writeln(23)
			end
			else
			begin
				writeln(42)
			end
		end.`},
	}

	for idx, testEntry := range testData {
		t.Run(testEntry.name, func(t *testing.T) {
			p, err := parse(fmt.Sprintf("test_%d.pas", idx), testEntry.code)
			if err != nil {
				t.Errorf("%d. parse failed: %v", idx, err)
			}
			t.Logf("p = %s", spew.Sdump(*p))
		})
	}
}