# 조선 시리즈(Decent Sampler) 음색으로 재생하기

sojeongcompose(MuseScore 4 포크)는 **VST3 호스팅**으로 국악기 파트를 실제
국악기 음색(QLAUDIO **조선 시리즈**)으로 재생한다. 샘플을 추출/복제하지 않고
**Decent Sampler 플러그인을 연결만** 하므로 원본 음질 그대로이며 라이선스
위반이 없다.

## 1. 준비

1. [Decent Sampler](https://www.decentsamples.com/product/decent-sampler-plugin/)
   **VST3** 버전 설치 (Windows/macOS/Linux 공통).
2. 구매한 **조선 시리즈 플래티넘 번들** 라이브러리를 Decent Sampler 로 불러올 수
   있도록 압축 해제·등록(개별 라이선스 비밀번호는 본인만 보관, 공유 금지).
3. MuseScore 가 VST3 를 찾는 폴더에 Decent Sampler 가 설치됐는지 확인
   ([MuseScore VST 핸드북](https://handbook.musescore.org/sound-and-playback/working-with-vst-and-vsti)).

## 2. 파트에 음색 연결

1. 악보를 열고 **보기 → 믹서**(F10).
2. 각 국악기 파트(예: *가야금*, *대금*, *해금*)의 **음원 슬롯**에서 Muse Sounds
   대신 **Decent Sampler (VST3)** 를 선택.
3. 열린 Decent Sampler 창에서 해당 **조선 시리즈 악기**(가야금/대금/해금 등)를
   로드.
4. 파트별로 1~3 반복. 여러 악기를 동시에 울리려면 파트마다 Decent Sampler
   인스턴스를 각각 둔다(MuseScore 가 파트별 인스턴스를 관리).

## 3. 매핑 권장값

| sojeongcompose 파트(id) | 조선 시리즈에서 로드할 악기 |
|---|---|
| `gayageum-sanjo` / `gayageum-jeongak` | 가야금 |
| `geomungo` | 거문고 |
| `haegeum` | 해금 |
| `ajaeng` | 아쟁 |
| `daegeum` | 대금 |
| `sogeum` | 소금/당적 계열 |
| `piri` | 피리 |
| `danso` | 단소 |
| `taepyeongso` | 태평소 |
| `yanggeum` | 양금 |
| `janggu` / `buk` / `kkwaenggwari` / `jing` | 타악(장구/북/꽹과리/징) |

## 4. 아티큘레이션·시김새(후속)

조선 시리즈가 키스위치/라운드로빈으로 농현·퇴성·추성 등 시김새를 제공하면,
악보의 아티큘레이션을 해당 키스위치로 매핑한다. 이 매핑 프리셋은 M2~M5 단계에서
`overlay/` 의 사운드 매핑으로 추가한다(라이브러리 키스위치 사양 확인 필요).

> 참고: 한 악기 인스턴스는 기본적으로 단음색이므로, 같은 파트 안에서 주법을
> 키스위치로 전환하는 방식이 권장된다.
