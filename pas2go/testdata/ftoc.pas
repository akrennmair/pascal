program ftoc;

var fgr, cgr : real;

begin
	write('Degree F: ');
	readln(fgr);
	cgr := (fgr-32.0)*5.0/9.0;
	writeln('Degree C: ', cgr:7:2)
end.
