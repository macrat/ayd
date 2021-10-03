package ayd_test

import (
	"errors"
	"fmt"

	"github.com/macrat/ayd/lib-ayd"
)

func ExampleAydError() {
	_, err := ayd.ParseRecord("this is invalid record")

	// You can check the error is about what via errors.Is function.
	fmt.Println("this is invalid record error:", errors.Is(err, ayd.ErrInvalidRecord))

	// Output:
	// this is invalid record error: true
}
