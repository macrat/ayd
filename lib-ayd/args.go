package ayd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
)

// ProbePluginArgs is arguments for probe plugin
type ProbePluginArgs struct {
	TargetURL *url.URL
}

// ParseProbePluginArgsFrom is parse arguments for probe plugin
func ParseProbePluginArgsFrom(args []string) (ProbePluginArgs, error) {
	if len(args) != 2 {
		return ProbePluginArgs{}, fmt.Errorf("%w: should give just 1 argument", ErrInvalidArgument)
	}

	target, err := url.Parse(args[1])
	if err != nil {
		return ProbePluginArgs{}, fmt.Errorf("%w: invalid target URL: %s", ErrInvalidArgument, err)
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
	TargetURL *url.URL
	Message   string
}

// ParseAlertPluginArgsFrom is parse arguments for alert plugin
func ParseAlertPluginArgsFrom(args []string) (AlertPluginArgs, error) {
	if len(args) != 6 {
		return AlertPluginArgs{}, fmt.Errorf("%w: should give just 5 argument", ErrInvalidArgument)
	}

	alertURL, err := url.Parse(args[1])
	if err != nil {
		return AlertPluginArgs{}, fmt.Errorf("%w: invalid alert URL: %s", ErrInvalidArgument, err)
	}

	checkedAt, err := time.Parse(time.RFC3339, args[2])
	if err != nil {
		return AlertPluginArgs{}, fmt.Errorf("%w: invalid checked time: %s", ErrInvalidArgument, err)
	}

	status := ParseStatus(strings.ToUpper(args[3]))

	targetURL, err := url.Parse(args[4])
	if err != nil {
		return AlertPluginArgs{}, fmt.Errorf("%w: invalid target URL: %s", ErrInvalidArgument, err)
	}

	return AlertPluginArgs{
		AlertURL:  alertURL,
		CheckedAt: checkedAt,
		Status:    status,
		TargetURL: targetURL,
		Message:   args[5],
	}, nil
}

// ParseAlertPluginArgs is get arguments for alert plugin
//
// This function is shorthand of `ayd.ParseAlertPluginArgs(os.Args)`.
func ParseAlertPluginArgs() (AlertPluginArgs, error) {
	return ParseAlertPluginArgsFrom(os.Args)
}
