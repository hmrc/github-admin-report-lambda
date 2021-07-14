package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func main() {
	lambda.Start(HandleLambdaEvent)
}

func HandleLambdaEvent() error {
	session := session.Must(session.NewSession())
	return runReport(&report{
		executor:        &command{},
		parameterGetter: &ssmService{},
		uploader:        &s3Uploader{},
	},
		session)
}

func runReport(r *report, session *session.Session) error {
	if err := r.setup(session); err != nil {
		return fmt.Errorf("setup error: %v", err)
	}

	dryRun, _ := strconv.ParseBool(os.Getenv("GHTOOL_DRY_RUN"))

	if err := r.generate(dryRun); err != nil {
		return fmt.Errorf("generate error: %v", err)
	}

	if dryRun {
		return nil
	}

	if err := r.store(session, "report.csv"); err != nil {
		return fmt.Errorf("store error: %v", err)
	}

	return nil
}

type report struct {
	executor        executor
	parameterGetter parameterGetter
	uploader        uploader
	bucketName      string
}

func (r *report) setup(session *session.Session) error {
	r.bucketName = os.Getenv("BUCKET_NAME")
	if r.bucketName == "" {
		return errors.New("bucket name not set")
	}

	ssmPath := os.Getenv("TOKEN_PATH")
	token, err := r.parameterGetter.getParameter(
		session,
		&ssm.GetParameterInput{
			Name:           aws.String(ssmPath),
			WithDecryption: aws.Bool(true),
		})
	if err != nil {
		return fmt.Errorf("get SSM param failed %v", err)
	}

	os.Setenv("GHTOOL_TOKEN", *token.Parameter.Value)

	return nil
}

func (r report) generate(dryRun bool) error {
	args := []string{"report", fmt.Sprintf("--dry-run=%t", dryRun)}
	output, err := r.executor.run("/github-admin-tool", args...)
	if err != nil {
		return fmt.Errorf("failed to run, got: %w, output: %s", err, output)
	}
	log.Printf("Output was %s", output)

	return nil
}

func (r report) store(session *session.Session, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", filename, err)
	}
	t := time.Now()
	objectName := fmt.Sprintf("%s-%s", filename, t.Format(time.RFC3339))

	result, err := r.uploader.upload(session, &s3manager.UploadInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(objectName),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}

	log.SetFlags(0)
	log.Printf("file uploaded to %v", result.Location)

	return nil
}

type uploader interface {
	upload(*session.Session, *s3manager.UploadInput) (*s3manager.UploadOutput, error)
}

type s3Uploader struct{}

func (s s3Uploader) upload(session *session.Session, artefact *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
	return s3manager.NewUploader(session).Upload(artefact)
}

type parameterGetter interface {
	getParameter(*session.Session, *ssm.GetParameterInput) (*ssm.GetParameterOutput, error)
}

type ssmService struct{}

func (s ssmService) getParameter(session *session.Session, input *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	return ssm.New(session).GetParameter(input)
}

type executor interface {
	run(string, ...string) ([]byte, error)
}

type command struct{}

func (c command) run(command string, args ...string) (outout []byte, err error) {
	return exec.Command(command, args...).CombinedOutput()
}
