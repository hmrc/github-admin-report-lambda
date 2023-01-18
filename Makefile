DOCKER_LIST=$(shell docker ps -q)
GIT_HASH=$(shell git rev-parse HEAD)

DOCKER = docker run \
	--interactive \
	--rm \
	--volume "${PWD}:${PWD}" \
	--workdir "${PWD}"

.PHONY: go gofmt golangci-lint
go gofmt golangci-lint:
	@docker build \
		--tag $@ \
		--build-arg "user_id=$(shell id -u)" \
		--build-arg "group_id=$(shell id -g)" \
		--build-arg "home=${HOME}" \
		--build-arg "workdir=${PWD}" \
		--target $@ . \
		--file Dockerfile.tests \
		>/dev/null

.PHONY: fmt
fmt: gofmt
	@$(DOCKER) gofmt -l -s -w .

.PHONY: fmt-check
fmt-check: gofmt
	@$(DOCKER) gofmt -l -s -d .

.PHONY: test
test: go
	@$(DOCKER) go test -cover ./...

.PHONY: lint
lint: golangci-lint
	@$(DOCKER) golangci-lint run --fix --issues-exit-code 0

.PHONY: lint-check
lint-check: golangci-lint
	@$(DOCKER) golangci-lint run --color always

.PHONY: build-image
build-image:
	go mod tidy
	docker build -t container-release:local --target distro .
.PHONY: build-image

.PHONY: build-rie
build-rie:
	docker build -t github-admin-report-rie --target rie .

.PHONY: clean
clean:
	@docker kill $(DOCKER_LIST) || true
	@docker rm github-admin-report-rie || true
	@docker rmi github-admin-report-rie || true

.PHONY: clean-build-run
clean-build-run: clean build-image local_run

.PHONY: local_run
local_run: build-rie
	aws-profile -p platsec-sandbox-RoleSandboxAccess docker run \
		--detach \
		--publish 9000:8080 \
		--name github-admin-report-rie \
		--env AWS_ACCESS_KEY_ID \
		--env AWS_SECRET_ACCESS_KEY \
		--env AWS_SESSION_TOKEN \
		--env AWS_REGION \
		--env BUCKET_NAME \
		--env GHTOOL_DRY_RUN \
		--env GHTOOL_FILE_PATH \
		--env GHTOOL_FILE_TYPE \
		--env GHTOOL_ORG \
		--env GHTOOL_TEAM \
		--env TOKEN_PATH \
		github-admin-report-rie:latest \
		/main
	curl -XPOST \
		"http://localhost:9000/2015-03-31/functions/function/invocations"  -d '{}' | jq
	docker logs github-admin-report-rie

.PHONY: show_test_cover
show_test_cover:
	@$(DOCKER) go test -coverprofile /tmp/cover.out
	@$(DOCKER) go tool cover -func=/tmp/cover.out

.PHONY: test_pr_check
test_pr_check:
	$(MAKE) fmt-check
	$(MAKE) lint-check
	$(MAKE) test

.PHONY: push
push:
	# docker login -u AWS -p <password> <aws_account_id>.dkr.ecr.<region>.amazonaws.com
	# aws ecr get-login-password --region eu-west-2 | docker login --username AWS --password-stdin 304923144821.dkr.ecr.eu-west-2.amazonaws.com
	docker tag github-admin-report 304923144821.dkr.ecr.eu-west-2.amazonaws.com/github-admin-report:$(GIT_HASH)
	docker push 304923144821.dkr.ecr.eu-west-2.amazonaws.com/github-admin-report:$(GIT_HASH)

.PHONY: push-prod
push-prod:
	# docker login -u AWS -p <password> <aws_account_id>.dkr.ecr.<region>.amazonaws.com
	# aws ecr get-login-password --region eu-west-2 | docker login --username AWS --password-stdin 324599906584.dkr.ecr.eu-west-2.amazonaws.com
	docker tag github-admin-report 324599906584.dkr.ecr.eu-west-2.amazonaws.com/github-admin-report:$(GIT_HASH)
	docker push 324599906584.dkr.ecr.eu-west-2.amazonaws.com/github-admin-report:$(GIT_HASH)
