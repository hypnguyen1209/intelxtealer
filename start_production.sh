#!/bin/bash
# Production startup script for Intel Stealer application

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BACKEND_DIR="$SCRIPT_DIR/backend"
FRONTEND_DIR="$SCRIPT_DIR/frontend"

# Function to check if a command exists
command_exists() {
    command -v "$1" &> /dev/null
}

# Function to verify prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"
    
    # Check if Go is installed
    if ! command_exists go; then
        echo -e "${RED}ERROR: Go is not installed. Please install Go first.${NC}"
        exit 1
    fi
    
    # Check if Node.js and npm are installed
    if ! command_exists node || ! command_exists npm; then
        echo -e "${RED}ERROR: Node.js or npm is not installed. Please install them first.${NC}"
        exit 1
    fi
    
    # Check if the backend directory exists
    if [ ! -d "$BACKEND_DIR" ]; then
        echo -e "${RED}ERROR: Backend directory not found at $BACKEND_DIR${NC}"
        exit 1
    fi
    
    # Check if the frontend directory exists
    if [ ! -d "$FRONTEND_DIR" ]; then
        echo -e "${RED}ERROR: Frontend directory not found at $FRONTEND_DIR${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}All prerequisites satisfied.${NC}"
}

# Function to build the application
build_application() {
    echo -e "${YELLOW}Building the application for production...${NC}"
    
    # Build the backend
    echo -e "${BLUE}Building backend...${NC}"
    cd "$BACKEND_DIR"
    go build -o app
    if [ $? -ne 0 ]; then
        echo -e "${RED}ERROR: Failed to build backend.${NC}"
        exit 1
    fi
    echo -e "${GREEN}Backend built successfully.${NC}"
    
    # Build the frontend
    echo -e "${BLUE}Building frontend...${NC}"
    cd "$FRONTEND_DIR"
    npm ci --quiet
    npm run build
    if [ $? -ne 0 ]; then
        echo -e "${RED}ERROR: Failed to build frontend.${NC}"
        exit 1
    fi
    echo -e "${GREEN}Frontend built successfully.${NC}"
    
    echo -e "${GREEN}Application built successfully.${NC}"
}

# Function to create or update environment files
setup_environment() {
    echo -e "${YELLOW}Setting up environment...${NC}"
    
    # Create log directory if it doesn't exist
    mkdir -p "$BACKEND_DIR/log"
    echo -e "${GREEN}Log directory created at:${NC} $BACKEND_DIR/log"
    
    # Check if environment file exists, create if not
    ENV_FILE="$BACKEND_DIR/.env.production"
    if [ ! -f "$ENV_FILE" ]; then
        echo -e "${BLUE}Creating production environment file...${NC}"
        cat > "$ENV_FILE" << EOL
DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
PORT=3000
LOG_LEVEL=info
EOL
        echo -e "${GREEN}Created $ENV_FILE${NC}"
    else
        echo -e "${GREEN}Environment file already exists at:${NC} $ENV_FILE"
    fi
}

# Function to run in production mode
run_production() {
    echo -e "${YELLOW}Starting application in production mode...${NC}"
    
    # Export environment variables
    export NODE_ENV=production
    if [ -f "$BACKEND_DIR/.env.production" ]; then
        export $(cat "$BACKEND_DIR/.env.production" | grep -v '^#' | xargs)
    fi
    
    # Start the backend
    cd "$BACKEND_DIR"
    if command_exists pm2; then
        # Using PM2 if available
        echo -e "${BLUE}Starting backend with PM2...${NC}"
        pm2 delete intel-stealer-backend 2>/dev/null || true
        pm2 start ./app --name "intel-stealer-backend"
    else
        # Using nohup as fallback
        echo -e "${BLUE}Starting backend with nohup...${NC}"
        nohup ./app > app.log 2>&1 &
        BACKEND_PID=$!
        echo $BACKEND_PID > "$BACKEND_DIR/backend.pid"
        echo -e "${GREEN}Backend started with PID:${NC} $BACKEND_PID"
    fi
    
    # Start the frontend
    cd "$FRONTEND_DIR"
    
    # Check if serve is installed
    if ! command_exists serve; then
        echo -e "${BLUE}Installing serve package...${NC}"
        npm install -g serve
    fi
    
    if command_exists pm2; then
        # Using PM2 if available
        echo -e "${BLUE}Starting frontend with PM2...${NC}"
        pm2 delete intel-stealer-frontend 2>/dev/null || true
        pm2 start "serve -s dist" --name "intel-stealer-frontend"
        
        # Save PM2 configuration
        pm2 save
        
        echo -e "${BLUE}To make the services start on boot:${NC}"
        echo -e "Run: ${CYAN}pm2 startup${NC} and follow the instructions"
    else
        # Using nohup as fallback
        echo -e "${BLUE}Starting frontend with nohup...${NC}"
        nohup npx serve -s dist > frontend.log 2>&1 &
        FRONTEND_PID=$!
        echo $FRONTEND_PID > "$FRONTEND_DIR/frontend.pid"
        echo -e "${GREEN}Frontend started with PID:${NC} $FRONTEND_PID"
        
        echo -e "\n${YELLOW}To stop the services, run:${NC}"
        echo -e "kill \$(cat $BACKEND_DIR/backend.pid) \$(cat $FRONTEND_DIR/frontend.pid)"
    fi
    
    echo -e "\n${GREEN}Application is now running in production mode!${NC}"
    echo -e "${CYAN}Frontend available at:${NC} http://localhost:3000"
    echo -e "${GREEN}Backend API available at:${NC} http://localhost:3000/api/hello"
}

# Main script execution
echo -e "${YELLOW}=== Intel Stealer Production Startup ===${NC}"
echo -e "${YELLOW}=======================================${NC}"

# Run the functions
check_prerequisites
setup_environment
build_application
run_production
