FROM golang:alpine

WORKDIR /api
COPY . .

RUN go build -o server .
EXPOSE 8080

ENTRYPOINT ["/api/server"]
