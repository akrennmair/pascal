program test;

var i, j : integer;
	r, s : real;

begin
	i := -23;
	j := abs(i);
	r := -3.1415;
	s := abs(r);
	writeln('abs(', i, ') = ', j);
	writeln('abs(', r, ') = ', s);

	s := arctan(r);
	writeln('arctan(', r, ') = ', s)
end.
