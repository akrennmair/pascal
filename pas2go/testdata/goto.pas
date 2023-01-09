program test;

label 10;

var i : integer;

begin
    i := 0;
    10:
    i := i + 1;
    writeln('i = ', i);
    if i < 10 then
        goto 10
end.