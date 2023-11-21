# builder image
FROM golang:1.21-alpine AS builder

# copy the source code to the container
RUN mkdir /build
WORKDIR /build
COPY . .

# install vendor dependencies, you may use `GOPROXY` to speed it up
# in mainland China.
#ENV GOPROXY=https://goproxy.cn,direct
RUN go mod tidy

# build the executable
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o web-fetcher .

# final target image for multi-stage builds
FROM alpine:3.18

RUN apk --no-cache add ca-certificates

# set the working directory and copy binary
RUN mkdir /app /app/output
WORKDIR /app
COPY --from=builder --chmod=755 /build/web-fetcher ./fetcher

# environment settings
ENV ROOT_STORE_DIR=/app/output

# run the executable with the arguments
ENTRYPOINT [ "./fetcher" ]
CMD [ "--help" ]