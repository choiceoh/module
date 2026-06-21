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
- 브라우저가 API를 **직접 호출**한다. 일부 공급자는 **CORS**로 브라우저 호출을
  막으므로, 그럴 땐 CORS 허용 공급자(OpenRouter 등)나 로컬 모델(Ollama)을 쓴다.
- 압축 MusicXML(.mxl)·MIDI는 아직 미지원(우선 .xml). 추후 확장 가능.
- "추가 지식"란에 [SJY051/music-composition](https://github.com/SJY051/music-composition)
  의 국악 레퍼런스를 붙여넣으면 품질↑(CC BY 4.0, 출처 표기 자동 포함).

## MuseScore 플러그인과의 관계
- 이 HTML = **자문/아이디어**(빠르게, 설치 없이). 악보에 직접 삽입은 안 함.
- `plugins/ai-compose` = MuseScore **안에서** 생성·삽입(앱 필요).
  둘은 보완 관계: 아이디어는 여기서, 실제 입력은 MuseScore에서.
