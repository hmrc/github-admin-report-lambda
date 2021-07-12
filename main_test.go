package main

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

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

func (r TestRunner) Store(*session.Session, string) error {
	if r.storeFail {
		return errors.New("fail") // nolint // only mock error for test
	}

	return nil
}

func Test_runReport(t *testing.T) {
	type args struct {
		s *session.Session
		r Runner
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "runReport success",
			args: args{
				r: TestRunner{},
			},
		},
		{
			name: "runReport set failure",
			args: args{
				r: TestRunner{
					setupFail: true,
				},
			},
			wantErr:    true,
			wantErrMsg: "Setup error: fail",
		},
		{
			name: "runReport run failure",
			args: args{
				r: TestRunner{
					runFail: true,
				},
			},
			wantErr:    true,
			wantErrMsg: "Run error: fail",
		},
		{
			name: "runReport store failure",
			args: args{
				r: TestRunner{
					storeFail: true,
				},
			},
			wantErr:    true,
			wantErrMsg: "Store error: fail",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runReport(tt.args.s, tt.args.r)

			if (err != nil) != tt.wantErr {
				t.Errorf("runReport error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrMsg != "" && tt.wantErrMsg != err.Error() {
				t.Errorf("runReport error = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
			}
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

type testUploader struct {
	uploadFail bool
}

func (u testUploader) upload(session *session.Session, artefact *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
	if u.uploadFail {
		return nil, errors.New("fail") // nolint // only mock error for test
	}

	return &s3manager.UploadOutput{Location: "here"}, nil
}

func TestRealRunner_Store(t *testing.T) {
	defer os.Setenv("BUCKET_NAME", os.Getenv("BUCKET_NAME"))
	defaultSession := session.Must(session.NewSession())
	type args struct {
		session  *session.Session
		filename string
	}
	tests := []struct {
		name           string
		runner         RealRunner
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
		{
			name:           "Fail to upload file",
			setEnvVar:      true,
			wantErr:        true,
			setEnvVarValue: "some-bucket-id",
			runner:         RealRunner{uploader: &testUploader{uploadFail: true}},
			args: args{
				session:  defaultSession,
				filename: "hello.txt",
			},
		},
		{
			name:           "Successfully upload hello.txt",
			setEnvVar:      true,
			wantErr:        false,
			setEnvVarValue: "some-bucket-id",
			runner:         RealRunner{uploader: &testUploader{uploadFail: false}},
			args: args{
				session:  defaultSession,
				filename: "hello.txt",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.args.filename
			if tt.args.filename != "" {
				file, err := ioutil.TempFile(t.TempDir(), tt.args.filename)
				if err != nil {
					t.Fatalf("cannot create helper file: %v", err)
				}
				filename = file.Name()
			}
			os.Unsetenv("BUCKET_NAME")
			if tt.setEnvVar {
				os.Setenv("BUCKET_NAME", tt.setEnvVarValue)
			}
			if err := tt.runner.Store(tt.args.session, filename); (err != nil) != tt.wantErr {
				t.Errorf("RealRunner.Store() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleLambdaEvent(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "HandleLambdaEvent is successful",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := HandleLambdaEvent(); (err != nil) != tt.wantErr {
				t.Errorf("HandleLambdaEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
