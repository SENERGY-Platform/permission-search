FROM golang:1.11

COPY . /go/src/permission-search
WORKDIR /go/src/permission-search

ENV GO111MODULE=on

RUN go build

EXPOSE 8080

CMD ./permission-search