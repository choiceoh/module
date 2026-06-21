# Finale → sojeongcompose 치트시트

Finale로 작곡하던 분이 **5분 안에 첫 마디**를 입력하도록 돕는 빠른 안내.
sojeongcompose는 MuseScore 4 기반이라 조작이 Finale와 *철학은 같고*(오선보·파트·
MIDI 입력) 일부 키만 다르다. 좋은 소식: **음가 숫자 키는 Finale와 동일**(5=4분음표).

---

## 1. 음표 입력 — Finale Speedy ≈ "Input by Duration"

sojeongcompose 기본 입력 모드는 **Input by Duration**(MuseScore 4.5+)으로,
Finale **Speedy Entry**와 거의 같다:

1. 입력 시작: **N** (또는 음표 입력 버튼)
2. **음가 숫자 선택**(누르고 있는 동안 적용) — Finale와 동일 매핑:

   | 키 | 음가 | 키 | 음가 |
   |---|---|---|---|
   | 1 | 64분 | 5 | **4분** |
   | 2 | 32분 | 6 | 2분 |
   | 3 | 16분 | 7 | 온음표 |
   | 4 | 8분 | . | 점(부점) |

3. **음 지정**: 컴퓨터 키보드 음이름 **A~G**, 또는 **MIDI 건반**을 눌러 미리듣고 확정.
4. 옥타브 이동: **Ctrl+↑/↓**(mac: Cmd). 쉼표: **0**. 붙임줄: **T**. 입력 종료: **Esc**.

> Simple Entry처럼 마우스로 팔레트 고르고 오선보 클릭하는 방식도 그대로 가능하다.

## 2. 자주 쓰는 동작 (Finale → sojeongcompose 키)

| 동작 | Finale(대략) | sojeongcompose(MuseScore) |
|---|---|---|
| 음표 입력 토글 | Speedy: **`** | **N** |
| 음가 선택 | 1~8(숫자) | **1~7**(동일 개념, 5=4분) |
| 위/아래 반음 | ↑/↓ | **↑/↓** |
| 옥타브 위/아래 | Shift+↑/↓ 등 | **Ctrl/Cmd+↑/↓** |
| 점음표 | . | **.** |
| 임시표 ♯/♭ | +/− 등 | **↑/↓**(반음) 또는 툴바 |
| 붙임줄(tie) | T/= | **T** |
| 이음줄(slur) | S | **S** |
| 셋잇단음표 | Ctrl+3 등 | **Ctrl/Cmd+3** |
| 마디 삽입 | — | **Ins**(끝에 추가: **Ctrl/Cmd+B**) |
| 다음/이전 음표 | →/← | **→/←** |
| 실행취소 | Ctrl/Cmd+Z | **Ctrl/Cmd+Z** |

> 정확한 키 충돌·차이는 `presets/shortcuts/finale-like.xml` 프리셋으로 맞춘다
> (Preferences → Shortcuts → 가져오기). 빌드 후 키별 점검표로 확정.

## 3. 국악 작곡 빠른 시작

1. **새 악보 → 국악 템플릿 선택**(산조·정악·사물놀이·시나위 등) → 편성·장단·음색이
   이미 세팅됨(자세히는 `templates/spec.md`).
2. 장단은 **Plugins → Composing/arranging tools → 장단 입력**으로 굿거리·자진모리
   등을 한 번에 삽입(`plugins/jangdan`).
3. 재생 음색은 믹서에서 **조선 시리즈(Decent Sampler VST3)** 연결
   (`docs/decent-sampler-josun.md`).

## 4. Finale와 다른 점(미리 알면 편한 것)

- **자동 배치/간격**: Finale처럼 일일이 끌어 옮기지 않아도 보표가 자동 정렬된다.
- **실시간 미리듣기**: 음 입력·선택 시 바로 소리가 난다(끄기: Edit → Preferences).
- **팔레트(Palettes)**: Finale의 도구/메타툴 대신 좌측 팔레트에서 셈여림·아티큘
  레이션·선 등을 적용(검색창에서 이름으로 빠르게 찾기).
- **MusicXML 왕복**: Finale에서 만든 곡은 **MusicXML로 내보내 가져오기** 하면 그대로
  편집·재생할 수 있다.
