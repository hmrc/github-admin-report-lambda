FROM golang:1.16-alpine3.14 as build
# cache dependencies
WORKDIR /app
# cache dependencies
COPY go.mod go.sum ./
RUN go mod download
# install github-admin-tool
RUN wget -O- https://github.com/hmrc/github-admin-tool/releases/download/v0.1.2/github-admin-tool_0.1.2_Linux_arm64.tar.gz \
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
EXPOSE 8080
ENTRYPOINT [ "/aws-lambda-rie" ]
