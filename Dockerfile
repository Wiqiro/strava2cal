FROM golang:alpine AS builder

WORKDIR /api

COPY api .

RUN go mod download
RUN go build -o server .

FROM nginx:alpine

COPY --from=builder /api/server /usr/local/bin/server

COPY index.html /usr/share/nginx/html/index.html

COPY nginx.conf /etc/nginx/conf.d/default.conf

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV CALENDAR_FILE="/data/calendar.ics"
ENV STATE_FILE="/data/state.json"

VOLUME ["/data"]

EXPOSE 80

ENTRYPOINT ["/entrypoint.sh"]
