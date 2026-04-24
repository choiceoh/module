# module-scanner build script
#  - Works even when the project path contains non-ASCII characters
#  - Copies sources to $env:TEMP\module-scanner-build (ASCII), builds there,
#    then copies the resulting .exe back to <project>\build\bin\
#
# Usage (from project root):
#   PowerShell -ExecutionPolicy Bypass -File .\build.ps1

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$src = $PSScriptRoot
$tmp = Join-Path $env:TEMP "module-scanner-build"

Write-Host "[1/5] Cleaning temp build path: $tmp"
if (Test-Path $tmp) { Remove-Item $tmp -Recurse -Force }
New-Item -ItemType Directory -Path $tmp | Out-Null

Write-Host "[2/5] Copying sources (excluding node_modules / build / dist / .git / .claude)"
$robocopyArgs = @(
    $src, $tmp,
    "/E",
    "/XD", "node_modules", "build", "dist", ".git", ".claude",
    "/XF", "*.exe", "*.log",
    "/NFL", "/NDL", "/NJH", "/NJS", "/NP"
)
& robocopy @robocopyArgs | Out-Null
if ($LASTEXITCODE -ge 8) {
    Write-Error "robocopy failed with exit code $LASTEXITCODE"
    exit 1
}

Write-Host "[3/5] Running 'wails build' at ASCII path"
Push-Location $tmp
try {
    & wails build -clean
    $buildExit = $LASTEXITCODE
} finally {
    Pop-Location
}
if ($buildExit -ne 0) {
    Write-Error "wails build failed with exit code $buildExit"
    exit $buildExit
}

Write-Host "[4/5] Copying .exe back to project"
$destDir = Join-Path $src "build\bin"
if (-not (Test-Path $destDir)) { New-Item -ItemType Directory -Path $destDir -Force | Out-Null }
$exeSrc = Join-Path $tmp "build\bin\module-scanner.exe"
if (-not (Test-Path $exeSrc)) {
    Write-Error "Built exe not found at $exeSrc"
    exit 1
}
Copy-Item $exeSrc $destDir -Force

Write-Host "[5/5] Removing temp path"
Remove-Item $tmp -Recurse -Force

$exeDst = Join-Path $destDir "module-scanner.exe"
Write-Host ""
Write-Host "Build succeeded: $exeDst" -ForegroundColor Green
