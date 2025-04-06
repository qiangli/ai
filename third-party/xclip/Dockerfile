# syntax=docker/dockerfile:1

###
FROM ubuntu:latest AS builder

WORKDIR /app

RUN apt-get update && \
    apt-get install -y \
    build-essential \
    git \
    autoconf \
    libtool \
    libxmu-dev \
    xorg-dev \
    xz-utils \
    wget \
    libx11-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root

COPY ./build-xclip.sh .

RUN chmod +x build-xclip.sh && \
    ./build-xclip.sh

CMD [ "bash" ]