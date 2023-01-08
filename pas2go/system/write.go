package system

import "fmt"

func Write(args ...any) {
	for _, arg := range args {
		if b, isByte := arg.(byte); isByte {
			fmt.Printf("%c", b)
		} else {
			fmt.Print(arg)
		}
	}
}

func Writeln(args ...any) {
	Write(args...)
	fmt.Println("")
}
