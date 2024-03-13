package output

import "fmt"

var Verbosity bool

func Println(a ...any) (n int, err error) {
	if !Verbosity {
		return -1, nil
	}

	return fmt.Println(a...)
}
