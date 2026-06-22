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

### 2단계 — MusicXML 골격 (예정)
EnigmaXML(정규화된 entry/frame/layer 모델)을 MusicXML(마디/성부 중첩 트리)로 재조립.
난이도 순: ① 단성 음높이+리듬 → ② 조표·박자 → ③ 다보표/다악기(part) → ④ 타이.

### 3단계 — 고급 기보 (예정)
다성 레이어(`<backup>`)·잇단음표(`<time-modification>`)·가사·셈여림·아티큘레이션·**타악기**.
`musx2mxl`가 약한 지점(마디 위치·다성·잇단음표·타악기)을 우선 타깃. Finale 자체 MusicXML
내보내기를 정답지로 한 회귀 테스트로 검증.

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

## 상태
- ✅ 1단계 디코딩 구현 — 실제 `.musx`로 **EnigmaXML 추출 검증 필요**(개발 환경에선 실행 불가).
- ⏳ 2·3단계는 1단계가 실제 파일에서 동작 확인된 뒤 착수.
