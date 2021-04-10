package probe

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/macrat/ayd/store"
)

type ExecuteProbe struct {
	target *url.URL
	env    []string
}

func NewExecuteProbe(u *url.URL) ExecuteProbe {
	p := ExecuteProbe{}

	path := u.Opaque
	if u.Opaque == "" {
		path = u.Path
	}
	p.target = &url.URL{
		Scheme:   "exec",
		Opaque:   path,
		RawQuery: u.RawQuery,
		Fragment: u.Fragment,
	}

	p.env = os.Environ()
	for k, v := range u.Query() {
		p.env = append(p.env, fmt.Sprintf("%s=%s", k, v[len(v)-1]))
	}

	return p
}

func (p ExecuteProbe) Target() *url.URL {
	return p.target
}

func (p ExecuteProbe) Check() store.Record {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	if p.target.Fragment != "" {
		cmd = exec.CommandContext(ctx, p.target.Opaque, p.target.Fragment)
	} else {
		cmd = exec.CommandContext(ctx, p.target.Opaque)
	}

	cmd.Env = p.env

	st := time.Now()
	stdout, err := cmd.CombinedOutput()
	d := time.Now().Sub(st)

	status := store.STATUS_OK
	message := string(stdout)

	if err != nil {
		status = store.STATUS_FAIL
		if message == "" {
			message = err.Error()
		}
	}

	return store.Record{
		CheckedAt: st,
		Target:    p.target,
		Status:    status,
		Message:   message,
		Latency:   d,
	}
}
