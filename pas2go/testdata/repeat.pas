program test;

var i : integer;

begin
    i := 0;
    repeat
        writeln('i = ', i);
        i := i + 1
    until i >= 10
end.