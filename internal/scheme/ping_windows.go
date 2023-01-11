//go:build windows
// +build windows

package scheme

import (
	"context"
)

func (p *simplePinger) startPingers(ctx context.Context) error {
	if err := p.v4.Start(ctx); err != nil {
		return err
	}
	return p.v6.Start(ctx)
}
