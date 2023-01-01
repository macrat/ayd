package ayd

import (
	"os"
	"time"

	"github.com/macrat/ayd/internal/ayderr"
)

// ProbePluginArgs is arguments for probe plugin
//
// Deprecated: since version 0.16. This struct will removed in future version.
// Please parse using ParseURL instead of this.
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
//
// Deprecated: since version 0.16. This struct will removed in future version.
// Please parse using ParseURL and ParseRecord instead of this.
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
	if len(args) != 3 {
		return AlertPluginArgs{}, ayderr.New(ErrArgumentCount, nil, "invalid argument: should give exactly 2 arguments")
	}

	alertURL, err := ParseURL(args[1])
	if err != nil {
		return AlertPluginArgs{}, ayderr.New(ErrInvalidArgumentValue, err, "invalid alert URL")
	}

	record, err := ParseRecord(args[2])
	if err != nil {
		return AlertPluginArgs{}, err
	}

	return AlertPluginArgs{
		AlertURL:  alertURL,
		Time:      record.Time,
		Status:    record.Status,
		Latency:   record.Latency,
		TargetURL: record.Target,
		Message:   record.Message,
		Extra:     record.Extra,
	}, nil
}

// ParseAlertPluginArgs is get arguments for alert plugin
//
// This function is shorthand of `ayd.ParseAlertPluginArgs(os.Args)`.
func ParseAlertPluginArgs() (AlertPluginArgs, error) {
	return ParseAlertPluginArgsFrom(os.Args)
}
