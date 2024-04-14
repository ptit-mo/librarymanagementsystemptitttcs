FROM golang:1.21.0-alpine
WORKDIR /app
COPY . .
RUN apk update && apk upgrade
RUN apk add sqlite
RUN sqlite3 library.db < db.sql
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -o main ./
ENTRYPOINT ["/app/main"]
