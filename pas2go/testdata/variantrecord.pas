program test;

type foo = record
		c : integer;
		d : real;
		case bla : integer of
			1, 2, 3: ( a : real );
			3, 4, 5: ( b : string );
		end;

var x : foo;

begin
	x.c := 42;
	x.d := 23.5;
	x.bla := 1;
	x.a := 42.23;
	x.b := 'judgement day'
end.
