FROM golang:alpine3.11 AS builder
MAINTAINER x0rzkov

# RUN apk add --no-cache make gcc g++ ca-certificates musl-dev make git

COPY . /go/src/github.com/x0rzkov/manticore-test
WORKDIR /go/src/github.com/x0rzkov/manticore-test

RUN go install

FROM alpine:3.11 AS runtime
MAINTAINER x0rzkov

ARG TINI_VERSION=${TINI_VERSION:-"v0.18.0"}

# Install tini to /usr/local/sbin
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-muslc-amd64 /usr/local/sbin/tini

# Install runtime dependencies & create runtime user
RUN apk --no-cache --no-progress add ca-certificates \
 && chmod +x /usr/local/sbin/tini && mkdir -p /opt \
 && adduser -D manticore -h /opt/manticore -s /bin/sh \
 && su manticore -c 'cd /opt/manticore; mkdir -p bin config data ui'

# Switch to user context
USER manticore
WORKDIR /opt/ncarlier/data

# copy executable
COPY --from=builder /go/bin/manticore-test /opt/manticore/bin/manticore-test

ENV PATH $PATH:/opt/manticore/bin

# Container configuration
VOLUME ["/opt/manticore/data"]
ENTRYPOINT ["tini", "-g", "--"]
CMD ["/opt/manticore/bin/manticore-test"]
