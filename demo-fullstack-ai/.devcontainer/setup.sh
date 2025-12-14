#!/bin/bash
# 🚀 Full-Stack AI Platform Setup Script
# Executed by Container-Maker after container creation

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 Setting up Full-Stack AI Platform"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Backend setup
echo "📦 Installing Python dependencies..."
cd /workspace/backend
pip install -r requirements.txt --quiet

# Frontend setup
echo "📦 Installing Node.js dependencies..."
cd /workspace/frontend
pnpm install --silent

# Database setup
echo "📊 Setting up database..."
cd /workspace
python scripts/init_db.py

# Download ML models
echo "🧠 Downloading AI models..."
python -c "from transformers import AutoModel; AutoModel.from_pretrained('sentence-transformers/all-MiniLM-L6-v2')" || true

# Git hooks
echo "🔧 Setting up git hooks..."
cd /workspace
git config core.hooksPath .githooks || true

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Setup complete!"
echo ""
echo "Available commands:"
echo "  make dev       - Start development servers"
echo "  make test      - Run tests"
echo "  make train     - Train ML model"
echo ""
echo "Services:"
echo "  Frontend:   http://localhost:3000"
echo "  Backend:    http://localhost:8000"
echo "  API Docs:   http://localhost:8000/docs"
echo "  Grafana:    http://localhost:3001"
echo "  Prometheus: http://localhost:9090"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
