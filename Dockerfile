FROM golang:1.19-alpine AS builder
RUN apk update && apk add --no-cache git make bash
WORKDIR $GOPATH/src/silta-deployment-remover
COPY /app .
RUN go mod download \
  && CGO_ENABLED=0 GOOS=linux go build -a -gcflags=-trimpath=$(go env GOPATH) -asmflags=-trimpath=$(go env GOPATH) -ldflags '-extldflags "-static"' -o silta-deployment-remover

FROM alpine:3.16
RUN apk add --no-cache bash curl tini

# # kubectl for testing
# RUN curl -L "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" -o /bin/kubectl \
#   && chmod +x /bin/kubectl

# Copy go application
RUN mkdir /app
COPY --from=builder /go/src/silta-deployment-remover /app
WORKDIR "/app"

EXPOSE 8080

# Start application
ENTRYPOINT [ "/sbin/tini", "--"]
CMD ["/bin/silta-deployment-remover"]

# # Debugging
# CMD ["sh", "-c", "tail -f /dev/null"]
