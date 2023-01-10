program ftoc;

var fgr, cgr : real;
	thirtytwo : integer;
	five : integer;
	nine : integer;

begin
	thirtytwo := 32;
	five := 5;
	nine := 9;
	write('Degree F: ');
	readln(fgr);
	cgr := (fgr-thirtytwo)*five/nine;
	writeln('Degree C: ', cgr:7:2)
end.
