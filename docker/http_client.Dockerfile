FROM golang as builder
MAINTAINER Venil Noronha <veniln@vmware.com>

WORKDIR /root/go/src/github.com/venilnoronha/grpc-web-istio-demo/
COPY ./ .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -v -mod=vendor -o bin/http_client ./cmd/http_client.go

FROM scratch
WORKDIR /bin/
COPY --from=builder /root/go/src/github.com/venilnoronha/grpc-web-istio-demo/bin/http_client .
ENTRYPOINT [ "/bin/http_client" ]
