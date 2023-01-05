program test;

var arr : array[1..5] of integer;
	i, j : integer;
	matrix : array[-5..+5, -3..+3] of real;

begin
	for i := 1 to 5 do
		arr[i] := i*i;

	for i := -5 to 5 do
		for j := 3 downto -3 do
		begin
			matrix[i, j] := 23.5
		end
end.
