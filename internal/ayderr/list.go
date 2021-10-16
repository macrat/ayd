package ayderr

import (
	"fmt"
	"strings"
)

// List is a list of errors.
type List struct {
	// What is the error that describes what kind of errors is this.
	What error

	// Children is the detail errors in this error list.
	Children []error
}

// Error implements error interface.
func (l List) Error() string {
	ss := make([]string, 0, len(l.Children)+1)
	ss = append(ss, l.What.Error()+":")

	for _, e := range l.Children {
		for _, s := range strings.Split(e.Error(), "\n") {
			ss = append(ss, "  "+s)
		}
	}

	return strings.Join(ss, "\n")
}

// Unwrap implement for errors.Unwrap.
// This function returns What member.
func (l List) Unwrap() error {
	return l.What
}

func (l List) Is(err error) bool {
	if l.What == err {
		return true
	}
	for _, e := range l.Children {
		if e == err {
			return true
		}
	}
	return false
}

// ListBuilder is the List builder.
type ListBuilder struct {
	What     error
	Children []error
}

// Push appends a error as a child.
func (lb *ListBuilder) Push(err ...error) {
	lb.Children = append(lb.Children, err...)
}

// Pushf calls fmt.Errorf and then push as a child of this list.
func (lb *ListBuilder) Pushf(format string, values ...interface{}) {
	lb.Push(fmt.Errorf(format, values...))
}

// Build creates List if it has any child.
// It returns nil if there is no child.
func (lb *ListBuilder) Build() error {
	if len(lb.Children) == 0 {
		return nil
	}

	return List{
		What:     lb.What,
		Children: lb.Children,
	}
}
