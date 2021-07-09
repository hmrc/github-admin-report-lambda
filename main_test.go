package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
)

func Test_isDryRun(t *testing.T) {
	defer os.Setenv("GHTOOL_DRY_RUN", os.Getenv("GHTOOL_DRY_RUN"))

	tests := []struct {
		name           string
		want           bool
		setEnvVar      bool
		setEnvVarValue string
	}{
		{
			name:      "isDryRun returns true with nothing set",
			want:      true,
			setEnvVar: false,
		},
		{
			name:           "isDryRun returns true with empty string",
			want:           true,
			setEnvVar:      true,
			setEnvVarValue: "",
		},
		{
			name:           "isDryRun returns false with false string",
			want:           false,
			setEnvVar:      true,
			setEnvVarValue: "false",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("GHTOOL_DRY_RUN")
			if tt.setEnvVar {
				os.Setenv("GHTOOL_DRY_RUN", tt.setEnvVarValue)
			}
			got := isDryRun()
			if got != tt.want {
				t.Errorf("isDryRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

type TestRunner struct {
	setupFail bool
	runFail   bool
	storeFail bool
}

func (r TestRunner) Setup(*session.Session) error {
	if r.setupFail {
		return errors.New("fail") // nolint // only mock error for test
	}

	return nil
}

func (r TestRunner) Run(bool) error {
	if r.runFail {
		return errors.New("fail") // nolint // only mock error for test
	}

	return nil
}

func (r TestRunner) Store(*session.Session) error {
	if r.storeFail {
		return errors.New("fail") // nolint // only mock error for test
	}

	return nil
}

func Test_runReport(t *testing.T) {
	type args struct {
		r Runner
	}
	tests := []struct {
		name          string
		args          args
		wantLogOutput string
	}{
		{
			name: "runReport success",
			args: args{
				TestRunner{},
			},
		},
		{
			name: "runReport set failure",
			args: args{
				TestRunner{
					setupFail: true,
				},
			},
			wantLogOutput: "Setup error: fail",
		},
		{
			name: "runReport run failure",
			args: args{
				TestRunner{
					runFail: true,
				},
			},
			wantLogOutput: "Run error: fail",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			log.SetOutput(&buf)

			defer func() { log.SetOutput(os.Stderr) }()

			runReport(tt.args.r)

			if tt.wantLogOutput != strings.TrimSpace(buf.String()) {
				t.Errorf("runReport() log = \n\n%v, wantLogOutput \n\n%v", buf.String(), tt.wantLogOutput)
			}
		})
	}
}

// This is just for coverage!!
func TestHandleLambdaEvent(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "HandleLambdaEvent",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleLambdaEvent()
		})
	}
}

func TestRealRunner_Run(t *testing.T) {
	type args struct {
		dryRun bool
	}
	tests := []struct {
		name       string
		r          RealRunner
		args       args
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "Run throws error",
			wantErr:    true,
			wantErrMsg: "",
			args:       args{dryRun: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := RealRunner{}
			err := r.Run(tt.args.dryRun)
			if (err != nil) != tt.wantErr {
				t.Errorf("RealRunner.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrMsg != "" && tt.wantErrMsg != err.Error() {
				t.Errorf("RealRunner.Run() error = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestRealRunner_Store(t *testing.T) {
	defer os.Setenv("GHTOOL_BUCKET_NAME", os.Getenv("GHTOOL_BUCKET_NAME"))
	defaultSession := session.Must(session.NewSession())
	type args struct {
		session *session.Session
	}
	tests := []struct {
		name           string
		r              RealRunner
		args           args
		wantErr        bool
		setEnvVar      bool
		setEnvVarValue string
	}{
		{
			name:           "Store throws error",
			wantErr:        true,
			setEnvVarValue: "",
		},
		{
			name:           "Store throws error past bucket name",
			setEnvVar:      true,
			wantErr:        true,
			setEnvVarValue: "some-bucket-id",
			args: args{
				session: defaultSession,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("GHTOOL_BUCKET_NAME")
			if tt.setEnvVar {
				os.Setenv("GHTOOL_BUCKET_NAME", tt.setEnvVarValue)
			}
			r := RealRunner{}
			if err := r.Store(tt.args.session); (err != nil) != tt.wantErr {
				t.Errorf("RealRunner.Store() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
