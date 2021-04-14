// +build !windows

package probe

import (
	"github.com/go-ping/ping"
)

func getPinger(target string) (*ping.Pinger, error) {
	return ping.NewPinger(target)
}
