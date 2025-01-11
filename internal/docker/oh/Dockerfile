# syntax=docker/dockerfile:1

###
FROM python:3.12.3-slim AS repo

RUN apt-get update && apt-get install -y git

WORKDIR /app

ARG VERSION=0.19.0

RUN git clone --branch ${VERSION} --depth 1 https://github.com/All-Hands-AI/OpenHands.git /app

RUN chmod -R 770 /app/microagents && \
    chmod -R 770 /app/openhands/ && \
    chmod -R 770 /app/containers/app/entrypoint.sh

###
# https://github.com/All-Hands-AI/OpenHands/blob/0.19.0/containers/app/Dockerfile
FROM python:3.12.3-slim AS backend-builder

WORKDIR /app
ENV PYTHONPATH='/app'

ENV POETRY_NO_INTERACTION=1 \
    POETRY_VIRTUALENVS_IN_PROJECT=1 \
    POETRY_VIRTUALENVS_CREATE=1 \
    POETRY_CACHE_DIR=/tmp/poetry_cache

RUN apt-get update -y \
    && apt-get install -y curl make git build-essential \
    && python3 -m pip install poetry==1.8.2  --break-system-packages

COPY --from=repo /app/pyproject.toml /app/poetry.lock ./
RUN touch README.md
RUN export POETRY_CACHE_DIR && poetry install --without evaluation,llama-index --no-root && rm -rf $POETRY_CACHE_DIR

FROM python:3.12.3-slim AS openhands-app

WORKDIR /app

ARG OPENHANDS_BUILD_VERSION #re-declare for this section

ENV RUN_AS_OPENHANDS=true
# A random number--we need this to be different from the user's UID on the host machine
ENV OPENHANDS_USER_ID=42420
ENV SANDBOX_LOCAL_RUNTIME_URL=http://host.docker.internal
ENV USE_HOST_NETWORK=false
ENV WORKSPACE_BASE=/opt/workspace_base
ENV OPENHANDS_BUILD_VERSION=$OPENHANDS_BUILD_VERSION
ENV SANDBOX_USER_ID=0
ENV FILE_STORE=local
ENV FILE_STORE_PATH=/.openhands-state
RUN mkdir -p $FILE_STORE_PATH
RUN mkdir -p $WORKSPACE_BASE

RUN apt-get update -y \
    && apt-get install -y curl ssh sudo

# Default is 1000, but OSX is often 501
RUN sed -i 's/^UID_MIN.*/UID_MIN 499/' /etc/login.defs
# Default is 60000, but we've seen up to 200000
RUN sed -i 's/^UID_MAX.*/UID_MAX 1000000/' /etc/login.defs

RUN groupadd app
RUN useradd -l -m -u $OPENHANDS_USER_ID -s /bin/bash openhands && \
    usermod -aG app openhands && \
    usermod -aG sudo openhands && \
    echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers
RUN chown -R openhands:app /app && chmod -R 770 /app
RUN sudo chown -R openhands:app $WORKSPACE_BASE && sudo chmod -R 770 $WORKSPACE_BASE
USER openhands

ENV VIRTUAL_ENV=/app/.venv \
    PATH="/app/.venv/bin:$PATH" \
    PYTHONPATH='/app'

COPY --chown=openhands:app --from=backend-builder ${VIRTUAL_ENV} ${VIRTUAL_ENV}
RUN playwright install --with-deps chromium

COPY --chown=openhands:app --from=repo /app/microagents ./microagents
COPY --chown=openhands:app --from=repo /app/openhands ./openhands
COPY --chown=openhands:app --from=repo /app/openhands/runtime/plugins ./openhands/runtime/plugins
COPY --chown=openhands:app --from=repo /app/openhands/agenthub ./openhands/agenthub
COPY --chown=openhands:app --from=repo /app/pyproject.toml ./pyproject.toml
COPY --chown=openhands:app --from=repo /app/poetry.lock ./poetry.lock
COPY --chown=openhands:app --from=repo /app/README.md ./README.md
COPY --chown=openhands:app --from=repo /app/MANIFEST.in ./MANIFEST.in
COPY --chown=openhands:app --from=repo /app/LICENSE ./LICENSE

# This is run as "openhands" user, and will create __pycache__ with openhands:openhands ownership
RUN python openhands/core/download.py # No-op to download assets
# Add this line to set group ownership of all files/directories not already in "app" group
# openhands:openhands -> openhands:app
RUN find /app \! -group app -exec chgrp app {} +

# COPY --chown=openhands:app --chmod=770 --from=frontend-builder /app/build ./frontend/build
COPY --chown=openhands:app --from=repo /app/containers/app/entrypoint.sh /app/entrypoint.sh

USER root

RUN chmod -R 777 ${FILE_STORE_PATH}

WORKDIR /app

ENTRYPOINT ["/app/entrypoint.sh"]
# CMD ["uvicorn", "openhands.server.listen:app", "--host", "0.0.0.0", "--port", "3000"]
