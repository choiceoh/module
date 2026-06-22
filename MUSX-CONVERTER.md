# Finale `.musx` → MusicXML 변환기 (자체 구현)

sojeongcompose 내장 변환기. Finale가 단종(2024)되어 `.musx`를 열 수 없게 된 사용자가
**Finale 없이** 악보를 가져오도록, 브라우저에서 외부 런타임 없이 동작한다.

## 목표
기존 유일한 오픈소스 end-to-end 변환기 `musx2mxl`(Python)는 "기본 기보만, 마디 위치·
음표 배치 보장 안 됨"이고 타악기 버그가 열려 있다. 우리는 **마디·박 타임라인을 처음부터
정확히** 잡고, **검증된 지원 목록**을 갖춘 더 충실한 변환기를 목표로 한다.

## 파이프라인
```
.musx (ZIP) → score.dat → [XOR 난독화 해제] → [gunzip] → EnigmaXML → [의미 변환] → MusicXML
```

### 1단계 — 디코딩 (구현됨, `index.html`)
- `.musx`는 ZIP 컨테이너(`META-INF/container.xml`·`mimetype`·`NotationMetadata.xml`·`score.dat`).
- `score.dat` = **XOR 난독화 + gzip** 된 EnigmaXML.
- 난독화: 32비트 LCG(BSD/glibc `rand`) 키스트림으로 XOR.
  - seed `0x28006D45`, ×`0x41C64E6D`, +`0x3039`(mod 2³²), **131072(0x20000)바이트마다 seed 리셋**.
  - 바이트마다: state 전진 → `upper = state>>16` → `key = (upper + upper//255) & 0xFF` → XOR.
  - JS는 `Math.imul`로 32비트 곱을 정확히 처리(`denigmaDecode`).
- 해제 순서: **난독화 해제 먼저, 그다음 gunzip**(`DecompressionStream("gzip")`).
- 브라우저 내장만 사용: ZIP 파싱(자체)·`DecompressionStream`·`DOMParser`·`Math.imul`.

### 2단계 — MusicXML 골격 (구현됨, `enigmaToMusicXml`)
EnigmaXML(정규화된 entry/frame/layer 모델)을 MusicXML(마디/성부 중첩 트리)로 재조립.
**구현·검증된 것**(실제 musxdom 샘플로 Node 테스트): 단성(레이어1) 음높이+리듬, 쉼표,
음길이(EDU 비트→type+dot), 조표(`keySig>key`=signed byte→fifths), 박자(beats/divbeat),
음자리표(treble/bass), 다보표(part), 타이, 화음, 잇단음표(time-modification, 길이 근사).
`.musx` 드롭 시 디코딩→변환→기존 `parseMusicXml`로 AI 입력(parsedSummary)에 연결 +
MusicXML/EnigmaXML 내려받기.

> ★ **음높이 핵심**: `harmLev`는 **으뜸음(가온다 옥타브) 기준** 변위다(C장조만 C4 기준).
> 절대음 = `harmLev + ((fifths*4)%7+7)%7`(으뜸음 온음계 오프셋). musxdom `Entries.h`
> 주석으로 확인, `tie_across_key` 샘플(G장조 G4 타이)로 검증.

### 3단계 — 다층 레이어 + 정밀 잇단음표 (구현됨)
- **다층 레이어**: `gfhold`의 `frame1~4`를 각각 MusicXML **voice**로, 두 번째 레이어부터
  `<backup>`으로 마디 처음으로 되감아 출력(`renderLayer`). `layers` 샘플에서 v1+v2,
  staff2의 layer3까지 정상 — 2단계에서 비어 나오던 성부가 채워짐.
- **정밀 잇단음표**: 잇단음표 마지막 음에 잔여 길이를 배정해 마디 합이 정확(반올림 누적 제거).
- **빈 마디 보정**: 음표 없는 마디는 마디 전체 쉼표로 — 유효 MusicXML 보장.
- **레이어 내부 voice1/voice2**: 한 프레임 entry 체인에 섞인 `<v2/>` 표시를 분리해 각각
  별도 voice로(layer li → voice `li*2+1`/`li*2+2`). `v1v2tuplets`에서 한 voice로 뭉치던
  15음이 v1[5]+v2[10]으로 정확히 분리됨.
- 실제 musxdom 샘플 **전체 96개로 검증: 빈 마디 0·변환 오류 0**.

