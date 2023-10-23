FROM golang as builder
MAINTAINER Venil Noronha <veniln@vmware.com>

WORKDIR /root/go/src/github.com/venilnoronha/grpc-web-istio-demo/
COPY ./ .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -v -mod=vendor -o bin/http_server ./cmd/http_server.go

FROM scratch
WORKDIR /bin/
COPY --from=builder /root/go/src/github.com/venilnoronha/grpc-web-istio-demo/bin/http_server .
ENTRYPOINT [ "/bin/http_server" ]
EXPOSE 12345
