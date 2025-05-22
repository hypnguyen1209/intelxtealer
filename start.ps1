# PowerShell script to start both frontend and backend

# Define functions for starting each service
function Start-Backend {
    Write-Host "Starting Go Fiber backend..." -ForegroundColor Green
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PSScriptRoot\backend'; go run main.go"
}

function Start-Frontend {
    Write-Host "Starting Vue.js frontend..." -ForegroundColor Cyan
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PSScriptRoot\frontend'; npm run dev"
}

# Main execution
Write-Host "Starting Hello World App (Vue.js + Go Fiber)" -ForegroundColor Yellow
Write-Host "----------------------------------------" -ForegroundColor Yellow

# Start both services
Start-Backend
Start-Frontend

Write-Host "`nServices started in separate windows. Press Ctrl+C in each window to stop them." -ForegroundColor Yellow
Write-Host "Frontend will be available at: http://localhost:5173" -ForegroundColor Cyan
Write-Host "Backend API will be available at: http://localhost:3000/api/hello" -ForegroundColor Green
