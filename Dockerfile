FROM golang:1.14 AS builder
ADD . /app/backend
WORKDIR /app/backend
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /main .

# final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY ./schemas ./schemas
COPY --from=builder /main ./
RUN chmod +x ./main
ENTRYPOINT ["./main"]