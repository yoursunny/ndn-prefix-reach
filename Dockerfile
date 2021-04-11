FROM golang:1 AS build
WORKDIR /go/src/app
COPY . .
RUN env CGO_ENABLED=0 go build .

FROM scratch
COPY --from=build /go/src/app/ndn-prefix-reach /ndn-prefix-reach
EXPOSE 6774
ENTRYPOINT ["/ndn-prefix-reach"]
