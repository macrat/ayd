package ayd

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/macrat/ayd/internal/ayderr"
)

// ProbePluginArgs is arguments for probe plugin
type ProbePluginArgs struct {
	TargetURL *url.URL
}

// ParseProbePluginArgsFrom is parse arguments for probe plugin
func ParseProbePluginArgsFrom(args []string) (ProbePluginArgs, error) {
	if len(args) != 2 {
		return ProbePluginArgs{}, ayderr.New(ErrArgumentCount, nil, "invalid argument: should give just 1 argument")
	}

	target, err := url.Parse(args[1])
	if err != nil {
		return ProbePluginArgs{}, ayderr.New(ErrInvalidArgumentValue, err, "invalid target URL")
	}

	return ProbePluginArgs{target}, nil
}

// ParseProbePluginArgs is get arguments for probe plugin
//
// This function is shorthand of `ayd.ParseProbePluginArgs(os.Args)`.
func ParseProbePluginArgs() (ProbePluginArgs, error) {
	return ParseProbePluginArgsFrom(os.Args)
}

// AlertPluginArgs is arguments for alert plugin
type AlertPluginArgs struct {
	AlertURL  *url.URL
	CheckedAt time.Time
	Status    Status
	Latency   time.Duration
	TargetURL *url.URL
	Message   string
}

// ParseAlertPluginArgsFrom is parse arguments for alert plugin
func ParseAlertPluginArgsFrom(args []string) (AlertPluginArgs, error) {
	if len(args) != 7 {
		return AlertPluginArgs{}, ayderr.New(ErrArgumentCount, nil, "invalid argument: should give exactly 6 arguments")
	}

	alertURL, err := url.Parse(args[1])
	if err != nil {
		return AlertPluginArgs{}, ayderr.New(ErrInvalidArgumentValue, err, "invalid alert URL")
	}

	checkedAt, err := time.Parse(time.RFC3339, args[2])
	if err != nil {
		return AlertPluginArgs{}, ayderr.New(ErrInvalidArgumentValue, err, "invalid checked at timestamp")
	}

	status := ParseStatus(strings.ToUpper(args[3]))

	latency, err := strconv.ParseFloat(args[4], 64)
	if err != nil {
		return AlertPluginArgs{}, ayderr.New(ErrInvalidArgumentValue, err, "invalid latency")
	}

	targetURL, err := url.Parse(args[5])
	if err != nil {
		return AlertPluginArgs{}, ayderr.New(ErrInvalidArgumentValue, err, "invalid target URL")
	}

	return AlertPluginArgs{
		AlertURL:  alertURL,
		CheckedAt: checkedAt,
		Status:    status,
		Latency:   time.Duration(latency) * time.Millisecond,
		TargetURL: targetURL,
		Message:   args[6],
	}, nil
}

// ParseAlertPluginArgs is get arguments for alert plugin
//
// This function is shorthand of `ayd.ParseAlertPluginArgs(os.Args)`.
func ParseAlertPluginArgs() (AlertPluginArgs, error) {
	return ParseAlertPluginArgsFrom(os.Args)
}
