# Navigation — sojeongcompose (창작국악 특화 큐레이션)

이 스킬은 **예술음악으로서의 창작국악** 작곡 보조에 맞춰 정리됐다(팝/힙합/EDM/
믹싱/트렌드리서치 등 무관 문서는 제거). 질문을 보고 **관련 문서 1~3개만** 골라
`read_doc(path)` 로 읽어라. 전부 읽지 말 것 — 5개 이상이면 질문이 너무 넓다.

기본 맥락: 별도 언급이 없으면 **창작국악(국악기 예술음악)** 으로 간주한다.

> **경로 규칙(중요):** `read_doc` 의 path 는 **제공된 문서 목록에 적힌 그대로**의
> 상대경로여야 한다 — 즉 `references/...` 또는 `assets/...` 로 시작한다. 아래 표의
> 경로를 **그대로 복사**해 쓰라(`references/` 를 빼면 "문서 없음" 오류가 난다).

## 무엇을 읽을까 (요청 → 문서 경로)

| 요청 유형 | 문서 경로(그대로 사용) |
|---|---|
| **창작국악 작곡(선법 화성화·시김새·장단·국악관현악·실전 처방)** | `references/genres/korean-creative-art-music.md` ★먼저(허브) → 상세는 `references/genres/creative-art-music/`(modal-harmony·sigimsae-texture·jangdan-as-form·gugak-orchestration·notation-for-players·diagnostics·lineage-and-aesthetics) |
| **국악기 주법·특성·편성(가야금/거문고/해금/대금/장구 등)** | `references/instrument-idiom/korean/00-index.md` ★먼저 → 해당 악기 파일 |
| **국악 지역별 어법(토리: 경기·메나리·육자배기·수심가·제주)·향토 색채** | `references/genres/korean-regional-toris.md` |
| **산조 유파(전승 계보)별 어법(가야금·거문고·대금·해금·아쟁산조)** | `references/genres/sanjo/00-index.md` ★먼저 → 악기 파일 |
| **민요 채보 예시(토리별 실제 곡 분석: 아리랑·도라지·진도·수심가·오돌또기)** | `references/genres/minyo/00-index.md` ★먼저 → 곡 파일 |
| 전통 국악 토대(정악·민속·산조·시나위·시김새) | `references/genres/korean-traditional.md` |
| 민속·전통 일반 / 월드 | `references/genres/folk-roots-and-traditions.md`, `references/genres/folk-and-world.md` |
| 서양 예술음악 시대 양식(바로크~현대) | `references/genres/classical-periods.md` |
| 화성 — 기능/모달/반음계/전조/재화성/보이싱 | `references/harmony/functional-harmony.md`, `references/harmony/modal-harmony.md`, `references/harmony/chord-construction.md`, `references/harmony/voice-leading.md`, `references/harmony/chromatic-harmony.md`, `references/harmony/modulation.md`, `references/harmony/reharmonization.md`, `references/harmony/jazz-harmony.md` |
| 선율 — 구성/모티프 전개/악구 | `references/melody/melodic-construction.md`, `references/melody/motivic-development.md`, `references/melody/phrase-structure.md` |
| 형식·전개·트랜지션 | `references/form/classical-forms.md`, `references/form/narrative-and-transitions.md` |
| 리듬·박자·복합박/폴리리듬 | `references/rhythm-groove/rhythmic-devices.md`, `references/rhythm-groove/odd-meters-polyrhythm.md`, `references/fundamentals/rhythm-meter.md` |
| 대위 | `references/counterpoint.md` |
| 분석(기존 곡/내 곡 진단) | `references/analysis.md`, `references/critique-and-feedback.md` |
| 관현악·텍스처·밀도 | `references/orchestration/instruments-ranges-character.md`, `references/orchestration/voicing-and-texture.md`, `references/orchestration/arrangement-density.md`, `references/orchestration/choral-writing.md` |
| 악기 이디엄(서양 현/관/성악 — 국악기는 위 korean/ 우선) | `references/instrument-idiom/overview.md`, `references/instrument-idiom/strings.md`, `references/instrument-idiom/winds.md`, `references/instrument-idiom/vocals.md` |
| 20세기·현대 기법 / 미분음 / 주제와 변주 / 제약·불확정성 / 알고리즘 | `references/techniques/20th-century-techniques.md`, `references/techniques/microtonal.md`, `references/techniques/theme-and-variation.md`, `references/techniques/constraint-based-composition.md`, `references/techniques/algorithmic-and-AI-assisted.md` |
| 음정·음계·기보 기초 / 가사·운율 | `references/fundamentals/pitch-intervals-scales.md`, `references/fundamentals/notation-and-conventions.md`, `references/fundamentals/prosody-and-language.md` |
| 피드백·수정 루프·협업 방식 | `references/creative-workflows/revision-and-feedback-loops.md`, `references/creative-workflows/musical-brainstorming.md`, `references/creative-workflows/answer-calibration.md`, `references/creative-workflows/user-agent-collaboration.md` |
| 교육·워크플로·참고문헌 | `references/teaching-composition.md`, `references/workflow.md`, `references/source-bibliography.md` |

## 즉시 인용 가능한 치트시트 (assets/)
케이던스 `assets/cadence-reference.md` · 음계·음정 공식 `assets/intervals-and-scale-formulas.md`
· 모드 `assets/modes-cheatsheet.md` · 음도 표기 `assets/scale-degree-spelling-cheatsheet.md`
· 진행 카탈로그 `assets/progressions-catalog.md` · 형식 템플릿 `assets/form-templates.md`
· 코드기호 `assets/chord-symbol-conventions.md` · 진단 체크리스트 `assets/diagnostic-checklists.md`
· 이론 감수 루브릭 `assets/music-theory-audit-rubric.md` · 브레인스토밍 카드 `assets/musical-brainstorming-cards.md`
· 응답 템플릿 `assets/response-templates.md` · 세션 브리프 `assets/session-brief-and-decision-log.md`

> 참고문서(references/)는 인용하지 말고 **종합**해서 답하라. 치트시트(assets/)는
> 짧으니 직접 인용해도 된다. **국악기 주법·편성을 말하기 전엔 반드시**
> `references/instrument-idiom/korean/` 의 해당 악기 문서를 확인하라(발음 수단·농현
> 방식 혼동 금지). 창작국악 질문은 일반 이론 문서와
> `korean-creative-art-music.md`·`korean-traditional.md` 를 **함께** 보는 경우가 많다.
