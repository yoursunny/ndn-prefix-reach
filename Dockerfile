FROM golang:1.21 AS build
WORKDIR /app
COPY . .
RUN env CGO_ENABLED=0 GOBIN=/build go install .

FROM scratch
COPY --from=build /build/* /
EXPOSE 6774
ENTRYPOINT ["/ndn-prefix-reach"]
