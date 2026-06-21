# AI 작곡 보조 (OpenAI 호환)

`plugins/ai-compose` — 자연어 지시나 선택 구간 이어쓰기로 국악 가락을 생성해
악보에 삽입한다. **공급자 독립**: OpenAI 호환 `/chat/completions` 엔드포인트면
무엇이든 사용(설정에서 base URL·모델·키만 변경).

## 설치
1. `plugins/ai-compose` 폴더를 MuseScore Plugins 폴더로 복사
   (Win: `문서\MuseScore4\Plugins\`, mac: `~/Documents/MuseScore4/Plugins/`).
2. MuseScore **Home → Plugins** 에서 활성화.
3. **Plugins → Composing/arranging tools → "AI 작곡 보조"** 실행.

## 공급자 설정 예시

| 공급자 | Base URL | 모델 예시 |
|---|---|---|
| **GLM (Zhipu)** | `https://api.z.ai/api/paas/v4` (중국: `https://open.bigmodel.cn/api/paas/v4`) | `glm-4.6` |
| OpenAI | `https://api.openai.com/v1` | `gpt-4o-mini` |
| DeepSeek | `https://api.deepseek.com/v1` | `deepseek-chat` |
| 로컬(Ollama) | `http://localhost:11434/v1` | 설치한 모델명 |

> 정확한 URL·모델명은 각 공급자 문서에서 확인. **키는 플러그인 설정(로컬)에만
> 저장**되며 이 레포에는 절대 포함되지 않는다.

## 사용
1. 악기·조(평조/계면조)·장단·박자·마디 수를 입력.
2. **자연어 요청**을 적거나(예: "굿거리 장단으로 해금 8마디, 애절한 계면조"),
   비워두면 **선택 구간을 이어쓰기** 한다.
3. "생성·삽입" → AI가 JSON 음표를 반환하면 커서 위치부터 삽입.

## 동작 방식
- 악기 음역(`overlay/share/instruments/sojeong_gukak.xml` 와 일치)·장단·조를
  프롬프트로 인코딩(`prompt.js`).
- OpenAI 호환 Chat Completions 호출(`provider.js`, `response_format=json_object`).
- 엄격한 JSON(`{notes:[{dur,midi|rest}]}`) 파싱 후 커서 API로 삽입, 음역 밖이면
  옥타브 보정.

## music-composition 스킬과 페어링 (품질 강화)

[SJY051/music-composition](https://github.com/SJY051/music-composition) — LLM의
작곡 응답을 정확·실전적으로 만드는 모듈식 "작곡 쿡북" 스킬(SKILL.md + references/
assets, 에이전트가 필요한 1~3개만 lazy-load). **국악 깊이가 특히 강함**: 시김새
(요성·퇴성·추성)를 센트·밀리초 단위 구현값까지(`references/genres/korean-traditional.md`).
라이선스 **CC BY 4.0**(콘텐츠) → 출처 표기 시 활용 가능.

연동 두 가지 모드:

- **모드 A — 에이전트(native lazy-load)**: SKILL.md를 지원하는 하네스(Claude Code·
  Codex·GLM+툴콜 등)를 백엔드로 두면 스킬을 그대로 설치해 lazy-load.
  `npx skills add SJY051/music-composition`. (작곡 자문 품질 최상, 인프라 필요)
- **모드 B — 프롬프트 주입(이 플러그인)**: 우리 플러그인은 단일 `/chat/completions`
  호출이라 에이전트가 아니다. 따라서 요청에 맞는 **레퍼런스 본문을 시스템 프롬프트에
  주입**한다(스킬 벤치마크의 "system-prompt injection" 모드, 6/8 우세). 국악 요청이면
  `korean-traditional.md`를 주입 → `prompt.js`의 얇은 제약이 **쿡북 기반**으로 격상.

사용법(구현됨 — 벤더링 없이 연결):
1. 스킬 설치: `npx skills add SJY051/music-composition` (또는 레포 클론).
2. 플러그인의 **"스킬 레퍼런스"** 칸에 레퍼런스 파일 경로 지정. 창작국악이면
   `.../references/genres/korean-traditional.md`.
3. "생성·삽입" 시 플러그인이 그 파일을 읽어(`prompt.js`의 `skillRef`) 시스템
   프롬프트에 주입 → 쿡북 기반 출력.

> 레퍼런스를 레포에 벤더링하지 않고 **사용자가 지정한 파일을 런타임에 읽는다**(원작
> 버전 추적·존중). CC BY 4.0 출처 표기는 프롬프트에 자동 포함된다.
> "Composition guidance from SJY051/music-composition (CC BY 4.0)".

## 한계 / 다음 단계
- 국악 전용 학습 모델이 희소해, 범용 LLM + 제약 프롬프트 방식이라 **품질은 반복
  튜닝 필요**(프롬프트·few-shot 예시 보강).
- 오선보·평균율 범위 → 미분음·시김새(농현 등)는 근사. 시김새는 후속(아티큘레이션
  매핑)으로.
- 장치 검증 필요 항목(ai-compose.qml TODO): 네트워크 호출·커서 삽입 거동·점음표
  처리·여러 공급자의 `response_format` 지원 차이.
- 보안: 키는 로컬 저장. 회사/공용 PC에서는 사용 후 키 제거 권장.
