FROM golang:1.20.6 as builder
MAINTAINER Venil Noronha <veniln@vmware.com>

WORKDIR /root/go/src/github.com/venilnoronha/grpc-web-istio-demo/
COPY ./ .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -v -mod=vendor -o bin/server ./cmd/server.go

#FROM scratch
FROM alpine:3.18.4
WORKDIR /bin/
COPY --from=builder /root/go/src/github.com/venilnoronha/grpc-web-istio-demo/bin/server .
#ENTRYPOINT [ "/bin/server" ]
CMD ["/bin/server"]
EXPOSE 9000
