FROM golang:1 AS builder

WORKDIR /workdir

COPY . .

ENV CGO_ENABLED=0
RUN go build -o requestrewind

FROM scratch

COPY --from=builder /workdir/requestrewind /requestrewind

ENTRYPOINT ["/requestrewind"]
