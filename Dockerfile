FROM golang:1.19-alpine AS builder
RUN apk update && apk add --no-cache git make bash
WORKDIR $GOPATH/src/silta-deployment-remover
COPY /app .
RUN go mod download \
  && CGO_ENABLED=0 GOOS=linux go build -a -gcflags=-trimpath=$(go env GOPATH) -asmflags=-trimpath=$(go env GOPATH) -ldflags '-extldflags "-static"' -o silta-deployment-remover

FROM alpine:3.16
RUN apk add --no-cache bash curl tini

# Copy go application
COPY --from=builder /go/src/silta-deployment-remover /bin

EXPOSE 8080

# Start application
ENTRYPOINT [ "/sbin/tini", "--"]
CMD ["/bin/silta-deployment-remover"]
