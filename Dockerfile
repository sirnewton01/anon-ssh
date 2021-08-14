FROM golang:buster

WORKDIR /go/src/app

COPY . .
RUN go get -u ./...
RUN go install ./...

