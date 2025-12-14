# 🚀 Full-Stack AI Platform
# Powered by Container-Maker

.PHONY: help dev backend frontend test lint deploy clean

# Default target
help:
	@echo ""
	@echo "🚀 Full-Stack AI Platform Commands"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "  Development:"
	@echo "    make dev        - Start all services (backend + frontend)"
	@echo "    make backend    - Start FastAPI backend only"
	@echo "    make frontend   - Start React frontend only"
	@echo ""
	@echo "  AI & ML:"
	@echo "    make train      - Train the ML model"
	@echo "    make inference  - Run inference server"
	@echo "    make embed      - Generate embeddings"
	@echo ""
	@echo "  Testing:"
	@echo "    make test       - Run all tests"
	@echo "    make lint       - Run linters"
	@echo "    make typecheck  - Run type checking"
	@echo ""
	@echo "  Database:"
	@echo "    make migrate    - Run database migrations"
	@echo "    make seed       - Seed database with demo data"
	@echo ""
	@echo "  Deployment:"
	@echo "    make build      - Build production images"
	@echo "    make deploy     - Deploy to production"
	@echo ""

# === Development ===

dev: 
	@echo "🚀 Starting Full-Stack AI Platform..."
	@make -j2 backend frontend

backend:
	@echo "🐍 Starting FastAPI backend on http://localhost:8000"
	cd backend && uvicorn main:app --reload --host 0.0.0.0 --port 8000

frontend:
	@echo "⚛️  Starting React frontend on http://localhost:3000"
	cd frontend && pnpm dev

# === AI & ML ===

train:
	@echo "🧠 Training ML model..."
	python ml/train.py --epochs 10 --batch-size 32

inference:
	@echo "🔮 Starting inference server..."
	python ml/inference_server.py

embed:
	@echo "📊 Generating embeddings..."
	python ml/generate_embeddings.py

# === Testing ===

test:
	@echo "🧪 Running tests..."
	pytest backend/tests -v --cov=backend
	cd frontend && pnpm test

lint:
	@echo "🔍 Running linters..."
	ruff check backend/
	cd frontend && pnpm lint

typecheck:
	@echo "📝 Type checking..."
	mypy backend/
	cd frontend && pnpm typecheck

# === Database ===

migrate:
	@echo "📦 Running migrations..."
	alembic upgrade head

seed:
	@echo "🌱 Seeding database..."
	python scripts/seed_db.py

# === Deployment ===

build:
	@echo "🏗️  Building production images..."
	docker build -t aiplatform-backend:latest ./backend
	docker build -t aiplatform-frontend:latest ./frontend

deploy:
	@echo "🚀 Deploying to production..."
	kubectl apply -f k8s/

# === Cleanup ===

clean:
	@echo "🧹 Cleaning up..."
	find . -type d -name __pycache__ -exec rm -rf {} +
	find . -type f -name "*.pyc" -delete
	cd frontend && rm -rf node_modules .next
