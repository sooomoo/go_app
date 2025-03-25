package errs

import "fmt"

type ErrorOfCustom struct {
	MyField string
}

func (e ErrorOfCustom) Error() string {
	return fmt.Sprintf("description for this err. %s", e.MyField)
}
