program test;

var pavcs : packed array [1..10] of 'g'..'p';
	i : integer;

begin
	for i := 1 to 10 do
		pavcs[i] := chr(i+ord('f'))
end.
