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
	return runReport(session, &RealRunner{
		uploader:   &realUploader{},
		SSMService: &realSSMService{},
	})
}

func runReport(session *session.Session, r Runner) error {
	if err := r.Setup(session); err != nil {
		return fmt.Errorf("Setup error: %v", err)
	}

	dryRun, _ := strconv.ParseBool(os.Getenv("GHTOOL_DRY_RUN"))

	if err := r.Run(dryRun); err != nil {
		return fmt.Errorf("Run error: %v", err)
	}

	if dryRun {
		return nil
	}

	if err := r.Store(session, "report.csv"); err != nil {
		return fmt.Errorf("Store error: %v", err)
	}

	return nil
}

type Runner interface {
	Setup(*session.Session) error
	Run(bool) error
	Store(*session.Session, string) error
}

type RealRunner struct {
	uploader   uploader
	SSMService SSMService
}

func (r RealRunner) Setup(session *session.Session) error {
	ssmPath := os.Getenv("TOKEN_PATH")
	token, err := r.SSMService.getParameter(
		session,
		&ssm.GetParameterInput{
			Name:           aws.String(ssmPath),
			WithDecryption: aws.Bool(true),
		})
	if err != nil {
		return fmt.Errorf("Get SSM param failed %v", err)
	}

	os.Setenv("GHTOOL_TOKEN", *token.Parameter.Value)

	return nil
}

func (r RealRunner) Run(dryRun bool) error {
	args := []string{"report", fmt.Sprintf("--dry-run=%t", dryRun)}
	output, err := exec.Command("/github-admin-tool", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run, got: %w, output: %s", err, output)
	}
	log.Printf("Output was %s", output)

	return nil
}

func (r RealRunner) Store(session *session.Session, filename string) error {
	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		return errors.New("bucket name not set")
	}

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", filename, err)
	}
	t := time.Now()
	objectName := fmt.Sprintf("%s-%s", filename, t.Format(time.RFC3339))

	result, err := r.uploader.upload(session, &s3manager.UploadInput{
		Bucket: aws.String(bucketName),
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

type realUploader struct{}

func (r realUploader) upload(session *session.Session, artefact *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
	return s3manager.NewUploader(session).Upload(artefact)
}

type SSMService interface {
	getParameter(*session.Session, *ssm.GetParameterInput) (*ssm.GetParameterOutput, error)
}

type realSSMService struct{}

func (r realSSMService) getParameter(session *session.Session, input *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	return ssm.New(session).GetParameter(input)
}
