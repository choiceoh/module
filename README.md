# sojeongcompose

**창작국악 작곡가를 위한 AI 작곡 보조.**

Finale로 작업하는 창작국악 작곡가가, **Finale에서 내보낸 MusicXML**을 넣고 질문하면
다음 **선율·화성·시김새·장단 아이디어**를 받는 도구. AI는 **OpenAI 호환 API**
(GLM·OpenAI·로컬 등)를 쓴다.

> 창작국악 = 서양 화성·작곡 기법을 배운 작곡가가 오선보·평균율 위에서 국악기·국악
> 어법(장단·평조/계면조·시김새)을 살려 작곡하는 현대 한국 창작음악.

---

## 두 가지 실행 방법

### A. 독립 실행파일 (권장 — CORS 문제 없음)
`main.go` 가 `index.html` 을 내장해 localhost로 띄우고 **백엔드가 AI를 호출**한다
(브라우저 직접 호출이 아니라 CORS 차단이 없음). 더블클릭하면 기본 브라우저로 열린다.

빌드:
```bash
go build -o sojeongcompose .          # 현재 OS용
# 크로스컴파일 예) GOOS=windows GOARCH=amd64 go build -o sojeongcompose.exe .
#                 GOOS=darwin  GOARCH=arm64 go build -o sojeongcompose .
#                 GOOS=linux   GOARCH=arm64 go build -o sojeongcompose .
```
실행 후 안내되는 `http://127.0.0.1:PORT` 가 열린다. 창을 닫으면 종료.

> 배포 바이너리는 **서명이 없어** 첫 실행 시 OS 경고가 뜬다: macOS는 우클릭 → 열기
> (또는 `xattr -d com.apple.quarantine <파일>`), Windows는 SmartScreen → 추가 정보
> → 실행. mac/Linux는 `chmod +x <파일>` 필요할 수 있음.

### B. HTML 단독 (`index.html` 더블클릭)
서버 없이 브라우저로 바로 열림. 단, 브라우저가 API를 직접 호출하므로 일부 공급자는
**CORS**로 막을 수 있음(그럴 땐 A를 쓰거나 CORS 허용 공급자/로컬 모델).

---

## 사용 (공통)

1. 실행(A) 또는 `index.html` 열기(B).
2. **AI 설정**: Base URL·모델·키 입력
   (예: GLM `https://api.z.ai/api/paas/v4`, 모델 `glm-4.6`, 본인 키).
   키는 그 브라우저(localStorage)에만 저장된다.
3. **곡 정보**(악기·조·장단) 입력 + **MusicXML(.xml)** 드래그&드롭.
   - Finale: **파일 → MusicXML 내보내기**(압축 안 함) → 생긴 `.xml`을 넣기.
   - ⚠ Finale 전용 **`.musx`는 직접 못 읽음** → 위처럼 MusicXML로 내보내야 함.
4. **질문** 적고 "아이디어 받기" (예: "이 도입부 다음 8마디, 계면조로 해금 음역 안에서").

---

## 특징

- **작곡 지식 내장(자동 적용)**: 펀더멘탈·화성·선율·형식·리듬·국악(+민속) 레퍼런스
  26종을 HTML에 내장해, 질문에 맞는 것만 자동 선별·적용한다(붙여넣을 필요 없음).
  출처 [SJY051/music-composition](https://github.com/SJY051/music-composition)
  (CC BY 4.0, 프롬프트에 출처 자동 포함).
- **악보 맥락 인식**: MusicXML을 브라우저 내장 파서로 읽어 파트·음·조·박자 요약을
  AI에 함께 전달 → 곡에 맞는 답.
- **공급자 독립**: OpenAI 호환이면 무엇이든(GLM·OpenAI·DeepSeek·Ollama 등).

## 메모 / 한계

- 브라우저가 API를 **직접 호출**한다. 일부 공급자는 **CORS**로 막으므로, 그럴 땐
  CORS 허용 공급자(OpenRouter 등)나 **로컬 모델(Ollama)** 을 쓴다.
- 내장 지식 때문에 `index.html`이 ~410KB. 요청당 토큰은 선별·상한으로 관리(저가 모델 권장).
- MusicXML은 `.xml`과 **압축 `.mxl` 모두 지원**(브라우저 내장 해제, 외부 의존성 0). MIDI(.mid)는 아직 미지원.

## 라이선스 / 출처

- 내장 작곡 지식: [SJY051/music-composition](https://github.com/SJY051/music-composition) — **CC BY 4.0**.
