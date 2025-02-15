# syntax=docker/dockerfile:1

###
FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends git build-essential

WORKDIR /app

ARG VERSION=v3.1.9

RUN git clone --branch ${VERSION} --depth 1 https://github.com/assafelovic/gpt-researcher.git /app

RUN pip install --upgrade --no-cache-dir pip
RUN pip install --no-cache-dir -r requirements.txt
RUN pip install --no-cache-dir duckduckgo-search

ENTRYPOINT ["python", "cli.py"]