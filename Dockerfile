FROM golang:1.10-alpine as builder

ARG VERSION

RUN apk add --no-cache git ca-certificates

 RUN git clone --branch "$VERSION" --single-branch --depth 1 \
      https://github.com/korylprince/nginx-swarm.git /go/src/github.com/korylprince/nginx-swarm

RUN go install github.com/korylprince/nginx-swarm

FROM alpine:3.7

RUN apk add --no-cache nginx nginx-mod-stream ca-certificates

COPY --from=builder /go/bin/nginx-swarm /

RUN mkdir /run/nginx

CMD ["/nginx-swarm", "-g", "load_module /usr/lib/nginx/modules/ngx_stream_module.so;"]
