# sojeongcompose

**프로 국악 작곡가를 위한 작곡 환경 — MuseScore 4 하드 포크.**

Finale로 작업하던 국악 작곡가가, 오선보 기반 위에서 **국악기 실제 음색 재생
(조선 시리즈 · Decent Sampler VST3) · 장단 · 국악기 편성 템플릿**을 한 곳에서
쓸 수 있도록, 오픈소스 [MuseScore Studio 4](https://github.com/musescore/MuseScore)
를 포크해 국악에 특화시킨다.

> 이 레포는 **포크 오버레이 + 국악 자산 + 빌드/브랜딩 도구**를 담는다.
> MuseScore 전체 소스를 복사해 두지 않고, 업스트림을 클론한 뒤 이 오버레이를
> 얹는 방식으로 깔끔한 하드 포크를 유지한다(업스트림 업데이트 병합 용이).

---

## 왜 MuseScore 포크인가 (요구사항 매핑)

| 요구사항 | MuseScore 4가 이미 제공 |
|---|---|
| 오픈소스·포크 | GPL-3, GitHub 공개 |
| 크로스플랫폼 | Windows · macOS · Linux |
| Finale 연동 | MusicXML import/export, MIDI |
| **국악기 실제 음색 재생** | **VST3 호스팅** → 조선 시리즈(Decent Sampler)를 믹서에 로드 |
| 국악기 파트/음역/이조 | 커스텀 `instruments.xml` |
| 장단·작업 보조 | QML 플러그인 API |

→ "처음부터 만들기" 대신 **검증된 노테이션 엔진을 재사용**하고, 국악 특화
부분만 얹는다.

---

## 레포 구조 (이 레포 루트)

```
.  (choiceoh/sojeongcompose)
├── README.md                         # (이 문서) 전략·빌드·로드맵
├── overlay/                          # 업스트림 소스 위에 덮어쓸 파일들
│   └── share/instruments/
│       └── sojeong_gukak.xml         # ★ 국악기 정의 (음역·이조·음자리표·폴백음원)
├── branding/
│   └── branding.md                   # 앱 이름/식별자 → sojeongcompose 변경 위치
├── plugins/
│   └── jangdan/                      # 장단 입력 QML 플러그인
│       ├── jangdan.qml
│       └── patterns.js               # 굿거리·자진모리·세마치·진양조 … 패턴 정의
├── docs/
│   ├── decent-sampler-josun.md       # 조선 시리즈(Decent Sampler) VST 연동 가이드
│   ├── instrument-ranges.md          # 국악기 음역/이조 확정용 레퍼런스
│   ├── jangdan.md                    # 장단 드럼셋 매핑(덩·쿵·덕·기덕…) 설계
│   └── finale-to-sojeong.md          # Finale 사용자 적응 치트시트(단축키·워크플로)
├── presets/
│   └── shortcuts/                    # Finale식 단축키 프리셋(finale-like.xml)
├── templates/
│   └── spec.md                       # 국악 편성·장단·음색 템플릿 사양(빌드 후 .mscz 작성)
└── scripts/
    ├── bootstrap.sh                  # 업스트림 클론 + 태그 고정 + 오버레이 적용 (mac/Linux)
    └── bootstrap.ps1                 # 동일 (Windows)
```

---

## 빠른 시작 (포크 부트스트랩)

전제(기본 태그 `v4.7.3` 기준): `git`, **`Qt 6.8+`**, **`cmake`**(Linux ≥3.24,
**macOS ≥3.26**), C++ 컴파일러. 더 낮은 Qt/CMake 를 쓰려면 `UPSTREAM_TAG` 를 해당
버전을 지원하는 옛 태그로 낮춘다(단, Linux VST3 는 4.6.0+ 필요). 자세한 빌드 의존성은
업스트림 [MuseScore 빌드 문서](https://github.com/musescore/MuseScore#building-musescore)
참고.

```bash
# 1) 업스트림 MuseScore 를 작업 폴더에 클론하고 안정 태그로 고정 + 국악 오버레이 적용
#    (이 레포 루트에서 실행)
./scripts/bootstrap.sh            # Windows: pwsh ./scripts/bootstrap.ps1

# 2) 빌드 (예: Linux). ※ Linux VST3(조선 시리즈) 재생은 MuseScore 4.6.0+ 필요
cd .work/MuseScore
cmake -S . -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j

# 3) 국악기 그룹 노출시키기 (아래 둘 중 하나, 필수 단계)
#  (a) 권장: 빌드/설치한 MuseScore Preferences → Score → "Instrument list 2"
#      에 overlay 의 sojeong_gukak.xml 경로를 지정 (XML 수술 불필요)
#  또는
#  (b) overlay 의 <InstrumentGroup> 를 업스트림 share/instruments/instruments.xml
#      에 병합한 뒤 빌드

# 4) 실행 → 새 악보 만들기에서 "국악기(Korean Traditional)" 그룹 확인
```

> ⚠ `bootstrap` 은 `sojeong_gukak.xml` 을 업스트림 트리에 복사할 뿐, 기본 악기
> 목록(`instruments.xml`)에 자동 병합하지는 않는다. 위 3단계(a/b)를 거쳐야 새 악보
> 마법사에 국악기 그룹이 노출된다.

`bootstrap` 스크립트가 하는 일:
1. `musescore/MuseScore` 를 `.work/MuseScore` 로 클론(이미 있으면 fetch).
2. 검증된 릴리스 태그(`UPSTREAM_TAG`)로 체크아웃 후 `sojeongcompose` 브랜치 생성.
3. `overlay/` 의 파일을 업스트림 트리에 복사(국악기 정의 등).
4. `plugins/` 를 사용자 MuseScore 플러그인 폴더로 안내(또는 동봉).

> 빌드는 무겁기 때문에 이 저장소 CI/원격 컨테이너가 아니라 **로컬에서** 수행한다.

---

## 조선 시리즈(Decent Sampler) 음색으로 재생하기

1. [Decent Sampler](https://www.decentsamples.com/product/decent-sampler-plugin/)
   **VST3** 버전을 설치하고, 구매한 **조선 시리즈** 라이브러리를 불러온다.
2. sojeongcompose(포크) 실행 → **믹서**에서 각 국악기 파트의 음원을 해당 조선
   시리즈 VST3 인스턴스로 지정한다.
3. 자세한 단계: [`docs/decent-sampler-josun.md`](docs/decent-sampler-josun.md).

> 라이선스: 조선 시리즈 샘플을 추출·복제·재배포하지 않는다. 본 포크는 VST3
> **호스팅(연결)** 만 사용하므로 원본 음질 그대로, 라이선스 위반 없이 재생한다.

---

## 로드맵

- **M1 — 국악기 편성**: `sojeong_gukak.xml` 국악기 그룹을 포크에 통합, 음역/이조
  확정([`docs/instrument-ranges.md`]). Finale MusicXML 가져오기 검증.
- **M2 — 조선 시리즈 재생**: 믹서에서 국악기↔Decent Sampler VST3 매핑 가이드/프리셋.
- **M3 — 장단**: `plugins/jangdan` QML 플러그인(장단 패턴 삽입) + 장구/북 드럼셋 매핑.
- **M4 — 브랜딩**: 앱 이름·아이콘·기본 템플릿을 sojeongcompose 로([`branding/`]).
- **M-UX — Finale 사용감 + 더 쉬움**: 첫 실행 기본값을 한글 UI·Finale 유사
  입력("Input by Duration")·Finale식 단축키([`presets/shortcuts/`])로. 국악
  편성 템플릿([`templates/spec.md`])으로 즉시 작곡. 적응 안내
  [`docs/finale-to-sojeong.md`].
- **M5 (확장)**: 율명 표기 옵션, 시김새(농현 등) 입력/재생 표현, 정간보 뷰.

---

## 라이선스

MuseScore Studio 는 **GPL-3.0** 이다. 이를 포크해 **배포**하는 경우 파생물도
GPL-3.0 으로 소스를 공개해야 한다(`branding/branding.md`의 상표 가이드 참고:
MuseScore 상표·로고는 제거/교체). 본 디렉터리의 오버레이·플러그인·문서 역시
배포 시 GPL-3.0 을 따른다.
