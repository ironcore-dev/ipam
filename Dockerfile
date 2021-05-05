# Build the manager binary
FROM golang:1.15 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

RUN go env -w GOPRIVATE='github.com/onmetal/*'
ARG GIT_USER
ARG GIT_PASSWORD
RUN if [ ! -z "$GIT_USER" ] && [ ! -z "$GIT_PASSWORD" ]; then \
        printf "machine github.com\n \
            login ${GIT_USER}\n \
            password ${GIT_PASSWORD}\n \
            \nmachine api.github.com\n \
            login ${GIT_USER}\n \
            password ${GIT_PASSWORD}\n" \
            >> ${HOME}/.netrc;\
    fi

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
