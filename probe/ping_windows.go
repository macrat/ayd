// +build windows

package probe

import (
	"github.com/go-ping/ping"
)

func getPinger(target string) (*ping.Pinger, error) {
	p, err := ping.NewPinger(target)
	if err != nil {
		return nil, err
	}
	p.SetPrivileged(true)
	return p, nil
}
