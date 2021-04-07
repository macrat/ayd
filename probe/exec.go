package probe

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

func ExecuteProbe(u *url.URL) Result {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	if u.Fragment != "" {
		cmd = exec.CommandContext(ctx, u.Path, u.Fragment)
	} else {
		cmd = exec.CommandContext(ctx, u.Path)
	}

	for k, v := range u.Query() {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, strings.Join(v, ",")))
	}

	st := time.Now()
	stdout, err := cmd.CombinedOutput()
	d := time.Now().Sub(st)

	status := STATUS_OK
	message := string(stdout)

	if err != nil {
		status = STATUS_FAIL
		if message == "" {
			message = err.Error()
		}
	}

	return Result{
		CheckedAt: st,
		Target:    u,
		Status:    status,
		Message:   message,
		Latency:   d,
	}
}
