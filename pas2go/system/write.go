package system

import "fmt"

func Write(args ...any) {
	for _, arg := range args {
		fmt.Print(arg)
	}
}

func Writeln(args ...any) {
	Write(args...)
	fmt.Println("")
}
