//go:build linux || darwin
// +build linux darwin

package scheme

import (
	"context"
)

func (p *autoPingerStruct) startPingers(ctx context.Context) error {
	if err := p.v4.Start(ctx); err == nil {
		return p.v6.Start(ctx)
	}

	p.v4.SetPrivileged(true)
	p.v4.SetPrivileged(false)

	if err := p.v4.Start(ctx); err != nil {
		return err
	}
	return p.v6.Start(ctx)
}
