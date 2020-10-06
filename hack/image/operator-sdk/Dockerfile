FROM golang:1.13-alpine as builder
RUN apk update && apk add make git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./
RUN make build/operator-sdk-dev-linux-gnu

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

COPY --from=builder /app/build/operator-sdk-dev-linux-gnu  /usr/local/bin/operator-sdk

RUN microdnf install -y golang make which

ENTRYPOINT ["/usr/local/bin/operator-sdk"]
