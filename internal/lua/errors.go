package lua

import "fmt"

func newTypeError(message string) error {
	return fmt.Errorf("lua type error: %s", message)
}
