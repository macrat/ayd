package query

type operator uint8

const (
	opIncludes     operator = 0b0000
	opEqual        operator = 0b0001
	opLessThan     operator = 0b0010
	opGreaterThan  operator = 0b0100
	opLessEqual    operator = 0b0011
	opGreaterEqual operator = 0b0101
	opNotEqual     operator = 0b1001
	opNotMask      operator = 0b0111
	opNotFlag      operator = 0b1000
)

func (o operator) String() string {
	switch o {
	case opIncludes:
		return "in"
	case opGreaterEqual:
		return ">="
	case opGreaterThan:
		return ">"
	case opEqual:
		return "="
	case opLessThan:
		return "<"
	case opLessEqual:
		return "<="
	case opNotEqual:
		return "!="
	default:
		panic("unreachable")
	}
}
