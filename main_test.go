package main_test

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"

	"github.com/macrat/ayd"
)

func MakeTestCommand(t testing.TB, taskArgs []string) (*main.AydCommand, *bytes.Buffer) {
	t.Helper()

	tasks, err := main.ParseArgs(taskArgs)
	if err != nil {
		t.Fatalf("failed to parse tasks: %s", err)
	}

	buf := bytes.NewBuffer([]byte{})

	return &main.AydCommand{
		OutStream: buf,
		ErrStream: buf,

		ListenPort: 9000,

		Tasks: tasks,
	}, buf
}

func TestAydCommand_ParseArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Args     []string
		Pattern  string
		ExitCode int
		Extra    func(*testing.T, main.AydCommand)
	}{
		{
			Args:     []string{"ayd"},
			Pattern:  `^Ayd -- Easy status monitoring tool`,
			ExitCode: 2,
		},
		{
			Args:     []string{"ayd", "-f", "-"},
			Pattern:  `^Ayd -- Easy status monitoring tool`,
			ExitCode: 2,
		},
		{
			Args:     []string{"ayd", "--no-such-option", "dummy:"},
			Pattern:  "^unknown flag: --no-such-option\n\nPlease see `ayd -h` for more information\\.\n$",
			ExitCode: 2,
		},
		{
			Args:     []string{"ayd", "-v", "-1", "-p", "1234", "dummy:"},
			Pattern:  `^$`,
			ExitCode: 0,
		},
		{
			Args:     []string{"ayd", "-h", "-c", "somewhere", "dummy:"},
			Pattern:  `^$`,
			ExitCode: 0,
		},
		{
			Args:     []string{"ayd", "-1", "-p", "1234", "dummy:"},
			Pattern:  "warning: port option will ignored in the oneshot mode\\.\n",
			ExitCode: 0,
		},
		{
			Args:     []string{"ayd", "-1", "-u", "foo:bar", "dummy:"},
			Pattern:  "warning: user option will ignored in the oneshot mode\\.\n",
			ExitCode: 0,
		},
		{
			Args:     []string{"ayd", "-1", "-c", "./path/to/cert", "dummy:"},
			Pattern:  "warning: ssl cert and key options will ignored in the oneshot mode\\.\n",
			ExitCode: 0,
		},
		{
			Args:     []string{"ayd", "-1", "-k", "./path/to/key", "dummy:"},
			Pattern:  "warning: ssl cert and key options will ignored in the oneshot mode\\.\n",
			ExitCode: 0,
		},
		{
			Args:     []string{"ayd", "-c", "./path/to/cert", "dummy:"},
			Pattern:  "invalid argument: the both of -c and -k option is required if you want to use HTTPS\\.\n",
			ExitCode: 2,
		},
		{
			Args:     []string{"ayd", "-k", "./path/to/key", "dummy:"},
			Pattern:  "invalid argument: the both of -c and -k option is required if you want to use HTTPS\\.\n",
			ExitCode: 2,
		},
		{
			Args:     []string{"ayd", "-f", "-", "dummy:"},
			ExitCode: 0,
			Extra: func(t *testing.T, cmd main.AydCommand) {
				if cmd.StorePath != "" {
					t.Errorf("expected StorePath is empty but got %#v", cmd.StorePath)
				}
			},
		},
		{
			Args:     []string{"ayd", "dummy:#A", "dummy:#B"},
			ExitCode: 0,
			Extra: func(t *testing.T, cmd main.AydCommand) {
				if len(cmd.Tasks) != 2 {
					t.Errorf("expected 2 tasks but got %d tasks", len(cmd.Tasks))
				}
			},
		},
		{
			Args:     []string{"ayd", "::invalid URL"},
			Pattern:  "invalid argument:\n  ::invalid URL: Not valid as schedule or target URL.\n\nPlease see `ayd -h` for more information\\.\n",
			ExitCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.Args), func(t *testing.T) {
			buf := bytes.NewBuffer([]byte{})
			cmd := main.AydCommand{
				OutStream: buf,
				ErrStream: buf,
			}

			exitCode := cmd.ParseArgs(tt.Args)

			if ok, _ := regexp.MatchString(tt.Pattern, buf.String()); !ok {
				t.Errorf("output expected to match with %q but not matched:\n%s", tt.Pattern, buf.String())
			}

			if exitCode != tt.ExitCode {
				t.Errorf("expected exit code is %d but got %d", tt.ExitCode, exitCode)
			}

			if tt.Extra != nil {
				tt.Extra(t, cmd)
			}
		})
	}
}

func TestAydCommand_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Args     []string
		Pattern  string
		ExitCode int
		Extra    func(*testing.T, main.AydCommand)
	}{
		{
			Args:     []string{"ayd"},
			Pattern:  `^Ayd -- Easy status monitoring tool`,
			ExitCode: 2,
		},
		{
			Args:     []string{"ayd", "-h"},
			Pattern:  `^Ayd -- Easy status monitoring tool`,
			ExitCode: 0,
		},
		{
			Args:     []string{"ayd", "-v"},
			Pattern:  `^Ayd version HEAD \(UNKNOWN\)` + "\n$",
			ExitCode: 0,
		},
		{
			Args:     []string{"ayd", "-f", "-", "-1", "-a", "dummy:#alert", "ping:localhost"},
			Pattern:  "^[-+:0-9TZ]+\tHEALTHY\t[0-9]+\\.[0-9]{3}\tping:localhost\tip=(127\\.0\\.0\\.1|::1) rtt\\(min/avg/max\\)=[0-9./]+ recv/sent=3/3\n$",
			ExitCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.Args), func(t *testing.T) {
			buf := bytes.NewBuffer([]byte{})
			cmd := main.AydCommand{
				OutStream: buf,
				ErrStream: buf,
			}

			exitCode := cmd.Run(tt.Args)

			if ok, _ := regexp.MatchString(tt.Pattern, buf.String()); !ok {
				t.Errorf("output expected to match with %q but not matched:\n%s", tt.Pattern, buf.String())
			}

			if exitCode != tt.ExitCode {
				t.Errorf("expected exit code is %d but got %d", tt.ExitCode, exitCode)
			}

			if tt.Extra != nil {
				tt.Extra(t, cmd)
			}
		})
	}
}
