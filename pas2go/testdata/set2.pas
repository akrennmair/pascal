program test;

var s1, s2, s3 : set of integer;

begin
	s1 := [ 10, 5, 17 ];
	s2 := [ 5, 18, 20 ];
	s3 := s1 + s2;

	if not (20 in s3) then
		writeln('error: 20 not found in union!');

	if not (17 in s3) then
		writeln('error: 17 not found in union!');

	s3 := s1 - s2;

	if not (10 in s3) then 
		writeln('error: 10 not found in difference!');

	if 5 in s3 then 
		writeln('error: 5 found in difference!');
	
	s3 := s1 * s2;

	if not (5 in s3) then
		writeln('error: 5 not found in intersection!');

	if 18 in s3 then
		writeln('error: 18 found in intersection!')

end.
