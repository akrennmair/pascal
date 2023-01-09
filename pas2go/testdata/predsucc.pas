program test;

type ttt = (foo, bar, baz);

var x : ttt;
	i : integer;

begin
	x := foo;
	writeln('x = ', x);
	x := succ(x);
	writeln('x = ', x);
	x := succ(x);
	writeln('x = ', x);

	x := pred(x);
	writeln('x = ', x);
	x := pred(x);
	writeln('x = ', x);

	i := 2;
	i := succ(i);
	writeln('i = ', i);
	i := pred(i);
	writeln('i = ', i)
end.
