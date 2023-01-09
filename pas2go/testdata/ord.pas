program test;

type ttt = ( foo, bar, baz );

var x : ttt;
	c : char;

begin
	c := 'A';
	writeln('ord(A) = ', ord(c));
	writeln('ord(a) = ', ord('a'));
	x := foo;
	writeln('ord(x) = ', ord(x));
	writeln('ord(baz) = ', ord(baz))
end.
