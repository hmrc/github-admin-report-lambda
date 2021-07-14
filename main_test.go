package main

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func Test_runReport(t *testing.T) {
	f, _ := os.Create("report.csv")
	defer func() {
		f.Close()
		os.Remove("report.csv")

		if _, exists := os.LookupEnv("BUCKET_NAME"); exists {
			os.Setenv("BUCKET_NAME", os.Getenv("BUCKET_NAME"))
		}

		if _, exists := os.LookupEnv("GHTOOL_DRY_RUN"); exists {
			os.Setenv("GHTOOL_DRY_RUN", os.Getenv("GHTOOL_DRY_RUN"))
		}
	}()
	os.Setenv("BUCKET_NAME", "some-bucket-name")
	os.Setenv("GHTOOL_DRY_RUN", "false")

	type args struct {
		s *session.Session
		r report
	}
	tests := []struct {
		name          string
		args          args
		wantErr       bool
		wantErrMsg    string
		setDryRunTrue bool
	}{
		{
			name: "runReport success",
			args: args{
				r: report{
					executor:        testExecutor{},
					parameterGetter: testParameterGetter{},
					uploader:        testUploader{},
				},
			},
		},
		{
			name: "runReport set failure",
			args: args{
				r: report{
					executor:        testExecutor{},
					parameterGetter: testParameterGetter{getParameterFail: true},
					uploader:        testUploader{},
				},
			},
			wantErr:    true,
			wantErrMsg: "Setup error: Get SSM param failed fail",
		},
		{
			name: "runReport run failure",
			args: args{
				r: report{
					executor:        testExecutor{runFail: true},
					parameterGetter: testParameterGetter{},
					uploader:        testUploader{},
				},
			},
			wantErr:    true,
			wantErrMsg: "Run error: failed to run, got: fail, output: nothing",
		},
		{
			name: "runReport dry run exit",
			args: args{
				r: report{
					executor:        testExecutor{},
					parameterGetter: testParameterGetter{},
					uploader:        testUploader{},
				},
			},
			wantErr:       false,
			setDryRunTrue: true,
		},
		{
			name: "runReport store failure",
			args: args{
				r: report{
					executor:        testExecutor{},
					parameterGetter: testParameterGetter{},
					uploader:        testUploader{uploadFail: true},
				},
			},
			wantErr:    true,
			wantErrMsg: "Store error: failed to upload file, fail",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setDryRunTrue {
				os.Setenv("GHTOOL_DRY_RUN", "true")
			}

			err := runReport(&tt.args.r, tt.args.s)

			if (err != nil) != tt.wantErr {
				t.Errorf("runReport error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrMsg != "" && tt.wantErrMsg != err.Error() {
				t.Errorf("runReport error = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
			}

			os.Setenv("GHTOOL_DRY_RUN", "false")
		})
	}
}

func TestReport_generate(t *testing.T) {
	type args struct {
		dryRun bool
	}
	tests := []struct {
		name       string
		args       args
		r          report
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "Generate throws error",
			r:          report{executor: testExecutor{runFail: true}},
			wantErr:    true,
			wantErrMsg: "failed to run, got: fail, output: nothing",
			args: args{
				dryRun: false,
			},
		},
		{
			name:       "Generate report successfully",
			r:          report{executor: testExecutor{}},
			wantErr:    false,
			wantErrMsg: "",
			args: args{
				dryRun: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.r.generate(tt.args.dryRun)
			if (err != nil) != tt.wantErr {
				t.Errorf("report.generate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrMsg != "" && tt.wantErrMsg != err.Error() {
				t.Errorf("report.generate() error = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
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

type testParameterGetter struct {
	getParameterFail bool
}

func (g testParameterGetter) getParameter(session *session.Session, input *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	if g.getParameterFail {
		return nil, errors.New("fail") // nolint // only mock error for test
	}

	output := new(ssm.GetParameterOutput)
	output.Parameter = &ssm.Parameter{Value: aws.String("param-value")}
	return output, nil
}

type testExecutor struct {
	runFail bool
}

func (c testExecutor) run(command string, args ...string) (outout []byte, err error) {
	if c.runFail {
		return []byte("nothing"), errors.New("fail") // nolint // only mock error for test
	}

	return []byte("success"), nil
}

func TestReport_store(t *testing.T) {
	defaultSession := session.Must(session.NewSession())
	type args struct {
		session  *session.Session
		filename string
	}
	tests := []struct {
		name           string
		r              report
		args           args
		wantErr        bool
		wantErrMsg     string
		setEnvVar      bool
		setEnvVarValue string
	}{
		{
			name:           "Store throws filed to open file",
			r:              report{uploader: testUploader{}},
			setEnvVar:      true,
			wantErr:        true,
			wantErrMsg:     "failed to open file \"\", open : no such file or directory",
			setEnvVarValue: "some-bucket-id",
			args: args{
				session: defaultSession,
			},
		},
		{
			name:           "Fail to upload file",
			r:              report{uploader: &testUploader{uploadFail: true}},
			setEnvVar:      true,
			wantErr:        true,
			wantErrMsg:     "failed to upload file, fail",
			setEnvVarValue: "some-bucket-id",
			args: args{
				session:  defaultSession,
				filename: "hello.txt",
			},
		},
		{
			name:           "Successfully upload hello.txt",
			r:              report{uploader: &testUploader{uploadFail: false}},
			setEnvVar:      true,
			wantErr:        false,
			setEnvVarValue: "some-bucket-id",
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
			err := tt.r.store(tt.args.session, filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("report.store() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrMsg != "" && tt.wantErrMsg != err.Error() {
				t.Errorf("runReport error = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
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

func TestReport_setup(t *testing.T) {
	defer func() {
		if _, exists := os.LookupEnv("BUCKET_NAME"); exists {
			os.Setenv("BUCKET_NAME", os.Getenv("BUCKET_NAME"))
		}
	}()
	defaultSession := session.Must(session.NewSession())
	type args struct {
		session *session.Session
	}
	tests := []struct {
		name           string
		r              report
		args           args
		setEnvVar      bool
		setEnvVarValue string
		wantErr        bool
		wantErrMsg     string
	}{
		{
			name: "Setup success",
			r: report{
				parameterGetter: testParameterGetter{
					getParameterFail: false,
				},
			},
			args: args{
				session: defaultSession,
			},
			setEnvVar:      true,
			setEnvVarValue: "some-bucket-id",
			wantErr:        false,
		},
		{
			name: "Setup failed",
			r: report{
				parameterGetter: testParameterGetter{
					getParameterFail: true,
				},
			},
			args: args{
				session: defaultSession,
			},
			setEnvVar:      true,
			setEnvVarValue: "some-bucket-id",
			wantErr:        true,
			wantErrMsg:     "Get SSM param failed fail",
		},
		{
			name:       "Setup throws error bucket not set",
			r:          report{uploader: testUploader{}},
			wantErr:    true,
			wantErrMsg: "bucket name not set",
			args: args{
				session: defaultSession,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("BUCKET_NAME")
			if tt.setEnvVar {
				os.Setenv("BUCKET_NAME", tt.setEnvVarValue)
			}
			err := tt.r.setup(tt.args.session)
			if (err != nil) != tt.wantErr {
				t.Errorf("setup() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrMsg != "" && tt.wantErrMsg != err.Error() {
				t.Errorf("setup() error = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
			}
		})
	}
}
