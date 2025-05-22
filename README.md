# Intel Stealer - Password Database Management System

This full-stack application provides a solution for storing, managing, and searching credential data from log files. The system includes:

- **Frontend**: Vue.js with Vite for a modern, responsive user interface
- **Backend**: Go with Fiber framework providing a robust API
- **Database**: PostgreSQL for reliable data storage
- **Log Processing**: Automatic monitoring and processing of credential log files

## Project Structure

```
hello-world/
├── backend/              # Go Fiber API server
│   ├── go.mod            # Go module definition
│   ├── go.sum            # Go dependency checksums
│   ├── main.go           # Main application entry point
│   ├── logparser.go      # Log file parsing functionality
│   ├── watcher.go        # File system watcher for log files
│   └── log/              # Directory containing log files to process
├── frontend/             # Vue.js frontend
│   ├── index.html        # HTML entry point
│   ├── package.json      # Node.js dependencies
│   ├── vite.config.js    # Vite configuration
│   └── src/              # Source code
│       ├── App.vue       # Root Vue component
│       ├── main.js       # JavaScript entry point
│       └── components/   # Vue components
└── start.ps1             # PowerShell startup script
```

## Features

### Backend (Go Fiber)

- **RESTful API**: JSON-based API for data retrieval and management
- **Pagination**: Built-in pagination support for large datasets
- **Search**: Advanced search capabilities with multiple filter options
- **Log Processing**: Automatic parsing and importing of credential data from logs
- **File Watching**: Real-time monitoring of log directory for new files
- **PostgreSQL Integration**: Efficient database operations with connection pooling

### Frontend (Vue.js)

- **Responsive UI**: Modern interface that works across devices
- **Real-time Updates**: Dynamic display of imported data
- **Search Interface**: User-friendly search with multiple filtering options
- **Pagination Controls**: Easy navigation through large datasets

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/hello` | GET | Simple API health check |
| `/api/entries` | GET | Get credentials with pagination |
| `/api/search` | GET | Search credentials with filters |
| `/api/import-logs` | POST | Trigger log import process |
| `/api/process-file` | POST | Process a specific log file |
| `/api/watcher-status` | GET | Check log watcher status |
| `/api/stats` | GET | Get database statistics |
| `/api/processed-files` | GET | List processed log files |

## Running the Application

### Starting the Application

You can use the included PowerShell script to start both backend and frontend:

```powershell
./start.ps1
```

### Backend (Go Fiber)

```bash
cd backend
go run main.go
```

The backend server will start on http://localhost:3000

### Frontend (Vue.js)

```bash
cd frontend
npm install
npm run dev
```

The frontend development server will start on http://localhost:5173

## Log Processing System

The application includes a sophisticated log processing system:

1. **Log Watcher**: Monitors the `./log` directory for new `.txt` files
2. **Automatic Processing**: New files are automatically detected and processed
3. **Record Tracking**: Processed files are tracked to prevent duplicate entries
4. **Manual Import**: Files can be manually imported through the API

## Database Schema

The application uses PostgreSQL with the following tables:

- **entries**: Stores credential data
  - id (SERIAL PRIMARY KEY)
  - url (TEXT)
  - username (TEXT)
  - password (TEXT)
  - created (TEXT)

- **processed_log_files**: Tracks processed log files
  - id (SERIAL PRIMARY KEY)
  - filename (TEXT UNIQUE)
  - processed_at (TIMESTAMP)
  - entries_added (INT)

## Environment Configuration

The application supports the following environment variables:

- `DATABASE_URL`: PostgreSQL connection string (default: `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`)

## Development

### Backend
- Modify Go API endpoints in `backend/*.go` files
- Run tests with `go test ./...`

### Frontend
- Modify Vue components in the `frontend/src` directory
- Build for production with `npm run build`

## Building for Production

### Frontend
```bash
cd frontend
npm run build
```

The build output will be in the `frontend/dist` directory.

### Backend
```bash
cd backend
go build -o app
```

This will create an executable file that can be deployed to your server.

## Docker Deployment

The application can be easily deployed using Docker and Docker Compose, which includes all required services:
- Backend (Go Fiber API)
- Frontend (Vue.js)
- Nginx (Web Server and Reverse Proxy)
- PostgreSQL (Database)

### Prerequisites

Ensure you have Docker and Docker Compose installed on your system:

```bash
# Installing Docker on Ubuntu/Debian
sudo apt update
sudo apt install docker.io docker-compose

# Start and enable Docker service
sudo systemctl start docker
sudo systemctl enable docker
```

### Deployment Steps

1. Clone the repository:
```bash
git clone https://your-repository-url/intel-stealer.git
cd intel-stealer/hello-world
```

2. Build and start all services:
```bash
docker-compose up -d
```

3. Access the application:
   - Web UI: http://localhost
   - API: http://localhost/api

### Volume Mounts

The Docker Compose configuration includes the following volume mounts:
- PostgreSQL data: Stored persistently on the host
- Backend data directory: Mounted from `./backend/data` on the host to `/app/data` in the container

### Controlling Docker Services

```bash
# View logs
docker-compose logs -f

# View logs for a specific service
docker-compose logs -f backend

# Stop all services
docker-compose down

# Restart a specific service
docker-compose restart backend

# Rebuild and update services after changes
docker-compose up -d --build
```

### Configuration

You can modify environment variables in the `docker-compose.yml` file to customize your deployment.

## Running on Linux

### Installing Dependencies

```bash
# Install PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib

# Start and enable PostgreSQL service
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Install Node.js and npm
sudo apt install nodejs npm

# Install Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc
source ~/.bashrc
```

### Running the Application

```bash
# Create log directory if it doesn't exist
mkdir -p backend/log

# Build and run backend
cd backend
go build -o app
./app
```

In another terminal:
```bash
# Build and serve frontend
cd frontend
npm install
npm run build
npx serve -s dist
```

## Production Mode

### Using the Production Script

The easiest way to run the application in production mode is to use the included script:

```bash
# Make the script executable
chmod +x start_production.sh

# Run the production script
./start_production.sh
```

This script will:
1. Check for required dependencies
2. Set up the environment
3. Build both backend and frontend
4. Start both services in production mode
5. Use PM2 for process management if available

### Manual Configuration

For manual production deployments, follow these steps:

#### Backend Configuration

Create a production environment file (`.env.production`):

```bash
# Create .env.production file
cat > backend/.env.production << EOL
DATABASE_URL=postgres://postgres:securepassword@localhost:5432/intel_stealer?sslmode=disable
PORT=3000
LOG_LEVEL=info
EOL
```

#### Running in Production Mode

```bash
# Start backend in production mode
cd backend
export NODE_ENV=production
export $(cat .env.production | xargs)
./app
```

#### Using Process Manager

For production environments, it's recommended to use a process manager like PM2:

```bash
# Install PM2
npm install -g pm2

# Start backend with PM2
cd backend
pm2 start ./app --name "intel-stealer-backend"

# Start frontend with PM2
cd frontend
pm2 start "npx serve -s dist" --name "intel-stealer-frontend"

# Configure PM2 to start on boot
pm2 startup
pm2 save
```

## License

This project is proprietary software.

*Last updated: May 22, 2025*
