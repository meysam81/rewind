FROM golang:1 AS builder

WORKDIR /workdir

COPY . .

RUN go build -o requestrewind

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /workdir/requestrewind /requestrewind

CMD ["/requestrewind"]
