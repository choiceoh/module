<#
  sojeongcompose 포크 부트스트랩 (Windows / PowerShell)

    1) musescore/MuseScore 를 .work\MuseScore 로 클론(또는 fetch)
    2) 검증된 업스트림 태그로 체크아웃 후 sojeongcompose 브랜치 생성
    3) overlay\ 의 국악 자산을 업스트림 트리에 복사

  사용:  pwsh ./scripts/bootstrap.ps1 [-UpstreamTag v4.4.4] [-WorkDir .\.work]
#>
param(
  [string]$UpstreamTag = "v4.4.4",
  [string]$WorkDir = ""
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir   = Resolve-Path (Join-Path $ScriptDir "..")
if ([string]::IsNullOrEmpty($WorkDir)) { $WorkDir = Join-Path $RootDir ".work" }

$UpstreamUrl = "https://github.com/musescore/MuseScore.git"
$SrcDir = Join-Path $WorkDir "MuseScore"
$Branch = "sojeongcompose"

Write-Host "==> sojeongcompose 부트스트랩"
Write-Host "    업스트림 : $UpstreamUrl @ $UpstreamTag"
Write-Host "    작업폴더 : $SrcDir"

New-Item -ItemType Directory -Force -Path $WorkDir | Out-Null

if (-not (Test-Path (Join-Path $SrcDir ".git"))) {
  Write-Host "==> 클론 중(시간이 걸립니다)..."
  git clone $UpstreamUrl $SrcDir
} else {
  Write-Host "==> 기존 클론에서 fetch..."
  git -C $SrcDir fetch --tags origin
}

Write-Host "==> $UpstreamTag 체크아웃 + $Branch 브랜치 생성"
git -C $SrcDir rev-parse -q --verify "refs/tags/$UpstreamTag" 2>$null
if ($LASTEXITCODE -eq 0) {
  git -C $SrcDir checkout -B $Branch "tags/$UpstreamTag"
} else {
  Write-Host "    ! 태그 $UpstreamTag 를 찾지 못함 → 기본 브랜치 사용."
  Write-Host "      -UpstreamTag 를 실제 릴리스 태그로 지정하세요."
  git -C $SrcDir checkout -B $Branch
}

Write-Host "==> overlay\ 적용"
Copy-Item -Recurse -Force (Join-Path $RootDir "overlay\*") $SrcDir

Write-Host "==> 완료."
Write-Host "    국악기 정의: $SrcDir\share\instruments\sojeong_gukak.xml"
Write-Host ""
Write-Host "  다음 단계:"
Write-Host "   1) instruments.xml 병합 또는 Preferences -> Score -> Instrument list 지정"
Write-Host "   2) cmake -S . -B build -DCMAKE_BUILD_TYPE=Release; cmake --build build -j"
Write-Host "   3) plugins\jangdan 를 MuseScore Plugins 폴더로 복사"
