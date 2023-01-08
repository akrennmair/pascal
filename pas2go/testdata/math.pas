program test;

var r : real;
	i : integer;
	c : char;
	b : boolean;

begin
	r := cos(23.5);
	writeln('cos(23.5) = ', r);
	r := frac(23.5);
	writeln('frac(23.5) = ', r);
	r := int(23.5);
	writeln('int(23.5) = ', r);
	r := ln(23.5);
	writeln('ln(23.5) = ', r);
	r := pi;
	writeln('pi = ', r);
	r := sin(23.5);
	writeln('sin(23.5) = ', r);
	r := sqr(23.5);
	writeln('sqr(23.5) = ', r);
	r := sqrt(23.5);
	writeln('sqrt(23.5) = ', r);
	i := trunc(23.6);
	writeln('trunc(23.6) = ', i);
	i := round(23.6);
	writeln('round(23.6) = ', i);
	c := chr(40);
	writeln('chr(40) = ', c);
	b := odd(2);
	writeln('odd(2) = ', b);
	b := odd(23);
	writeln('odd(23) = ', b)
end.
