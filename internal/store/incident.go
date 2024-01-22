package store

import (
	"net"
	"regexp"
	"strconv"

	api "github.com/macrat/ayd/lib-ayd"
)

// newIncidens makes a new api.Incident from an api.Record.
func newIncident(r api.Record) *api.Incident {
	return &api.Incident{
		Target:   r.Target,
		Status:   r.Status,
		Message:  r.Message,
		StartsAt: r.Time,
	}
}

type byIncidentCaused []*api.Incident

func (xs byIncidentCaused) Len() int {
	return len(xs)
}

func (xs byIncidentCaused) Less(i, j int) bool {
	if xs[i].StartsAt.Equal(xs[j].StartsAt) {
		return xs[i].Target.String() < xs[j].Target.String()
	}
	return xs[i].StartsAt.Before(xs[j].StartsAt)
}

func (xs byIncidentCaused) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}

var (
	// addressPattern is a regexp pattern for detecting address from message.
	// The pattern is including:
	// {hostname}:{port} (e.g. example.com:443)
	// {IPv4}:{port} (e.g. 192.168.1.1:80)
	// {IPv6}:{port} (e.g. [2001:db8::1]:8080)
	addressPattern = regexp.MustCompile(`(?:[-_.0-9a-zA-Z]+|(?:\d{1,3}\.){3}\d{1,3}|\[[0-9a-fA-F:]+\]):(\d+)`)
)

// isSameIncidentMessage checks that two messages of incident are the same or not.
// If port number detected in the message and the number is larger than 1024, the difference between port numbers is ignored.
func isSameIncidentMessage(a, b string) bool {
	if a == b {
		return true
	}

	replaceFunc := func(s string) string {
		host, port, err := net.SplitHostPort(s)
		if err != nil {
			return s
		}
		p, err := strconv.Atoi(port)
		if err != nil {
			return s
		}
		if p < 1024 {
			return s
		}
		return host + ":{PORT_NUMBER_IS_MASKED}"
	}

	a = addressPattern.ReplaceAllStringFunc(a, replaceFunc)
	b = addressPattern.ReplaceAllStringFunc(b, replaceFunc)

	return a == b
}
