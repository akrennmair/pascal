program test;

var i : integer;

begin
	i := 0;
	inc(i);
	if i <> 1 then
		writeln('Error: i should be 1, is actually ', i);
	inc(i, 23);
	if i <> 24 then
		writeln('Error: i should be 24, is actually ', i);
	dec(i);
	if i <> 23 then
		writeln('Error: i should be 23, is actually ', i);
	dec(i, 13);
	if i <> 10 then
		writeln('Error: i should be 23, is actually ', i)
end.
