#!/bin/bash
set -e

# OpenOIDC Development Environment Startup Script
# 开发环境一键启动脚本

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  OpenOIDC Development Environment${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}[INFO] .env not found, copying from .env.example${NC}"
    cp .env.example .env
    echo -e "${GREEN}✓ Created .env file${NC}"
fi

# Check Go installation
if ! command -v go &> /dev/null; then
    echo -e "${RED}[ERROR] Go is not installed${NC}"
    exit 1
fi

# Check npm/node installation
if ! command -v npm &> /dev/null; then
    echo -e "${RED}[ERROR] npm is not installed${NC}"
    exit 1
fi

# Build frontend if needed
if [ ! -d "frontend/node_modules" ]; then
    echo -e "${YELLOW}[INFO] Installing frontend dependencies...${NC}"
    cd frontend
    npm install
    cd ..
    echo -e "${GREEN}✓ Frontend dependencies installed${NC}"
fi

# Start local infrastructure for PostgreSQL-backed development.
if command -v docker >/dev/null 2>&1; then
    echo -e "${YELLOW}[INFO] Starting PostgreSQL and Redis...${NC}"
    docker compose up -d postgres redis
    echo -e "${GREEN}✓ PostgreSQL and Redis are ready or starting${NC}"
else
    echo -e "${YELLOW}[WARN] Docker not found; make sure PostgreSQL and Redis are running locally${NC}"
fi

# Check if frontend dev server is already running
if lsof -Pi :5173 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Frontend dev server already running on http://localhost:5173${NC}"
else
    echo -e "${YELLOW}[INFO] Starting frontend dev server...${NC}"
    cd frontend
    npm run dev > ../frontend-dev.log 2>&1 &
    FRONTEND_PID=$!
    cd ..
    echo $FRONTEND_PID > .frontend.pid
    echo -e "${GREEN}✓ Frontend dev server started (PID: $FRONTEND_PID)${NC}"
fi

# Wait for frontend to be ready
echo -e "${YELLOW}[INFO] Waiting for frontend...${NC}"
for i in {1..30}; do
    if curl -s http://localhost:5173 > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Frontend ready${NC}"
        break
    fi
    sleep 1
done

# Check if backend is already running
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Backend server already running on http://localhost:8080${NC}"
else
    echo -e "${YELLOW}[INFO] Starting backend server...${NC}"
    # Use China proxy for Go modules
    export GOPROXY=https://goproxy.cn,direct
    go run ./cmd/server > backend-dev.log 2>&1 &
    BACKEND_PID=$!
    echo $BACKEND_PID > .backend.pid
    echo -e "${GREEN}✓ Backend server started (PID: $BACKEND_PID)${NC}"

    # Wait for backend to be ready
    echo -e "${YELLOW}[INFO] Waiting for backend...${NC}"
    for i in {1..60}; do
        if curl -s http://localhost:8080/.well-known/openid-configuration > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Backend ready${NC}"
            break
        fi
        sleep 1
    done
fi

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  ✓ Development environment is ready!${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""
echo -e "${BLUE}Frontend:${NC}      http://localhost:5173"
echo -e "${BLUE}Backend API:${NC}   http://localhost:8080"
echo -e "${BLUE}Admin Panel:${NC}   http://localhost:8080/admin"
echo ""
echo -e "${BLUE}Admin Credentials:${NC}"
grep "OIDC_ADMIN_EMAIL\|OIDC_ADMIN_PASSWORD" .env | sed 's/OIDC_/  /' | sed 's/=/ = /'
echo ""
echo -e "${YELLOW}Logs:${NC}"
echo -e "  Frontend: tail -f frontend-dev.log"
echo -e "  Backend:  tail -f backend-dev.log"
echo ""
echo -e "${YELLOW}To stop:${NC} ./dev.sh stop"
echo ""
