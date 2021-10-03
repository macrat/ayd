package ayd_test

import (
	"fmt"
	"net/url"
	"os"

	"github.com/macrat/ayd/lib-ayd"
)

func Example_probePlugin() {
	args, err := ayd.ParseProbePluginArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger := ayd.NewLogger(args.TargetURL).StartTimer()

	// check your target here
	ok := true

	if ok {
		logger.Healthy("target is healthy!")
	} else {
		logger.Failure("target is down")
	}
}

func Example_alertPlugin() {
	args, err := ayd.ParseAlertPluginArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger := ayd.NewLogger(args.AlertURL)

	// Fetch extra information from Ayd json API
	aydURL, err := url.Parse(os.Getenv("AYD_URL"))
	if err != nil {
		logger.Failure("failed to get Ayd URL")
		return
	}
	report, err := ayd.Fetch(aydURL)
	if err != nil {
		logger.Failure("failed to fetch status")
		return
	}
	_ = report.CurrentIncidents // check extra information about current incidents

	logger = logger.StartTimer() // start timer for measure time to send alert

	// send alert to somewhere

	logger.Healthy("alert sent")
}
