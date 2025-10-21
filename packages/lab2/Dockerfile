FROM python:3.13-slim
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/
WORKDIR /workspace/

RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=bind,source=uv.lock,target=/workspace/uv.lock \
    --mount=type=bind,source=pyproject.toml,target=/workspace/pyproject.toml \
    --mount=type=bind,source=packages/,target=/workspace/packages/ \
    uv sync --locked --no-install-project

ADD . /workspace/

RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=bind,source=uv.lock,target=/workspace/uv.lock \
    --mount=type=bind,source=pyproject.toml,target=/workspace/pyproject.toml \
    --mount=type=bind,source=packages/lab2/pyproject.toml,target=/workspace/packages/lab2/pyproject.toml \
    uv sync --locked

CMD ["uv", "run", "./packages/lab2/src/lab2/server.py"]
