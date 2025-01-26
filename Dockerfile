FROM golang:1 AS builder

WORKDIR /workdir

COPY . .

ENV CGO_ENABLED=0
RUN go build -o rewind

FROM scratch

COPY --from=builder /workdir/rewind /rewind

ENTRYPOINT ["/rewind"]
