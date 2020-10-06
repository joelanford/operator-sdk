FROM golang:1.13-alpine as builder
RUN apk update && apk add make git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./
RUN GOOS=linux CGO_ENABLED=0 go build -o build/scorecard-test ./hack/image/scorecard-test/cmd/test/main.go

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV HOME=/tmp \
    USER_UID=1001 \
    USER_NAME=test

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd
COPY --from=builder /app/build/scorecard-test /usr/local/bin/scorecard-test

WORKDIR ${HOME}
USER ${USER_UID}
ENTRYPOINT ["/usr/local/bin/scorecard-test"]
