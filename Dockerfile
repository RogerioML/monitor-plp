FROM golang:1.16

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN go get -v github.com/go-delve/delve/cmd/dlv

WORKDIR ${GOPATH}/src
COPY . .

EXPOSE 2345
