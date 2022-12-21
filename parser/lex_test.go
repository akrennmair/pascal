package parser

import "testing"

func TestLexer(t *testing.T) {
	testData := []string{
		"procedure readinteger (var f : text; var x : integer);",
		"array [1..100] of real",
		"array [Boolean] of colour",
		"type polar = record r : real; theta : angle end;",
		"1..100",
		"-10..+10",
		"red..green",
		"'0'..'9'",
		"array [Boolean] of array [1 .. 10] of array [size] of real",
		"array [Boolean] of array [1..10, size] of real",
		"packed array [1..10] of packed array [1..8] of Boolean",
		`record
			year : 0..2000;
			month : 1..12;
			day : 1..31
			end`,
		`record
				name, firstname : string;
				age : 0..99;
				case married : Boolean of
					true : (Spousesname : string);
					false : ( )
				end`,
		"set of (club, diamond, heart, spade)",
		"file of real",
		`type
				natural = 0..maxint;
				count = integer;
				range = integer;
				colour = (red, yellow, green, blue);
				sex = (male, female);
				year = 1900..1999;
				shape = (triangle, rectangle, circle);
				punchedcard = array [1..80] of char;
				charsequence = file of char;
				polar = record
				r : real;
				theta : angle
				end;
				indextype = 1..limit;
				vector = array [indextype] of real;`,
		`var
				x, y, z, max : real;
				i, j : integer;
				k : 0..9;
				p, q, r : Boolean;
				operator : (plus, minus, times);
				a : array [0..63] of real;
				c : colour;
				f : file of char;
				hue1, hue2 : set of colour;`,
		"a[i + j]",
		"m[k][1]",
		"m[k, 1]",
		"coord.theta",
		`procedure AddVectors (var A, B, C : array [low..high : natural] of real);
				var
				i : natural;
				begin
				for i := low to high do A[i] := B[i] + C[i]
				end { of AddVectors };`,
		`begin z := x ; x := y ; y := z end`,
		`type foo = ^int;`,
	}

testLoop:
	for idx, entry := range testData {
		t.Logf("%d. Lexing %q", idx, entry)
		l := lex("", entry)
		for item := l.nextItem(); item.typ != itemEOF; item = l.nextItem() {
			t.Logf("\titem = %#v", item)
			if item.typ == itemError {
				t.Errorf("%d. error: %s", idx, item.val)
				continue testLoop
			}
		}
	}
}
