{$mode ISO}
program test;

type y = record
		c : ^integer;
	end;

var x : record
		a : ^integer;
		b : ^y;
	end;

begin
	x.a^ := x.b^.c^;
	x.b^.c^ := 23;
	x.b^.c^ := x.a^
end.
