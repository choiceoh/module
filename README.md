# 모듈 정보 추출기

모듈 사진/데이터시트 이미지를 업로드하면 로컬 vLLM(Vision 모델)로 사양을 추출해 엑셀로 내보내는 내부 도구.

- 백엔드: Go + chi + excelize
- 프론트: Vite + React + TypeScript + Ant Design
- AI: vLLM (OpenAI 호환 API), 기본 모델 `google/gemma-4-27b-a4b-it`

## 구조

```
backend/   Go API 서버 (포트 8080)
frontend/  Vite 개발 서버 (포트 5173, /api → 8080 프록시)
```

## 실행

### 1) 백엔드

```bash
cd backend
cp .env.example .env   # 필요시 서버 주소/모델명 수정
export $(grep -v '^#' .env | xargs)
go run .
```

기본 엔드포인트:
- `GET  /api/health`
- `POST /api/extract`  — multipart `images` 필드, 여러 파일 허용
- `POST /api/export`   — JSON `{ "modules": [...] }` → xlsx 스트림

### 2) 프론트

```bash
cd frontend
npm install
npm run dev
```

브라우저에서 http://localhost:5173 접속.

## 엑셀 컬럼 규격

| 키 | 라벨 |
|---|---|
| `model_name`    | 모델명 |
| `manufacturer`  | 제조사 |
| `category`      | 카테고리 |
| `voltage_rated` | 정격전압 |
| `current_rated` | 정격전류 |
| `interface`     | 인터페이스 |
| `temp_range`    | 동작온도 |
| `dimensions`    | 치수 |
| `weight`        | 무게 |
| `notes`         | 비고 |

변경하려면 `backend/internal/schema/schema.go` 와 `frontend/src/types.ts` 를 함께 수정.

## vLLM 전제

- OpenAI 호환 `/v1/chat/completions` 엔드포인트 제공
- Vision 입력(`image_url` data URL) 지원
- `guided_json`(JSON schema 가이드) 확장 인식 — 지원 안 해도 `response_format: json_object` 로 동작
