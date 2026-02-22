param(
    [string]$OutDir = "dist"
)

$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$targets = @(
    @{ GOOS = "windows"; GOARCH = "amd64"; OUTPUT = "void-windows-amd64.exe" },
    @{ GOOS = "windows"; GOARCH = "arm64"; OUTPUT = "void-windows-arm64.exe" },
    @{ GOOS = "linux"; GOARCH = "amd64"; OUTPUT = "void-linux-amd64" },
    @{ GOOS = "linux"; GOARCH = "arm64"; OUTPUT = "void-linux-arm64" },
    @{ GOOS = "darwin"; GOARCH = "amd64"; OUTPUT = "void-darwin-amd64" },
    @{ GOOS = "darwin"; GOARCH = "arm64"; OUTPUT = "void-darwin-arm64" }
)

foreach ($target in $targets) {
    $env:GOOS = $target.GOOS
    $env:GOARCH = $target.GOARCH

    $outputPath = Join-Path $OutDir $target.OUTPUT
    Write-Host "Building $($target.GOOS)/$($target.GOARCH) -> $outputPath"

    go build -trimpath -ldflags "-s -w" -o $outputPath ./cmd/void
}

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

Write-Host "Release binaries written to $OutDir"
