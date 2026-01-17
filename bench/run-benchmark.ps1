<# 
.SYNOPSIS
    Run k6 Benchmark (Windows PowerShell)

.DESCRIPTION
    Wrapper script to run k6 benchmarks with proper environment setup.

.PARAMETER Script
    Name of the benchmark script to run (default: redirect-latency.js)

.PARAMETER BaseUrl
    Base URL of the Penshort service (default: http://localhost:8080)

.PARAMETER ApiKey
    API key for authenticated benchmarks (optional)

.PARAMETER Output
    Output format: json, csv, or none (default: none)

.PARAMETER Duration
    Override test duration (e.g., "30s", "5m")

.EXAMPLE
    .\run-benchmark.ps1 -Script redirect-latency.js
    .\run-benchmark.ps1 -Script api-create-link.js -ApiKey "sk_abc123" -Output json
#>

param(
    [string]$Script = "redirect-latency.js",
    [string]$BaseUrl = "http://localhost:8080",
    [string]$ApiKey = "",
    [ValidateSet("json", "csv", "none")]
    [string]$Output = "none",
    [string]$Duration = "",
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"

# Resolve script path
$ScriptPath = Join-Path $PSScriptRoot "scripts" $Script
if (-not (Test-Path $ScriptPath)) {
    Write-Error "Script not found: $ScriptPath"
    exit 1
}

# Check k6 is installed
if (-not (Get-Command k6 -ErrorAction SilentlyContinue)) {
    Write-Error @"
k6 is not installed. Install via:
  winget install k6 --source winget
  # or
  choco install k6
  
Download: https://k6.io/docs/get-started/installation/
"@
    exit 1
}

Write-Host "============================================="
Write-Host "Penshort Benchmark Runner"
Write-Host "============================================="
Write-Host "Script: $Script"
Write-Host "Base URL: $BaseUrl"
Write-Host ""

# Build k6 command
$k6Args = @("run")

# Environment variables
$k6Args += "--env"
$k6Args += "BASE_URL=$BaseUrl"

if ($ApiKey) {
    $k6Args += "--env"
    $k6Args += "API_KEY=$ApiKey"
}

# Duration override
if ($Duration) {
    $k6Args += "--duration"
    $k6Args += $Duration
}

# Dry run (minimal execution)
if ($DryRun) {
    $k6Args += "--vus"
    $k6Args += "1"
    $k6Args += "--iterations"
    $k6Args += "10"
}

# Output format
if ($Output -eq "json") {
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $outputFile = Join-Path $PSScriptRoot "results" "$([System.IO.Path]::GetFileNameWithoutExtension($Script))-$timestamp.json"
    
    # Ensure results directory exists
    $resultsDir = Join-Path $PSScriptRoot "results"
    if (-not (Test-Path $resultsDir)) {
        New-Item -ItemType Directory -Path $resultsDir | Out-Null
    }
    
    $k6Args += "--out"
    $k6Args += "json=$outputFile"
    Write-Host "Output: $outputFile"
}

# Add script path
$k6Args += $ScriptPath

Write-Host ""
Write-Host ">>> Running: k6 $($k6Args -join ' ')"
Write-Host ""

# Execute k6
& k6 @k6Args
$exitCode = $LASTEXITCODE

Write-Host ""
if ($exitCode -eq 0) {
    Write-Host "============================================="
    Write-Host "Benchmark completed successfully"
    Write-Host "============================================="
} elseif ($exitCode -eq 99) {
    Write-Host "=============================================" -ForegroundColor Yellow
    Write-Host "Benchmark completed with THRESHOLD FAILURES" -ForegroundColor Yellow
    Write-Host "=============================================" -ForegroundColor Yellow
    Write-Host "Review the summary above for failed thresholds."
} else {
    Write-Host "=============================================" -ForegroundColor Red
    Write-Host "Benchmark FAILED (exit code: $exitCode)" -ForegroundColor Red
    Write-Host "=============================================" -ForegroundColor Red
}

exit $exitCode
