#
# Copyright (c) 2024 Pegasystems Inc.
# All rights reserved.
#

# Build the manager binary
# Use --platform=$BUILDPLATFORM to run the builder natively on the build machine
# while cross-compiling for the target architecture (ARM64/AMD64)
FROM --platform=$BUILDPLATFORM golang:1.25 as builder

ARG TARGETARCH
ARG BITBUCKET_USR
ARG BITBUCKET_PSW
ARG GITHUB_USR
ARG GITHUB_PSW

RUN echo "machine git.pega.io login $BITBUCKET_USR password $BITBUCKET_PSW" > $HOME/.netrc && \
    echo "machine github.com login $GITHUB_USR password $GITHUB_PSW" >> $HOME/.netrc

WORKDIR /workspace

# Copy the Go Modules manifests
COPY libs/go.mod go.mod
COPY libs/go.sum go.sum

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY libs/cmd/background cmd/background
COPY libs/cmd/middleware cmd/middleware
COPY libs/internal internal/
COPY libs/apidocs apidocs/

# Build - TARGETARCH is automatically set by Docker BuildKit (arm64 or amd64)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -a -o genaivectorstorebackground cmd/background/main.go

# Open port 8080 for listening and 8082 for healthcheck
EXPOSE 8080/tcp
EXPOSE 8082/tcp

# Use distroless as a minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM gcr.bin.pega.io/distroless/static:nonroot
# gcr.bin.pega.io is constantly failing with a following error:
# 429  - Server message: unknown: 'gcr' Docker remote repository reached the request rate limit set by the Docker registry. Please contact Artifactory admin.
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/genaivectorstorebackground .
COPY libs/apidocs apidocs/
USER 65532:65532

ENTRYPOINT ["/genaivectorstorebackground"]
