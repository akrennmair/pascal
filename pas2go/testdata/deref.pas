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
	new(x.a);
	new(x.b);
	new(x.b^.c);
	x.a^ := x.b^.c^;
	x.b^.c^ := 23;
	x.b^.c^ := x.a^;
	dispose(x.b^.c);
	dispose(x.b);
	dispose(x.a)
end.
