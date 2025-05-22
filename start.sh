#!/bin/bash
# Shell script to start both frontend and backend services

# Colors for output
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Define functions for starting each service
start_backend() {
    echo -e "${GREEN}Starting Go Fiber backend...${NC}"
    gnome-terminal --title="Backend Server" -- bash -c "cd '$SCRIPT_DIR/backend' && go run main.go; read -p 'Press Enter to close...'"
    # Alternative if gnome-terminal is not available
    # xterm -T "Backend Server" -e "cd '$SCRIPT_DIR/backend' && go run main.go; read -p 'Press Enter to close...'" &
}

start_frontend() {
    echo -e "${CYAN}Starting Vue.js frontend...${NC}"
    gnome-terminal --title="Frontend Server" -- bash -c "cd '$SCRIPT_DIR/frontend' && npm run dev; read -p 'Press Enter to close...'"
    # Alternative if gnome-terminal is not available
    # xterm -T "Frontend Server" -e "cd '$SCRIPT_DIR/frontend' && npm run dev; read -p 'Press Enter to close...'" &
}

# Check if we can use the terminal commands, otherwise fall back to basic method
start_services_in_terminal() {
    # Check if gnome-terminal exists
    if command -v gnome-terminal &> /dev/null; then
        start_backend
        start_frontend
    # Check if xterm exists as backup
    elif command -v xterm &> /dev/null; then
        # Uncomment the xterm lines above and use those instead
        echo -e "${YELLOW}Using xterm instead of gnome-terminal${NC}"
        xterm -T "Backend Server" -e "cd '$SCRIPT_DIR/backend' && go run main.go; read -p 'Press Enter to close...'" &
        xterm -T "Frontend Server" -e "cd '$SCRIPT_DIR/frontend' && npm run dev; read -p 'Press Enter to close...'" &
    else
        # Fall back to basic operation
        start_services_basic
    fi
}

# Fallback method if no terminal emulators are available
start_services_basic() {
    echo -e "${YELLOW}Starting services in background...${NC}"
    # Start backend
    cd "$SCRIPT_DIR/backend" && go run main.go &
    BACKEND_PID=$!
    
    # Start frontend
    cd "$SCRIPT_DIR/frontend" && npm run dev &
    FRONTEND_PID=$!
    
    echo -e "${YELLOW}Services started in the background.${NC}"
    echo -e "${YELLOW}Backend PID: ${GREEN}$BACKEND_PID${NC}"
    echo -e "${YELLOW}Frontend PID: ${CYAN}$FRONTEND_PID${NC}"
    echo -e "${YELLOW}Use 'kill $BACKEND_PID $FRONTEND_PID' to stop the services.${NC}"
}

# Main execution
echo -e "${YELLOW}Starting Intel Stealer App (Vue.js + Go Fiber)${NC}"
echo -e "${YELLOW}----------------------------------------${NC}"

# Create log directory if it doesn't exist yet
mkdir -p "$SCRIPT_DIR/backend/log"
echo -e "${GREEN}Ensuring log directory exists at:${NC} $SCRIPT_DIR/backend/log"

# Start the services
start_services_in_terminal

# Print access information
echo -e "\n${YELLOW}Services started in separate windows. Press Ctrl+C in each window to stop them.${NC}"
echo -e "${CYAN}Frontend will be available at:${NC} http://localhost:5173"
echo -e "${GREEN}Backend API will be available at:${NC} http://localhost:3000/api/hello"
