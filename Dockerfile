FROM golang:alpine as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN apk --update upgrade && \
    apk add --no-cache tzdata sqlite sqlite-dev gcc libc-dev
RUN go mod download
COPY . .
ARG GIT_COMMIT=empty
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-X main.GIT_COMMIT=$GIT_COMMIT -s -w -linkmode external -extldflags "-static"' \
  -gcflags="all=-trimpath=${PWD}" \
  -asmflags="all=-trimpath=${PWD}" \
  -o start

FROM alpine
WORKDIR /app
COPY --from=builder /app/start .

VOLUME [ "/app/data" ]
EXPOSE 8080
ENV GIN_MODE=release

CMD /app/start
