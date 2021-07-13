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

type testReport struct {
	setupFail    bool
	generateFail bool
	storeFail    bool
}

func (r testReport) setup(*session.Session) error {
	if r.setupFail {
		return errors.New("fail") // nolint // only mock error for test
	}

	return nil
}

func (r testReport) generate(bool) error {
	if r.generateFail {
		return errors.New("fail") // nolint // only mock error for test
	}

	return nil
}

func (r testReport) store(*session.Session, string) error {
	if r.storeFail {
		return errors.New("fail") // nolint // only mock error for test
	}

	return nil
}

func Test_runReport(t *testing.T) {
	type args struct {
		s *session.Session
		r reporter
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
				r: testReport{},
			},
		},
		{
			name: "runReport set failure",
			args: args{
				r: testReport{
					setupFail: true,
				},
			},
			wantErr:    true,
			wantErrMsg: "Setup error: fail",
		},
		{
			name: "runReport run failure",
			args: args{
				r: testReport{
					generateFail: true,
				},
			},
			wantErr:    true,
			wantErrMsg: "Run error: fail",
		},
		{
			name: "runReport store failure",
			args: args{
				r: testReport{
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

func TestReport_generate(t *testing.T) {
	type args struct {
		dryRun bool
	}
	tests := []struct {
		name       string
		r          report
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
			r := report{}
			err := r.generate(tt.args.dryRun)
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

type testSSMService struct {
	getParameterFail bool
}

func (g testSSMService) getParameter(session *session.Session, input *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	if g.getParameterFail {
		return nil, errors.New("fail") // nolint // only mock error for test
	}

	output := new(ssm.GetParameterOutput)
	output.Parameter = &ssm.Parameter{Value: aws.String("param-value")}
	return output, nil
}

func TestReport_store(t *testing.T) {
	defer os.Setenv("BUCKET_NAME", os.Getenv("BUCKET_NAME"))
	defaultSession := session.Must(session.NewSession())
	type args struct {
		session  *session.Session
		filename string
	}
	tests := []struct {
		name           string
		runner         report
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
			runner:         report{uploader: &testUploader{uploadFail: true}},
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
			runner:         report{uploader: &testUploader{uploadFail: false}},
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
			if err := tt.runner.store(tt.args.session, filename); (err != nil) != tt.wantErr {
				t.Errorf("report.store() error = %v, wantErr %v", err, tt.wantErr)
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
	defer os.Setenv("GHTOOL_TOKEN", os.Getenv("GHTOOL_TOKEN"))
	defaultSession := session.Must(session.NewSession())
	type fields struct {
		uploader        uploader
		parameterGetter parameterGetter
	}
	type args struct {
		session *session.Session
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Setup success",
			fields: fields{
				parameterGetter: testSSMService{
					getParameterFail: false,
				},
			},
			args: args{
				session: defaultSession,
			},
			wantErr: false,
		},
		{
			name: "Setup failed",
			fields: fields{
				parameterGetter: testSSMService{
					getParameterFail: true,
				},
			},
			args: args{
				session: defaultSession,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := report{
				uploader:        tt.fields.uploader,
				parameterGetter: tt.fields.parameterGetter,
			}
			if err := r.setup(tt.args.session); (err != nil) != tt.wantErr {
				t.Errorf("report.setup() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