**아직(다음)**: 셈여림/가사/아티큘레이션·**타악기**(musx2mxl 버그 지점)·복합박·이조악기·
음역 절대옥타브(실 Finale 대조 필요).

## EnigmaXML 인코딩 메모 (2·3단계용)
- **음길이**: EDU 단위. 4분음표=1024, 온음표=4096. 디코딩: **최상위 set 비트 = 음표 종류,
  그 아래 set 비트 수 = 점(dot)**(예: 점4분 = 1024+512 = 1536).
- **음높이**: 이름이 아니라 `harmLev`(가온다 기준 온음계 변위) + `harmAlt`(조표 대비 반음).
  **조표·음자리표·이조를 통해 계산**해야 step/octave/alter가 나온다(직접 읽기 불가).
- **구조**: Entry(리듬 사건, 음 1~15개, 쉼표=`isNote=false`, prev/next 이중연결리스트) →
  frame(레이어×마디×보표) → `gfhold`가 `frame1`~`frame4`(4레이어) 연결. 레이어 내 voice1/voice2.
- **타이**: `tieStart`/`tieEnd` 불리언. **잇단음표**: `TupletDef`(refNum·refDur / displayNumber·
  displayDuration 비율). **박자**: 실제 vs 표시(display) — 표시용을 인쇄 박자로 사용.

## 출처·라이선스 (모두 MIT — 본 변환기는 이 공개 정보에 근거)
- `rpatters1/denigma` (MIT) — `.musx` 추출/디코딩 알고리즘·상수의 1차 출처.
- `rpatters1/musxdom` (MIT) — EnigmaXML 문서 모델(필드·태그 의미)의 스키마 참고.
- `Project-Attacca/enigmaxml-documentation` (MIT) — 컨테이너 구조 문서.
- `joris-vaneyghen/musx2mxl` (MIT) — 디코딩 상수 교차검증 + 매핑 참고 구현.
- (Finale PDK Framework는 독점 — 참고만, 코드 포팅 금지.)

> 디코딩 상수·바이트 연산은 위 MIT 구현들에서 byte 단위로 교차검증했다. 어떤 MakeMusic
> 독점 소스도 포함하지 않으며, 사용자 본인 파일을 **읽기만** 한다(복제 변조 없음).

## 검증 (실제 `.musx`로 확인 완료)
`rpatters1/musxdom`의 실제 테스트 자산(`.musx` + 레퍼런스 `.enigmaxml` 95쌍, 96개 샘플)으로
`tools/validate-musx.cjs`로 검증:
- **디코딩: 95쌍 중 91개가 레퍼런스와 byte 단위 완전 일치**(나머지 4개는 버전 마이그레이션
  `<mappings>` 차이로 추정, 디코드 오류 0). → 1단계 디코드 알고리즘이 실파일에서 증명됨.
- **변환: 96개 샘플 전부 오류 0·빈 마디 0.**
- **옥타브 기준 확정**: musxdom 의 `calcNoteProperties` 테스트(`enharmonic_unlinked`→oct4,
  `hidden_keysigs`→oct5)와 대조해 **C4=가온다** 확인.
- **이조악기 동작 주의**: 변환기는 **콘서트(실음) 피치**를 낸다(Finale 표기음 아님). 분석엔
  유용하나, 표기음(이조)이 필요하면 staffSpec 이조값 적용이 추가로 필요(미구현).

## 상태
- ✅ 1단계 디코딩 — 구현·**실파일 91/95 byte-일치로 검증**.
- ✅ 2단계 EnigmaXML→MusicXML — 구현 + **실제 musxdom 샘플로 Node 검증**(음높이·음길이·
  조표·다보표·타이·잇단음표). `.musx` → AI 입력 + MusicXML 내려받기 연결.
- ✅ 3단계 다층 레이어 + 레이어 내 v1/v2 + 정밀 잇단음표 — 구현·검증.
- ⏳ 다음: 셈여림·가사·아티큘레이션·타악기·복합박·이조악기(표기음).

### 개발자용 — 회귀 테스트
`tools/validate-musx.cjs` 로 실파일 검증(디코드 byte 비교 + 변환 오류/빈마디 점검):
```
git clone --depth 1 https://github.com/rpatters1/musxdom /tmp/musxdom
npm i @xmldom/xmldom          # 변환 테스트에만 필요
node tools/validate-musx.cjs /tmp/musxdom/tests/data
```
index.html 의 디코드/변환 함수를 anchor 로 떼어 돌린다.
