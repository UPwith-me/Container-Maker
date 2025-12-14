FROM nvidia/cuda:12.2-devel-ubuntu22.04

# Avoid prompts
ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
RUN apt-get update && apt-get install -y \
    curl wget git vim nano htop \
    python3.11 python3.11-venv python3-pip \
    nodejs npm \
    build-essential cmake \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

# Set Python 3.11 as default
RUN update-alternatives --install /usr/bin/python python /usr/bin/python3.11 1 \
    && update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.11 1

# Install Python packages
RUN pip install --no-cache-dir \
    fastapi uvicorn[standard] \
    sqlalchemy asyncpg \
    redis aioredis \
    transformers torch \
    langchain openai \
    pandas numpy scikit-learn \
    pytest pytest-asyncio \
    black ruff mypy

# Install Node.js tools
RUN npm install -g pnpm typescript tsx

# Create workspace
WORKDIR /workspace

# Create non-root user
RUN useradd -m -s /bin/bash developer \
    && chown -R developer:developer /workspace

USER developer

CMD ["sleep", "infinity"]
