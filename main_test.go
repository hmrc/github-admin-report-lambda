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
					uploader:        &testUploader{},
				},
			},
		},
		{
			name: "runReport set failure",
			args: args{
				r: report{
					executor:        testExecutor{},
					parameterGetter: testParameterGetter{getParameterFail: true},
					uploader:        &testUploader{},
				},
			},
			wantErr:    true,
			wantErrMsg: "setup error: get SSM param failed fail",
		},
		{
			name: "runReport run failure",
			args: args{
				r: report{
					executor:        testExecutor{runFail: true},
					parameterGetter: testParameterGetter{},
					uploader:        &testUploader{},
				},
			},
			wantErr:    true,
			wantErrMsg: "generate error: failed to run, got: fail, output: nothing",
		},
		{
			name: "runReport dry run exit",
			args: args{
				r: report{
					executor:        testExecutor{},
					parameterGetter: testParameterGetter{},
					uploader:        &testUploader{},
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
					uploader:        &testUploader{uploadFail: true},
				},
			},
			wantErr:    true,
			wantErrMsg: "store error: failed to upload file, fail",
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
	tests := []struct {
		name       string
		r          report
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "Generate throws error",
			r: report{
				executor: testExecutor{runFail: true},
				dryRun:   false,
			},
			wantErr:    true,
			wantErrMsg: "failed to run, got: fail, output: nothing",
		},
		{
			name: "Generate report successfully",
			r: report{
				executor: testExecutor{},
				dryRun:   false,
			},
			wantErr:    false,
			wantErrMsg: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.r.generate()
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
	called     bool
}

func (u *testUploader) upload(session *session.Session, artefact *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
	u.called = true
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
		name             string
		r                report
		args             args
		wantUploadCalled bool
		wantErr          bool
		wantErrMsg       string
	}{
		{
			name:             "Store throws filed to open file",
			r:                report{uploader: &testUploader{}},
			wantUploadCalled: false,
			wantErr:          true,
			wantErrMsg:       "failed to open file \"\", open : no such file or directory",
			args: args{
				session: defaultSession,
			},
		},
		{
			name:             "Fail to upload file",
			r:                report{uploader: &testUploader{uploadFail: true}},
			wantUploadCalled: true,
			wantErr:          true,
			wantErrMsg:       "failed to upload file, fail",
			args: args{
				session:  defaultSession,
				filename: "hello.txt",
			},
		},
		{
			name:             "Successfully upload hello.txt",
			r:                report{uploader: &testUploader{uploadFail: false}},
			wantUploadCalled: true,
			wantErr:          false,
			args: args{
				session:  defaultSession,
				filename: "hello.txt",
			},
		},
		{
			name: "Do not store if running in dry run mode",
			r: report{
				uploader: &testUploader{uploadFail: false},
				dryRun:   true,
			},
			wantUploadCalled: false,
			wantErr:          false,
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
			gotUploader, ok := tt.r.uploader.(*testUploader)
			if !ok {
				t.Fatalf("cannot cast uploader to testUploader")
			}
			if gotUploader.called != tt.wantUploadCalled {
				if gotUploader.called {
					t.Errorf("uploader was called and it should not")
				} else {
					t.Errorf("uploader should not be called and it was")
				}
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
		if _, exists := os.LookupEnv("GHTOOL_DRY_RUN"); exists {
			os.Setenv("GHTOOL_DRY_RUN", os.Getenv("GHTOOL_DRY_RUN"))
		}
	}()
	defaultSession := session.Must(session.NewSession())
	type args struct {
		session *session.Session
	}
	tests := []struct {
		name             string
		r                report
		args             args
		setEnvVar        bool
		setEnvBucket     string
		setEnvDryRun     string
		wantReportDryRun bool
		wantErr          bool
		wantErrMsg       string
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
			setEnvVar:        true,
			setEnvBucket:     "some-bucket-id",
			setEnvDryRun:     "true",
			wantReportDryRun: true,
			wantErr:          false,
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
			setEnvVar:        true,
			setEnvBucket:     "some-bucket-id",
			setEnvDryRun:     "true",
			wantReportDryRun: true,
			wantErr:          true,
			wantErrMsg:       "get SSM param failed fail",
		},
		{
			name:             "Setup throws error bucket and dryRun not set",
			r:                report{parameterGetter: testParameterGetter{}},
			wantReportDryRun: true,
			wantErr:          true,
			wantErrMsg:       "bucket name not set",
			args: args{
				session: defaultSession,
			},
		},
		{
			name:             "Setup throws error bucket not set",
			r:                report{parameterGetter: testParameterGetter{}},
			wantReportDryRun: true,
			wantErr:          true,
			wantErrMsg:       "bucket name not set",
			args: args{
				session: defaultSession,
			},
		},
		{
			name:             "Setup throws error bucket name empty",
			r:                report{parameterGetter: testParameterGetter{}},
			setEnvVar:        true,
			setEnvBucket:     "",
			setEnvDryRun:     "true",
			wantReportDryRun: true,
			wantErr:          true,
			wantErrMsg:       "bucket name not set",
			args: args{
				session: defaultSession,
			},
		},
		{
			name:             "Success with no dry run settings",
			r:                report{parameterGetter: testParameterGetter{}},
			setEnvVar:        true,
			setEnvBucket:     "some-bucket-id",
			setEnvDryRun:     "",
			wantReportDryRun: true,
			wantErr:          false,
			args: args{
				session: defaultSession,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("BUCKET_NAME")
			if tt.setEnvVar {
				os.Setenv("BUCKET_NAME", tt.setEnvBucket)
			}
			os.Unsetenv("GHTOOL_DRY_RUN")
			if tt.setEnvVar {
				os.Setenv("GHTOOL_DRY_RUN", tt.setEnvBucket)
			}
			err := tt.r.setup(tt.args.session)
			if (err != nil) != tt.wantErr {
				t.Errorf("setup() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrMsg != "" && tt.wantErrMsg != err.Error() {
				t.Errorf("setup() error = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
			}
			if tt.r.dryRun != tt.wantReportDryRun {
				t.Errorf("report.dryRun = %v, wantReportDryRun %v", tt.r.dryRun, tt.wantReportDryRun)
			}
		})
	}
}
