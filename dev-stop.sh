#!/bin/bash

# OpenOIDC Development Environment Stop Script
# 停止开发环境脚本

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Stopping OpenOIDC development servers...${NC}"

# Stop frontend
if [ -f .frontend.pid ]; then
    FRONTEND_PID=$(cat .frontend.pid)
    if ps -p $FRONTEND_PID > /dev/null 2>&1; then
        kill $FRONTEND_PID
        echo -e "${GREEN}✓ Frontend server stopped (PID: $FRONTEND_PID)${NC}"
    fi
    rm .frontend.pid
fi

# Stop backend
if [ -f .backend.pid ]; then
    BACKEND_PID=$(cat .backend.pid)
    if ps -p $BACKEND_PID > /dev/null 2>&1; then
        kill $BACKEND_PID
        echo -e "${GREEN}✓ Backend server stopped (PID: $BACKEND_PID)${NC}"
    fi
    rm .backend.pid
fi

# Fallback: kill by port
if lsof -Pi :5173 -sTCP:LISTEN -t >/dev/null 2>&1; then
    lsof -ti:5173 | xargs kill -9
    echo -e "${GREEN}✓ Killed process on port 5173${NC}"
fi

if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    lsof -ti:8080 | xargs kill -9
    echo -e "${GREEN}✓ Killed process on port 8080${NC}"
fi

echo -e "${GREEN}✓ All servers stopped${NC}"
