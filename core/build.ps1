# Build script for devterm-core sidecar
# Produces binaries for all target platforms in ../src-tauri/binaries/

$ErrorActionPreference = "Stop"

$targets = @(
    @{ GOOS = "windows"; GOARCH = "amd64"; Triple = "x86_64-pc-windows-msvc"; Ext = ".exe" },
    @{ GOOS = "windows"; GOARCH = "arm64"; Triple = "aarch64-pc-windows-msvc"; Ext = ".exe" },
    @{ GOOS = "darwin";  GOARCH = "amd64"; Triple = "x86_64-apple-darwin"; Ext = "" },
    @{ GOOS = "darwin";  GOARCH = "arm64"; Triple = "aarch64-apple-darwin"; Ext = "" },
    @{ GOOS = "linux";   GOARCH = "amd64"; Triple = "x86_64-unknown-linux-gnu"; Ext = "" },
    @{ GOOS = "linux";   GOARCH = "arm64"; Triple = "aarch64-unknown-linux-gnu"; Ext = "" }
)

$outputDir = Join-Path $PSScriptRoot ".." "src-tauri" "binaries"
New-Item -ItemType Directory -Path $outputDir -Force | Out-Null

foreach ($target in $targets) {
    $env:GOOS = $target.GOOS
    $env:GOARCH = $target.GOARCH
    $env:CGO_ENABLED = "0"

    $outputName = "devterm-core-$($target.Triple)$($target.Ext)"
    $outputPath = Join-Path $outputDir $outputName

    Write-Host "Building $outputName..."
    go build -ldflags="-s -w" -o $outputPath ./cmd/devterm-core/

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to build for $($target.Triple)"
        exit 1
    }
}

# Remove env vars
Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host "All sidecar binaries built successfully."
