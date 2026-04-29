# Build the manager binary
# Use --platform=$BUILDPLATFORM to run the builder natively on the build machine
# while cross-compiling for the target architecture (ARM64/AMD64)
FROM --platform=$BUILDPLATFORM golang:1.25.8 AS builder

ARG TARGETARCH
ARG BITBUCKET_USR
ARG BITBUCKET_PSW
ARG GITHUB_USR
ARG GITHUB_PSW

RUN printf "machine git.pega.io login %s password %s\nmachine github.com login %s password %s\n" \
    "$BITBUCKET_USR" "$BITBUCKET_PSW" "$GITHUB_USR" "$GITHUB_PSW" > $HOME/.netrc

ENV GOPRIVATE=github.com/Pega-CloudEngineering/*
ENV GONOSUMDB=github.com/Pega-CloudEngineering/*

WORKDIR /workspace
# Copy the Go Modules manifests
COPY libs/go.mod go.mod
COPY libs/go.sum go.sum

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY libs/cmd/ops cmd/ops
COPY libs/cmd/service/health cmd/service/health
COPY libs/internal internal/
COPY libs/pkg pkg/

# Build - TARGETARCH is automatically set by Docker BuildKit (arm64 or amd64)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -a -o genaiops cmd/ops/main.go

# Open port 8080 for listening and 8082 for healthcheck
EXPOSE 8081/tcp
EXPOSE 8082/tcp

# Use distroless as a minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
#
# For debugging use the debug version to have shell on the container:
# $ kubectl exec -it genai-hub-service-9bb47d8d6-cxrhj -c genai-hub-service -- sh
#FROM gcr.io/distroless/static:debug
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/genaiops .
USER 65532:65532

ENTRYPOINT ["/genaiops"]