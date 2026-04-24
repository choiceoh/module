# 모듈 시리얼 스캐너

모듈 측면 라벨 사진에서 **Code 128 바코드**를 디코드해 시리얼을 추출하고, 수동 보정 후 엑셀로 내보내는 데스크톱 앱.

- 실행 형태: **Wails v2 단일 `.exe`** (Windows WebView2 사용, 런타임 설치 불필요)
- 백엔드: Go + [gozxing](https://github.com/makiuchi-d/gozxing)(바코드) + [imaging](https://github.com/disintegration/imaging)(전처리) + excelize
- 프론트: Vite + React + TypeScript + Ant Design
- 네트워크: **오프라인 동작**. 바코드 디코딩·엑셀 저장 전부 로컬
- 확장: `internal/ai/` 에 vLLM 클라이언트 코드가 남아 있어, 추후 **OCR 폴백**으로 붙일 수 있음 (현재는 미호출)

## 설치 요구사항

1. Go 1.24+
2. Node.js 18+
3. Wails CLI
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```
4. Windows 11 (WebView2 내장). Windows 10이면 Edge WebView2 Runtime 설치 필요.

## 개발 실행

프로젝트 루트에서:

```bash
wails dev
```

- Go 변경 시 자동 리빌드
- 프론트 HMR 포함
- 포트 5173은 Vite 내부용. 앱 창이 직접 열림

## 빌드 (단일 exe)

프로젝트 경로에 **한글/공백이 없는 경우**:

```bash
wails build
```

**한글/공백 경로인 경우** (예: `C:\Users\...\새 폴더\...`):

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

`build.ps1` 은 소스를 `%TEMP%\module-scanner-build` (ASCII)로 복사한 뒤 빌드하고 결과 `.exe` 만 원래 경로의 `build\bin\` 로 가져옵니다.

결과: `build/bin/module-scanner.exe` (약 16MB, 더블클릭 실행).

> **왜 이런 워크어라운드가 필요한가?** Rollup 4의 Windows 네이티브 바인딩이 Node 24 + 비-ASCII 경로 조합에서 `STATUS_STACK_BUFFER_OVERRUN`(0xc0000409)으로 조용히 크래시합니다. 빌드만 ASCII 경로에서 하면 되고, 생성된 `.exe` 는 어디서든 실행됩니다.

## 사용 흐름

1. 앱 실행
2. 모듈 라벨 사진 드래그&드롭 (여러 장 가능)
3. **스캔 실행** — 한 장에 바코드가 여러 개면 **각각 한 행**으로 추가
4. 필요 시 셀 직접 수정, 빈 행 추가, 중복 시리얼 확인
5. **엑셀 저장** — 네이티브 저장 다이얼로그에서 경로 선택 → xlsx 기록

## 엑셀 컬럼

| 키 | 라벨 |
|---|---|
| `filename` | 파일 |
| `serial` | 시리얼 |
| `suffix` | 접미사 |
| `source` | 출처 (barcode/manual/vllm) |
| `notes` | 비고 |

변경하려면 `internal/schema/schema.go` 와 `frontend/src/types.ts` 를 함께 수정.

## 바코드 디코더 동작

1. 이미지 → RGBA 디코드
2. 기본 회전(0/90/180/270°) 시도
3. 실패 시 그레이스케일+대비 조정 후 재시도
4. 한 이미지에서 **여러 바코드**는 `multi.GenericMultipleBarcodeReader`로 순차 검출
5. 모두 실패하면 `error` 필드로 해당 파일만 실패 표시, 다른 파일 처리는 계속

## vLLM 폴백 (미연결, 추후)

`internal/ai/client.go` 는 OpenAI 호환 vLLM Chat Completions 엔드포인트를 호출하는 코드가 **보존**되어 있으나, 현재 `app.go`의 `ScanImage` 경로에서 호출하지 않음.
추후 "바코드 디코딩 실패 시 하단 시리얼 텍스트를 VLM로 읽는 폴백"이 필요하면 여기에 분기 추가.

## 설정/프리셋 저장 위치

`~/.module-scanner/presets.json` 에 자동 생성/저장:
- vLLM Base URL / 모델명 (앱 내 **설정** 버튼에서 편집)
- 마스터 엑셀별 Module type / Cell Type 프리셋
- 최근 프로젝트명 목록 (프로젝트명 입력창 자동완성)

환경변수/`.env` 파일은 사용하지 않음. 배포된 `.exe`를 받은 사용자는 앱 안의 **설정** 버튼만으로 모든 값을 관리.
