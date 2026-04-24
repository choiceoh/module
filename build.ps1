# module-scanner build script
#  - Works even when the project path contains non-ASCII characters
#  - Copies sources to $env:TEMP\module-scanner-build (ASCII), builds there,
#    then copies the resulting binaries back to <project>\build\bin\
#  - -nsis   also builds the NSIS installer
#  - -embed  embeds the ~172MB WebView2 offline installer into the NSIS package
#            so recipients without internet can still install (recommended for
#            intranet/air-gapped environments)
#
# Usage (from project root):
#   PowerShell -ExecutionPolicy Bypass -File .\build.ps1
#   PowerShell -ExecutionPolicy Bypass -File .\build.ps1 -nsis
#   PowerShell -ExecutionPolicy Bypass -File .\build.ps1 -nsis -embed

param(
    [switch]$nsis,
    [switch]$embed,
    [switch]$debugBuild
)

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$src = $PSScriptRoot
$tmp = Join-Path $env:TEMP "module-scanner-build"
$webviewCacheDir = "C:\temp\webview2-offline"
$webviewCache = Join-Path $webviewCacheDir "MicrosoftEdgeWebview2Setup.exe"
$webviewOfflineUrl = "https://go.microsoft.com/fwlink/p/?LinkId=2099617"
$webviewOfflineMinSize = 100MB

# Ensure NSIS is on PATH so wails build -nsis can find makensis
$nsisDir = "C:\Program Files (x86)\NSIS"
if ($nsis -and (Test-Path $nsisDir) -and ($env:Path -notlike "*$nsisDir*")) {
    $env:Path = "$env:Path;$nsisDir"
}

if ($embed -and -not $nsis) {
    Write-Error "-embed requires -nsis"
    exit 1
}

if ($embed) {
    if (-not (Test-Path $webviewCache) -or (Get-Item $webviewCache).Length -lt $webviewOfflineMinSize) {
        Write-Host "Downloading WebView2 offline installer (~172MB, cached for future builds)..."
        New-Item -ItemType Directory -Path $webviewCacheDir -Force | Out-Null
        Invoke-WebRequest -Uri $webviewOfflineUrl -OutFile $webviewCache
        $actualSize = (Get-Item $webviewCache).Length
        if ($actualSize -lt $webviewOfflineMinSize) {
            Write-Error "Downloaded WebView2 file is too small ($actualSize bytes). URL may have changed."
            exit 1
        }
        Write-Host "  cached: $webviewCache ($([math]::Round($actualSize / 1MB, 1)) MB)"
    } else {
        Write-Host "Using cached WebView2 offline installer: $webviewCache"
    }
}

Write-Host "[1/6] Cleaning temp build path: $tmp"
if (Test-Path $tmp) { Remove-Item $tmp -Recurse -Force }
New-Item -ItemType Directory -Path $tmp | Out-Null

Write-Host "[2/6] Copying sources (excluding node_modules / build / dist / .git / .claude)"
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

$mode = if ($nsis)  { ' (NSIS installer)' } else { '' }
$mode += if ($embed) { ' + WebView2 offline embedded' } else { '' }
Write-Host "[3/6] Running 'wails build' at ASCII path$mode"
Push-Location $tmp
try {
    $wailsArgs = @("build", "-clean")
    if ($nsis) { $wailsArgs += "-nsis" }
    if ($debugBuild) {
        $wailsArgs += "-debug"
        $wailsArgs += "-devtools"
        $wailsArgs += "-windowsconsole"
    }
    & wails @wailsArgs
    $buildExit = $LASTEXITCODE
} finally {
    Pop-Location
}
if ($buildExit -ne 0) {
    Write-Error "wails build failed with exit code $buildExit"
    exit $buildExit
}

if ($embed) {
    Write-Host "[3b/6] Replacing 1.7MB bootstrapper with 172MB offline installer + rebuilding NSIS"
    $installerDir = Join-Path $tmp "build\windows\installer"
    $bootstrapper = Join-Path $installerDir "tmp\MicrosoftEdgeWebview2Setup.exe"
    Copy-Item $webviewCache $bootstrapper -Force
    Push-Location $installerDir
    try {
        & makensis `
            "/DARG_WAILS_AMD64_BINARY=$tmp\build\bin\module-scanner.exe" `
            "project.nsi"
        $nsisExit = $LASTEXITCODE
    } finally {
        Pop-Location
    }
    if ($nsisExit -ne 0) {
        Write-Error "makensis rebuild failed with exit code $nsisExit"
        exit $nsisExit
    }
}

Write-Host "[4/6] Copying artifacts back to project"
$destDir = Join-Path $src "build\bin"
if (-not (Test-Path $destDir)) { New-Item -ItemType Directory -Path $destDir -Force | Out-Null }

# Copy exe
$exeSrc = Join-Path $tmp "build\bin\module-scanner.exe"
if (-not (Test-Path $exeSrc)) {
    Write-Error "Built exe not found at $exeSrc"
    exit 1
}
Copy-Item $exeSrc $destDir -Force

# Copy installer(s) if -nsis
$installerFiles = @()
if ($nsis) {
    Get-ChildItem -Path (Join-Path $tmp "build\bin") -Filter "*installer*.exe" | ForEach-Object {
        Copy-Item $_.FullName $destDir -Force
        $installerFiles += (Join-Path $destDir $_.Name)
    }
}

Write-Host "[5/6] Removing temp path"
Remove-Item $tmp -Recurse -Force

Write-Host ""
Write-Host "Build succeeded:" -ForegroundColor Green
Write-Host "  exe:       $(Join-Path $destDir 'module-scanner.exe')"
foreach ($f in $installerFiles) {
    $size = [math]::Round((Get-Item $f).Length / 1MB, 1)
    Write-Host "  installer: $f ($size MB)"
}
