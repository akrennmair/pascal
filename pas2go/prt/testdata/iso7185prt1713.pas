{

PRT test 1713: It is an error if the file is undefined immediately prior to
               any use of reset.

               ISO 7185 reference: 6.6.5.2

}

program iso7185prt1713(output);

var a: file of integer;

begin

   reset(a)

end.
