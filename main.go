package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"

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

func retry(maxAttempts int, f func() error) (int, error) {
	var err error
	var attempts int

	for i := 0; i < maxAttempts; i++ {
		attempts = attempts + 1

		err = f()
		if err == nil {
			break
		}
	}

	return attempts, err
}

func runReport(r *report, session *session.Session) error {
	if err := r.setup(session); err != nil {
		return fmt.Errorf("setup error: %v", err)
	}

	if _, err := retry(4, r.generate); err != nil {
		return fmt.Errorf("generate error: %v", err)
	}

	if err := r.store(session); err != nil {
		return fmt.Errorf("store error: %v", err)
	}

	return nil
}

type report struct {
	executor        executor
	parameterGetter parameterGetter
	uploader        uploader
	bucketName      string
	dryRun          bool
	filePath        string
	fileType        string
}

func (r *report) setup(session *session.Session) error {
	dryRun, err := strconv.ParseBool(os.Getenv("GHTOOL_DRY_RUN"))
	if err != nil {
		dryRun = true // safe fallback
	}
	r.dryRun = dryRun

	r.bucketName = os.Getenv("BUCKET_NAME")
	if r.bucketName == "" {
		return errors.New("bucket name not set")
	}

	r.filePath = os.Getenv("GHTOOL_FILE_PATH")
	if r.filePath == "" {
		return errors.New("file path not set")
	}

	r.fileType = os.Getenv("GHTOOL_FILE_TYPE")
	re := regexp.MustCompile(`^(csv|json)$`)
	if r.fileType == "" || !re.Match([]byte(r.fileType)) {
		return errors.New("file type not set to csv or json")
	}

	token, err := r.parameterGetter.getParameter(
		session,
		&ssm.GetParameterInput{
			Name:           aws.String(os.Getenv("TOKEN_PATH")),
			WithDecryption: aws.Bool(true),
		})
	if err != nil {
		return fmt.Errorf("get SSM param failed %v", err)
	}

	os.Setenv("GHTOOL_TOKEN", *token.Parameter.Value)

	return nil
}

func (r report) generate() error {
	args := []string{
		"report",
		fmt.Sprintf("--dry-run=%t", r.dryRun),
		fmt.Sprintf("--file-path=%s", r.filePath),
		fmt.Sprintf("--file-type=%s", r.fileType),
	}
	output, err := r.executor.run("/github-admin-tool", args...)
	if err != nil {
		return fmt.Errorf("failed to run, got: %w, output: %s", err, output)
	}
	log.Printf("Output was %s", output)

	return nil
}

func (r report) store(session *session.Session) error {
	if r.dryRun {
		return nil
	}

	f, err := os.Open(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", r.filePath, err)
	}

	result, err := r.uploader.upload(session, &s3manager.UploadInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(fmt.Sprintf("github_admin_report.%s", r.fileType)),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}

	log.Printf("file uploaded to %v", result.Location)

	return nil
}

type uploader interface {
	upload(*session.Session, *s3manager.UploadInput) (*s3manager.UploadOutput, error)
}

type s3Uploader struct{}

func (s *s3Uploader) upload(session *session.Session, artefact *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
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
