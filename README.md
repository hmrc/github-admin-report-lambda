# github-admin-tool-lambda

This lambda runs the github-admin-tool in report mode.  The report will be generated and uploaded to a AWS S3 bucket.

It creates an image with the tool from [here]("https://github.com/hmrc/github-admin-tool").

Currently there is a manual process to push the image up to sandbox or prod.

## License

This code is open source software licensed under the [Apache 2.0 License]("http://www.apache.org/licenses/LICENSE-2.0.html").

## Setup

The Lambda requires the following to be setup:

* A secure string SSM parameter to be setup containing the github token.  The name of this token must be added to the ENV vars.
* A S3 bucket to store the report.  The name of this bucket must be added to the ENV vars.

The Lambda will temporarily store the report (in JSON format) before uploading to S3 bucket.  Lambdas can only store at `/tmp` so the filepath ENV var needs to be set to reflect this.  You can also run reports in CSV format by changing `GHTOOL_FILE_TYPE=csv`.

## Environment variables

The following ENV vars can be passed to the Lambda.

```bash
BUCKET_NAME=bucket-name-where-report-to-be-stored
GHTOOL_DRY_RUN=true-or-false
GHTOOL_FILE_PATH=/tmp/some-filename.json
GHTOOL_FILE_TYPE=json
GHTOOL_ORG=github-org-name
TOKEN_PATH=name-of-ssm-param-to-be-called
```

Optional ENV vars:

```bash
GHTOOL_TEAM="This can be set to team name to show access level to repository report"
```

## Develop

The lambda can be built and run locally by (this will ask for an MFA token):

```bash
make clean-build-run
```

## Test

To format and run the go tests:

```bash
make test
```

## Push

To tag and push the image to the sandbox account github-admin-report ECR:

```bash
make push
```

## Push to prod

To tag and push the image to the production account github-admin-report ECR:

```bash
make push-prod
```

## CI/CD pipeline

### Where can I find a CI/CD pipeline for this code base?

- PR build job - None yet
- [Deployment pipeline](https://eu-west-2.console.aws.amazon.com/codesuite/codepipeline/pipelines/github-admin-report/view?region=eu-west-2)

### How is the CI/CD pipeline configured?

- No PR build job is configured yet
- Codepipeline pipeline config for deployment can be found in [platsec-terraform repo](https://github.com/hmrc/platsec-terraform/blob/main/components/github_admin_report/main.tf)
