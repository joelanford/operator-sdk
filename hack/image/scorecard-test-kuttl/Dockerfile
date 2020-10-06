FROM golang:1.13-alpine as builder
RUN apk update && apk add make git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./
RUN GOOS=linux CGO_ENABLED=0 go build -o build/scorecard-test-kuttl ./hack/image/scorecard-test-kuttl/cmd/test-kuttl/main.go

FROM kudobuilder/kuttl:v0.6.1

ENV HOME=/tmp \
    USER_UID=1001 \
    USER_NAME=test \
    KUBECONFIG=

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd
COPY --from=builder /app/build/scorecard-test-kuttl /usr/local/bin/scorecard-test-kuttl

USER ${USER_UID}
ENTRYPOINT ["/usr/local/bin/scorecard-test-kuttl"]

