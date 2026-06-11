#!/usr/bin/env bash
# FlowBoard Local Development Startup Script (Linux/Mac)
# Run from the project root: ./scripts/start-local.sh

set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BLUE='\033[0;36m'; GREEN='\033[0;32m'; RED='\033[0;31m'; GRAY='\033[0;37m'; NC='\033[0m'

echo ""
echo -e "  ${BLUE}⚡ FlowBoard — Local Development${NC}"
echo -e "  ${GRAY}─────────────────────────────────────${NC}"
echo ""

# Check .env
if [ ! -f "$ROOT/.env" ]; then
    echo -e "  ${RED}[ERROR] .env file not found in project root.${NC}"
    echo -e "          Copy .env.example and fill in your values."
    exit 1
fi

# Kill any existing processes on required ports
for port in 8080 3000; do
    pid=$(lsof -ti tcp:$port 2>/dev/null || true)
    [ -n "$pid" ] && kill -9 $pid 2>/dev/null || true
done

echo -e "  ${GREEN}Starting backend (Go + SQLite)...${NC}"
cd "$ROOT/backend" && go run main.go &
BACKEND_PID=$!

sleep 2

echo -e "  ${GREEN}Starting frontend (Next.js)...${NC}"
cd "$ROOT/frontend" && npm run dev &
FRONTEND_PID=$!

echo ""
echo -e "  ${BLUE}✓ Backend:  http://localhost:8080${NC}"
echo -e "  ${BLUE}✓ Frontend: http://localhost:3000${NC}"
echo -e "  ${GRAY}✓ Data:     $ROOT/backend/db.json${NC}"
echo ""
echo -e "  ${GRAY}Press Ctrl+C to stop all services.${NC}"
echo ""

# Trap Ctrl+C to cleanly shut down both
trap "kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; echo '  Stopped.'; exit 0" SIGINT SIGTERM
wait $BACKEND_PID $FRONTEND_PID
