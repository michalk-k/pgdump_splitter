package output

import "fmt"

var Quiet bool

func Println(a ...any) (n int, err error) {
	if Quiet {
		return -1, nil
	}

	return fmt.Println(a...)
}
