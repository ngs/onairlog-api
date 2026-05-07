FROM golang:1.21 as builder

WORKDIR /go/src/cloudrun/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -v -o app

FROM gcr.io/distroless/static-debian12:latest
ENV TZ=Asia/Tokyo
COPY --from=builder /go/src/cloudrun/app/app /app

CMD ["/app"]
