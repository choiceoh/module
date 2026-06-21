# sojeongcompose

**창작국악 작곡가를 위한 AI 작곡 보조 (독립 실행파일).**

Finale에서 내보낸 **MusicXML**(`.xml`/`.mxl`)을 넣고 질문하면, 다음 **선율·화성·
시김새·장단 아이디어**를 받는다. AI는 **OpenAI 호환 API**(GLM·OpenAI 등)를 쓰며,
앱 안에 든 작곡 스킬 문서를 **에이전트가 필요한 것만 직접 찾아 읽고** 답한다.
구체적 선율·화성은 **오선보로 그려서** 답한다(AI가 ABC 표기를 내면 [abcjs](https://abcjs.net)
가 브라우저에서 악보로 렌더; 플랫/샵 등 이명동음은 조표에 맞춰 일관 표기).

> 창작국악 = 서양 화성·작곡 기법을 배운 작곡가가 오선보·평균율 위에서 국악기·국악
> 어법(장단·평조/계면조·시김새)을 살려 작곡하는 현대 한국 창작음악.

---

## 구조 (왜 독립 실행파일인가)

`main.go` 가 `index.html`(UI) 과 작곡 스킬 문서 전체(`skill/`)를 바이너리에 내장해
localhost로 띄운다. `/api/chat` 은 **LLM 에이전트 루프**:

1. 모델에게 `read_doc(path)` 도구를 준다(스킬 문서 목록 + SKILL.md 를 시스템 프롬프트에).
2. 모델이 질문을 보고 `references/00-navigation.md` 등 **필요한 문서 1~3개만** 읽는다.
3. 그 가이드에 근거해 답한다. (스킬이 의도한 **네이티브 lazy-loading** — 전부 안 먹임)

→ 브라우저가 아니라 **백엔드가 API를 호출**하므로 CORS 문제가 없고, 지식을 프런트에
욱여넣지 않아 UI가 가볍다.

## 실행

배포된 실행파일(또는 macOS `.app`)을 받아 실행 → 기본 브라우저로 자동으로 열린다.
창(터미널/앱)을 닫으면 종료.

> **미서명** 바이너리라 첫 실행 시 OS 경고: macOS는 `.app` 우클릭 → 열기(막히면
> `xattr -cr <앱>`), Windows는 SmartScreen → 추가 정보 → 실행, mac/Linux 단일
> 바이너리는 `chmod +x` 필요할 수 있음.

직접 빌드:
```bash
go build -o sojeongcompose .                         # 현재 OS
GOOS=windows GOARCH=amd64 go build -o sojeongcompose.exe .
GOOS=darwin  GOARCH=arm64 go build -o sojeongcompose .
GOOS=linux   GOARCH=arm64 go build -o sojeongcompose .
```

## 사용

1. **⚙ AI 설정**(접이식): Base URL·모델·키 (예: GLM `https://api.z.ai/api/paas/v4`,
   `glm-4.6`). 한 번 넣으면 저장됨. **공급자가 도구(tool) 호출을 지원해야 함**(GLM·OpenAI 등).
2. **곡 정보**(악기·조·장단) + **MusicXML**(`.xml`/`.mxl`) 드래그&드롭.
   - Finale: **파일 → MusicXML 내보내기** → `.xml` 또는 `.mxl`. (`.musx`는 전용 포맷이라 불가)
3. **질문** → "아이디어 받기". 결과 하단에 **AI가 읽은 참고 문서**가 표시된다.

## 메모 / 한계

- **도구 호출 미지원 공급자**에선 동작하지 않음(대부분의 OpenAI 호환 API는 지원).
- 멀티턴(도구 호출 1~2회)이라 응답이 단일 호출보다 **조금 느림**.
- MusicXML `.xml`/`.mxl` 지원(브라우저 내장 해제). **MIDI(.mid)는 미지원**, `.musx`도 불가.
- `index.html` 만 브라우저로 직접 열면 AI 호출은 안 됨(백엔드 필요) — **앱으로 실행**.

## 라이선스 / 출처

- `skill/` 의 작곡 지식: [SJY051/music-composition](https://github.com/SJY051/music-composition)
  — **CC BY 4.0**. 에이전트가 읽도록 내장하며 출처를 표기한다(`skill/NOTICE.md`).
