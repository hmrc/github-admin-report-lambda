FROM golang:1.15-alpine3.14 as build

ARG GITHUB_ADMIN_TOOL_VERSION

# cache dependencies
WORKDIR /app

# cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# install github-admin-tool
RUN wget -O - https://github.com/hmrc/github-admin-tool/releases/download/${GITHUB_ADMIN_TOOL_VERSION}/github-admin-tool_Linux_x86_64.tar.gz \
  | tar -xzv \
  && mv github-admin-tool github-admin-tool \
  && chmod 755 github-admin-tool

# download AWS Lambda RIE
RUN wget -O aws-lambda-rie \
  https://github.com/aws/aws-lambda-runtime-interface-emulator/releases/latest/download/aws-lambda-rie \
  && chmod +x aws-lambda-rie

# build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-w -s -extldflags "-static"' -o main

# copy artifacts to a clean image
FROM scratch AS distro
COPY --from=build /app/main /main
COPY --from=build /app/github-admin-tool /github-admin-tool
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT [ "/main" ]
WORKDIR /

FROM distro AS rie
COPY --from=build /app/aws-lambda-rie /aws-lambda-rie
# creates /tmp for storing the report
COPY --from=build /tmp /tmp
EXPOSE 8080
ENTRYPOINT [ "/aws-lambda-rie" ]
