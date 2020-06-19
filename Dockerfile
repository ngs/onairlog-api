FROM golang as builder

WORKDIR /go/src/cloudrun/app
COPY . .

RUN go mod vendor
RUN CGO_ENABLED=0 GOOS=linux go build -v -o app

FROM marketplace.gcr.io/google/ubuntu1804:latest
COPY --from=builder /go/src/cloudrun/app/app /app

CMD ["/app"]
