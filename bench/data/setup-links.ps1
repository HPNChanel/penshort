<# 
.SYNOPSIS
    Setup Test Links for Benchmarking (Windows PowerShell)

.DESCRIPTION
    Creates 1000 links for cache miss testing.
    Short codes are saved for use by k6 scripts.

.PARAMETER ApiKey
    API key for authentication (required)

.PARAMETER BaseUrl
    Base URL of the Penshort service (default: http://localhost:8080)

.PARAMETER Count
    Number of links to create (default: 1000)

.EXAMPLE
    .\setup-links.ps1 -ApiKey "sk_abc123"
    .\setup-links.ps1 -ApiKey "sk_abc123" -BaseUrl "http://localhost:8080" -Count 500
#>

param(
    [Parameter(Mandatory=$true)]
    [string]$ApiKey,
    
    [string]$BaseUrl = "http://localhost:8080",
    
    [int]$Count = 1000
)

$ErrorActionPreference = "Stop"
$OutputFile = "$env:TEMP\bench-codes.txt"

Write-Host "============================================="
Write-Host "Penshort Benchmark Data Setup"
Write-Host "============================================="
Write-Host "Base URL: $BaseUrl"
Write-Host "Links to create: $Count"
Write-Host "Output: $OutputFile"
Write-Host ""

# Wait for service readiness
Write-Host ">>> Checking service readiness..."
$ready = $false
for ($i = 1; $i -le 30; $i++) {
    try {
        $response = Invoke-RestMethod -Uri "$BaseUrl/readyz" -Method Get -TimeoutSec 5
        Write-Host "Service is ready"
        $ready = $true
        break
    } catch {
        Write-Host "Waiting for service... ($i/30)"
        Start-Sleep -Seconds 1
    }
}

if (-not $ready) {
    Write-Error "Service not ready after 30 seconds"
    exit 1
}

# Clear previous output
"" | Out-File -FilePath $OutputFile -NoNewline

# Create links with progress
Write-Host ""
Write-Host ">>> Creating $Count links..."
$success = 0
$failed = 0

$headers = @{
    "Authorization" = "Bearer $ApiKey"
    "Content-Type" = "application/json"
}

for ($i = 1; $i -le $Count; $i++) {
    $body = @{
        destination = "https://example.com/bench-$i"
    } | ConvertTo-Json

    try {
        $response = Invoke-RestMethod -Uri "$BaseUrl/api/v1/links" `
            -Method Post `
            -Headers $headers `
            -Body $body `
            -TimeoutSec 10
        
        if ($response.short_code) {
            $response.short_code | Out-File -FilePath $OutputFile -Append
            $success++
        } else {
            $failed++
        }
    } catch {
        $failed++
    }

    # Progress indicator every 100 links
    if ($i % 100 -eq 0) {
        Write-Host "  Created $success/$i links..."
    }
}

Write-Host ""
Write-Host "============================================="
Write-Host "Setup Complete"
Write-Host "============================================="
Write-Host "Created: $success links"
Write-Host "Failed: $failed links"
Write-Host "Output saved to: $OutputFile"
Write-Host ""
Write-Host "To run benchmarks:"
Write-Host "  k6 run bench\scripts\redirect-latency.js"
