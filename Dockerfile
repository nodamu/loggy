FROM golang:alpine3.20 AS build
WORKDIR /go/src/loggy
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /go/bin/loggy ./cmd/loggy

FROM scratch
COPY --from=build /go/bin/loggy /bin/loggy
ENTRYPOINT ["/bin/loggy"]