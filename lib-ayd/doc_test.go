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
		logger.Healthy("target is healthy!", nil)
	} else {
		logger.Failure("target is down", nil)
	}
}

func Example_alertPlugin() {
	args, err := ayd.ParseAlertPluginArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger := ayd.NewLogger(args.AlertURL).StartTimer()

	// send alert to somewhere

	logger.Healthy("alert sent", nil)
}

func Example_apiClient() {
	aydURL, _ := url.Parse("http://localhost:9000")

	// fetch status from Ayd server
	report, err := ayd.Fetch(aydURL)
	if err != nil {
		panic(err)
	}

	for target, status := range report.ProbeHistory {
		// show target name
		fmt.Printf("# %s\n", target)

		// show status history
		for _, x := range status.Records {
			fmt.Println(x.Status)
		}
	}
}
