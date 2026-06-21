# AI 작곡 보조 (HTML 단일 파일)

설치·빌드 없이 **브라우저로 여는** 창작국악 작곡 보조 도구.
Finale에서 내보낸 **MusicXML(.xml)** 을 넣고 질문하면 다음 선율·화성·시김새
아이디어를 준다. OpenAI 호환 API(GLM·OpenAI·로컬) 사용.

## 사용법
1. `index.html` 을 더블클릭해 브라우저로 연다(또는 GitHub Pages로 호스팅).
2. **AI 설정**: Base URL·모델·키 입력(예: GLM `https://api.z.ai/api/paas/v4`,
   `glm-4.6`). 키는 그 브라우저(localStorage)에만 저장.
3. **곡 정보**(악기·조·장단) 입력 + **MusicXML 파일** 끌어다 놓기.
   - Finale: 파일 → 다른 이름으로 저장 → **MusicXML(압축 안 함 .xml)**.
4. **질문** 적고 "아이디어 받기".

## 메모
- **작곡 지식 내장**: 펀더멘탈·화성·선율·형식·리듬·국악(+민속) 등 핵심 레퍼런스
  26종을 HTML에 내장해, 질문에 맞는 것만 자동 선별·적용한다(붙여넣기 불필요).
  출처 [SJY051/music-composition](https://github.com/SJY051/music-composition)
  (CC BY 4.0, 프롬프트에 출처 자동 포함). "추가 지식"란은 더 넣고 싶을 때만.
- **Finale 파일**: `.musx`(Finale 전용)는 직접 못 읽음 → Finale에서 **MusicXML
  내보내기**로 `.xml` 만들어 넣기. (압축 .mxl·MIDI는 아직 미지원)
- 브라우저가 API를 **직접 호출**한다. 일부 공급자는 **CORS**로 막으므로, 그럴 땐
  CORS 허용 공급자(OpenRouter 등)나 로컬 모델(Ollama)을 쓴다.
- 내장 지식 때문에 HTML이 ~400KB로 커졌고, 요청당 토큰이 늘어난다(질문에 맞는
  것만 보내 상한 적용). GLM 등 저가 모델 권장.

## MuseScore 플러그인과의 관계
- 이 HTML = **자문/아이디어**(빠르게, 설치 없이). 악보에 직접 삽입은 안 함.
- `plugins/ai-compose` = MuseScore **안에서** 생성·삽입(앱 필요).
  둘은 보완 관계: 아이디어는 여기서, 실제 입력은 MuseScore에서.
