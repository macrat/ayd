package probe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"time"
)

type PluginProbe struct {
	target  *url.URL
	command string
	env     []string
}

func NewPluginProbe(u *url.URL) (PluginProbe, error) {
	if u.Scheme == "ayd" || u.Scheme == "alert" {
		return PluginProbe{}, ErrUnsupportedScheme
	}

	p := PluginProbe{
		target:  u,
		command: "ayd-" + u.Scheme + "-probe",
		env:     os.Environ(),
	}

	if _, err := exec.LookPath(p.command); errors.Unwrap(err) == exec.ErrNotFound {
		return PluginProbe{}, ErrUnsupportedScheme
	} else if err != nil {
		return PluginProbe{}, err
	}

	p.env = append(
		p.env,
		fmt.Sprintf("ayd_target=%s", u),
	)

	return p, nil
}

func (p PluginProbe) Target() *url.URL {
	return p.target
}

func (p PluginProbe) Check(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	ExecuteExternalCommand(ctx, r, p.target, p.command, "", p.env)
}
