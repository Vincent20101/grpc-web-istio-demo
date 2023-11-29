FROM golang as builder
MAINTAINER Venil Noronha <veniln@vmware.com>

WORKDIR /root/go/src/github.com/venilnoronha/grpc-web-istio-demo/
COPY ./ .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -v -mod=vendor -o bin/client ./cmd/client.go

#FROM scratch
FROM alpine:3.18.4
WORKDIR /bin/
COPY --from=builder /root/go/src/github.com/venilnoronha/grpc-web-istio-demo/bin/client .
#ENTRYPOINT [ "/bin/client" ]
CMD [ "/bin/client" ]
