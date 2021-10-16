package ayderr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/macrat/ayd/internal/ayderr"
)

func TestList_Is(t *testing.T) {
	errA := errors.New("error A")
	errB := errors.New("error B")
	errC := errors.New("error C")

	listABC := ayderr.List{errA, []error{errB, errC}}
	listAB := ayderr.List{errA, []error{errB}}

	tests := []struct {
		List  error
		Error error
		Want  bool
	}{
		{listABC, errA, true},
		{listABC, errB, true},
		{listABC, errC, true},
		{listAB, errA, true},
		{listAB, errB, true},
		{listAB, errC, false},
	}

	for i, tt := range tests {
		if actual := errors.Is(tt.List, tt.Error); actual != tt.Want {
			t.Errorf("%d: expected %v but got %v", i, tt.Want, actual)
		}
	}
}

func ExampleListBuilder() {
	// make a base error.
	ErrSomething := errors.New("something wrong")

	// prepare builder with base error.
	e := &ayderr.ListBuilder{What: ErrSomething}

	// e.Build() returns nil because builder has no child error yet.
	fmt.Println("--- before push errors ---")
	fmt.Println(e.Build())
	fmt.Println()

	// push errors as children.
	e.Push(errors.New("A is wrong"), errors.New("B is wrong"))

	// or generate error and push as a child.
	e.Pushf("%s is also wrong", "C")

	// e.Build() returns List now, because it has children.
	fmt.Println("--- after push errors ---")
	fmt.Println(e.Build())

	// OUTPUT:
	// --- before push errors ---
	// <nil>
	//
	// --- after push errors ---
	// something wrong:
	//   A is wrong
	//   B is wrong
	//   C is also wrong
}
