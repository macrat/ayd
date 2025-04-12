package scheme

// SplitScheme splits scheme of URL.
//
// For example, "http-get" will splited to "http", '-', and "get".
func SplitScheme(scheme string) (subScheme string, separator rune, variant string) {
	for i, x := range scheme {
		if x == '-' || x == '+' {
			return scheme[:i], x, scheme[i+1:]
		}
	}
	return scheme, 0, ""
}

type Scheme interface {
	Prober
	Alerter
}
