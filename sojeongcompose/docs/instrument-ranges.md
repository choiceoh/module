# 국악기 음역·이조 레퍼런스 (확정용)

`overlay/share/instruments/sojeong_gukak.xml` 의 `aPitchRange`/`pPitchRange`
(MIDI 번호)와 이조 값을 확정하기 위한 작업 문서. 현재 xml 값은 **합리적
근사값**이며, 유파/악기/주법에 따라 다르므로 프로 작곡가가 검토·확정한다.

MIDI 번호 기준: **60 = 가온다(C4)**, 옥타브당 12.

| 악기(id) | 현재 a-range | 현재 p-range | 비고/확정 필요 |
|---|---|---|---|
| 가야금 산조 `gayageum-sanjo` | 43–69 | 41–74 | 농현 포함 상한, 12현 조율 |
| 정악 가야금 `gayageum-jeongak` | 45–69 | 43–72 | 정악 조율 기준 |
| 거문고 `geomungo` | 36–64 | 34–67 | 괘·문현 저음 |
| 양금 `yanggeum` | 55–84 | 55–88 | 양금 음판 배열 |
| 해금 `haegeum` | 57–84 | 55–88 | 무현(無絃)이라 상한 가변 |
| 아쟁 `ajaeng` | 36–64 | 34–67 | 대아쟁/소아쟁 구분 |
| 대금 `daegeum` | 58–84 | 57–88 | 청 울림역·역취 상한 |
| 소금 `sogeum` | 72–93 | 72–96 | 고음 |
| 피리 `piri` | 57–79 | 55–82 | 향피리/세피리/당피리 구분 |
| 단소 `danso` | 70–90 | 70–93 | |
| 태평소 `taepyeongso` | 58–81 | 57–84 | 호적/날라리 |

## 이조(transpose)

현재는 모두 **실음(concert pitch)** 기준(이조 없음). 다음을 확정한다:
- **대금/피리/소금** 등 관악기를 이조 표기로 다룰지(예: 특정 黃=C 기준 표기).
- 정악/민속악 표기 관습에 따른 기보 기준.

확정 시 xml 에 `<transposeChromatic>`/`<transposeDiatonic>` 을 추가한다(없으면
실음). 예) 장2도 높게 울리는 이조: `<transposeChromatic>2</transposeChromatic>
<transposeDiatonic>1</transposeDiatonic>`.

## musicXMLid (Finale 호환)

Finale 와 MusicXML 을 주고받을 때 악기 식별에 쓰인다. 현재는 근접 표준
sound id(예: `pluck.koto`, `wind.flutes.shakuhachi`)를 사용. 국악기 전용 표준
id 가 없는 경우가 많으므로, Finale 쪽 악기명과의 매핑표를 별도 관리한다.
