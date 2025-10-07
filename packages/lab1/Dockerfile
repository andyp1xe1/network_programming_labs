FROM python:3.13-slim
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/
WORKDIR /workspace/

RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=bind,source=uv.lock,target=/workspace/uv.lock \
    --mount=type=bind,source=pyproject.toml,target=/workspace/pyproject.toml \
    --mount=type=bind,source=packages/lab1/pyproject.toml,target=/workspace/packages/lab1/pyproject.toml \
    uv sync --locked --no-install-project

ADD . /workspace/

RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=bind,source=uv.lock,target=/workspace/uv.lock \
    --mount=type=bind,source=pyproject.toml,target=/workspace/pyproject.toml \
    --mount=type=bind,source=packages/lab1/pyproject.toml,target=/workspace/packages/lab1/pyproject.toml \
    uv sync --locked

EXPOSE 8080
CMD ["uv", "run", "./packages/lab1/src/lab1/server.py", "./packages/lab1/src/lab1/www", "8080"]
