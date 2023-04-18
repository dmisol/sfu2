FROM ubuntu:22.04 as build
WORKDIR /app

ARG DEBIAN_FRONTEND=noninteractive
ENV TZ=Etc/GMT
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates locales g++ gcc libc6-dev make pkg-config wget git libopus-dev libavcodec-dev libgstreamer1.0-dev ffmpeg apt-utils ssh-client yasm libx264-dev libsoxr-dev && \
    rm -rf /var/lib/apt/lists/*
RUN wget https://go.dev/dl/go1.19.5.linux-amd64.tar.gz && tar -C /usr/local -xzf go1.19.5.linux-amd64.tar.gz

RUN /usr/local/go/bin/go version

COPY go.mod ./
COPY go.sum ./

COPY internal ./internal
COPY main.go ./main.go

RUN /usr/local/go/bin/go mod tidy
RUN /usr/local/go/bin/go build -v -o main .


FROM ubuntu:22.04
WORKDIR /opt/env

ARG DEBIAN_FRONTEND=noninteractive
ENV TZ=Europe/Moscow
# RUN apt install gcc-9
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates locales g++ gcc libc6-dev make pkg-config wget git libopus-dev sox ffmpeg apt-utils tcpdump yasm libx264-dev libsoxr-dev && \
    rm -rf /var/lib/apt/lists/*

VOLUME /tmp

COPY --from=build /app/main ./main
COPY static ./static
COPY conf.yaml ./conf.yaml

ENTRYPOINT ["/bin/sh", "-c", "./main >> /tmp/sfu2.txt 2>&1"]
#ENTRYPOINT ["/bin/sh", "-c", "ls -al /static/demo"]
