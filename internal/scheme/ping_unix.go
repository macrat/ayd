//go:build linux || darwin
// +build linux darwin

package scheme

import (
	"context"

	"github.com/macrat/go-parallel-pinger"
)

func (p *simplePinger) startPingers(ctx context.Context) error {
	if err := p.v4.Start(ctx); err == nil {
		return p.v6.Start(ctx)
	}

	p.v4.SetPrivileged(!pinger.DEFAULT_PRIVILEGED)

	if err := p.v4.Start(ctx); err != nil {
		return err
	}
	return p.v6.Start(ctx)
}
