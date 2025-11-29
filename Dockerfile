FROM golang:alpine

COPY . /
WORKDIR /api

RUN go build -o server .
EXPOSE 8080

ENTRYPOINT ["/api/server"]
