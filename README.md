# github-admin-tool-lambda

This lambda runs the github-admin-tool in report mode.

It creates an image with the tool from [here]("https://github.com/hmrc/github-admin-tool").

Currently there is a manual process to push the image up to sandbox or prod.

## License

This code is open source software licensed under the [Apache 2.0 License]("http://www.apache.org/licenses/LICENSE-2.0.html").

## Environment variables

The following ENV Vars can be passed to the Lambda.

```bash
TOKEN_PATH=name-of-ssm-param-to-be-called
GHTOOL_ORG=github-org-name
GHTOOL_DRY_RUN=true-or-false
BUCKET_NAME=bucket-name-where-report-to-be-stored
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
