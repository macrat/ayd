package ayd

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/macrat/ayd/internal/ayderr"
)

// ProbePluginArgs is arguments for probe plugin
type ProbePluginArgs struct {
	TargetURL *URL
}

// ParseProbePluginArgsFrom is parse arguments for probe plugin
func ParseProbePluginArgsFrom(args []string) (ProbePluginArgs, error) {
	if len(args) != 2 {
		return ProbePluginArgs{}, ayderr.New(ErrArgumentCount, nil, "invalid argument: should give just 1 argument")
	}

	target, err := ParseURL(args[1])
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
	AlertURL  *URL
	Time      time.Time
	Status    Status
	Latency   time.Duration
	TargetURL *URL
	Message   string
	Extra     map[string]interface{}
}

// ParseAlertPluginArgsFrom is parse arguments for alert plugin
func ParseAlertPluginArgsFrom(args []string) (AlertPluginArgs, error) {
	if len(args) != 8 {
		return AlertPluginArgs{}, ayderr.New(ErrArgumentCount, nil, "invalid argument: should give exactly 7 arguments")
	}

	alertURL, err := ParseURL(args[1])
	if err != nil {
		return AlertPluginArgs{}, ayderr.New(ErrInvalidArgumentValue, err, "invalid alert URL")
	}

	timestamp, err := ParseTime(args[2])
	if err != nil {
		return AlertPluginArgs{}, ayderr.New(ErrInvalidArgumentValue, err, "invalid timestamp")
	}

	status := ParseStatus(strings.ToUpper(args[3]))

	latency, err := strconv.ParseFloat(args[4], 64)
	if err != nil {
		return AlertPluginArgs{}, ayderr.New(ErrInvalidArgumentValue, err, "invalid latency")
	}

	targetURL, err := ParseURL(args[5])
	if err != nil {
		return AlertPluginArgs{}, ayderr.New(ErrInvalidArgumentValue, err, "invalid target URL")
	}

	var extra map[string]interface{}
	if err := json.Unmarshal([]byte(args[7]), &extra); err != nil {
		return AlertPluginArgs{}, ayderr.New(ErrInvalidArgumentValue, err, "invalid extra values")
	}

	return AlertPluginArgs{
		AlertURL:  alertURL,
		Time:      timestamp,
		Status:    status,
		Latency:   time.Duration(latency) * time.Millisecond,
		TargetURL: targetURL,
		Message:   args[6],
		Extra:     extra,
	}, nil
}

// ParseAlertPluginArgs is get arguments for alert plugin
//
// This function is shorthand of `ayd.ParseAlertPluginArgs(os.Args)`.
func ParseAlertPluginArgs() (AlertPluginArgs, error) {
	return ParseAlertPluginArgsFrom(os.Args)
}
