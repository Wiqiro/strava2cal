FROM golang:alpine

COPY . /

WORKDIR /api

RUN go build -o server .

ENV STATE_FILE="/data/state.json"

VOLUME ["/data"]

EXPOSE 8080

ENTRYPOINT ["/api/server"]
