# 내장 작곡 지식 출처

이 `skill/` 디렉터리의 문서 **대부분**은 **SJY051/music-composition** 에서 가져왔습니다.
- 원본: https://github.com/SJY051/music-composition
- 라이선스: **CC BY 4.0** (콘텐츠). 출처 표기 하에 사용·재배포.
- sojeongcompose는 이 문서들을 에이전트(LLM)가 필요한 것만 읽도록 도구로 제공합니다.

## sojeongcompose 원본 추가분 (SJY051 자료 아님)
다음은 sojeongcompose가 공개된 학술·사전 정보를 근거로 직접 조사·작성한 원본입니다:
- `references/genres/korean-creative-art-music.md` (+ `references/genres/creative-art-music/`)
  — 예술음악으로서의 창작국악 작곡 실전 지침. 허브 + 상세 7개(선법 화성·시김새
  텍스처·장단 형식·국악관현악·기보·진단·계보미학).
- `references/instrument-idiom/korean/` — 국악기 15종 주법·특성 개별 문서
  (가야금·거문고·해금·아쟁·양금·대금·소금·단소·피리·태평소·생황·장구·북·
  사물금속·성악) + 인덱스 + **창작·확장 표현기법 개요**(extended-techniques.md).
  각 멜로디 악기 문서에 "창작·확장 표현기법" 절 추가. 발음 수단·농현·음역·확장 주법.
  > 확장 주법 부분은 국립국악원 「창작을 위한 국악기 이해와 활용」(공공누리 제4유형:
  > 출처표시·비상업·변경금지) 등을 **출처로 안내**하되 원문을 복제하지 않고, 일반에
  > 통용되는 기법 어휘를 sojeongcompose가 자체 정리(학술 자료 교차확인)했습니다.
- `references/genres/korean-regional-toris.md` — 국악 지역별 어법(토리: 경기·메나리·
  육자배기·수심가·제주 + 풍물 지역 가락) 작곡 실전 지침.
- `references/genres/sanjo/` — 산조 유파(전승 계보)별 어법, 악기별 개별 페이지
  (가야금·거문고·대금·해금·아쟁·피리산조) + 인덱스.
- `references/genres/minyo/` — 토리별 민요 채보 예시, 곡별 개별 페이지(본조아리랑·
  도라지타령·정선아리랑·진도아리랑·수심가·오돌또기) + 인덱스.
- `references/genres/jeongak/` — 정악 어법(만중삭 한배·연음) + 영산회상·수제천 개별
  페이지 + 인덱스.
- `references/genres/sinawi.md` — 시나위(즉흥 합주·헤테로포니·통제된 즉흥) 어법.
- `references/genres/gugak-orchestra.md` — 국악관현악 편성·좌석 배치·악기 역할 표준.
- `references/fundamentals/score-output-and-enharmonic-spelling.md` — ABC 악보 출력
  형식·이명동음(플랫/샵) 표기 규율.
- `references/00-navigation.md` — 창작국악 특화 큐레이션 라우터로 전면 재작성.

## 제3자 라이브러리
- 악보 렌더링: **abcjs** v6.4.4 (https://abcjs.net), **MIT License**. `abcjs-basic-min.js`
  를 그대로 번들(라이선스 배너 보존). ```abc 코드블록을 브라우저에서 오선보로 렌더.

## 큐레이션 (무관 문서 제거)
이 포크는 **예술음악으로서의 창작국악** 작곡 보조에 맞춰, SJY051 원본 중
팝/힙합/EDM/믹싱·트렌드 리서치·작사·릴리스 검증 등 무관한 문서들을 제거했습니다
(에이전트가 혼동하지 않도록). 화성·선율·형식·대위·관현악·기법·이론 치트시트 등
범용 작곡 지식과 한국 전통/창작국악 문서는 유지했습니다.
