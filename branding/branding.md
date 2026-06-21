# 브랜딩 가이드 — MuseScore → sojeongcompose

MuseScore Studio 는 **GPL-3.0** 이지만 **이름·로고는 상표(trademark)** 로
보호된다. 포크를 배포하려면 MuseScore 상표/로고를 **제거 또는 교체**하고
독자 이름(`sojeongcompose`)으로 빌드해야 한다. (배포 시 소스도 GPL-3.0 공개.)

> 아래 위치/변수명은 대표값이다. 고정한 업스트림 태그에서 정확한 파일을
> `grep` 으로 확인한 뒤 변경한다(버전에 따라 경로·변수명이 다를 수 있음).

## 1. 앱 이름·식별자

- 최상위 `CMakeLists.txt` 의 앱 이름/버전 변수
  (예: `MUSESCORE_NAME`, `MUSE_APP_*`, `MUSESCORE_NAME_VERSION`).
- 앱 셸 문자열: `src/appshell/` 의 제목/About 텍스트.
- 번들 식별자: macOS `Info.plist`(CFBundleName/Identifier),
  Windows 리소스, Linux `.desktop` 파일.

확인 예:
```bash
cd .work/MuseScore
grep -RIn --include=CMakeLists.txt -e "MUSESCORE_NAME" -e "MUSE_APP" .
grep -RIn "MuseScore" src/appshell | head
```

## 2. 로고·아이콘

- 앱 아이콘(`*.icns`, `*.ico`, splash, 트레이/About 로고)을 sojeongcompose
  자산으로 교체. MuseScore 워드마크/로고 이미지는 제거.

## 3. 업데이트·텔레메트리·계정

- 자동 업데이트가 musescore.org 를 가리키면 비활성화하거나 자체 채널로 변경.
- MuseScore 계정/원격 연동 기능은 필요 없으면 비활성화.

## 4. 기본 자산 동봉

- `overlay/share/instruments/sojeong_gukak.xml` 의 국악기 그룹을 기본 악기
  목록(`share/instruments/instruments.xml`)에 병합.
- 국악 편성 기본 템플릿(.mscz)을 `share/templates` 에 추가(후속).

## 5. 라이선스 표기

- About 화면에 "Based on MuseScore Studio (GPL-3.0)" 고지와 소스 위치를 명시.
- 본 저장소의 오버레이/플러그인/문서도 배포 시 GPL-3.0 을 따른다.

## 6. 첫 실행 기본값 (Finale 사용감 · 생산성)

포크가 **첫 실행부터** 다음 기본값을 갖도록 factory default(앱 기본 설정)를 조정.
관련 자산: `presets/shortcuts/finale-like.xml`, `docs/finale-to-sojeong.md`,
`templates/spec.md`.

- **언어 = 한국어(ko)**: 기본 UI 언어를 ko 로. (MuseScore 한글 번역 내장)
- **음표 입력 = "Input by Duration"**(MuseScore 4.5+): Finale Speedy 유사 모드를
  기본 입력 방식으로. + MIDI 입력 활성화.
- **단축키 = Finale식 프리셋**: `finale-like.xml` 을 기본 `shortcuts.xml` 로 동봉
  (config 폴더에 자동 배치되거나 factory default 로 컴파일).
- **기본 템플릿 = 국악 편성**: `share/templates` 의 국악 템플릿(`templates/spec.md`)
  을 새 악보 마법사 상단에 노출.
- **실시간 미리듣기 on**, 자동 저장 on 등 합리적 기본값.

> 설정 위치(버전별 상이): 언어/입력/단축키 factory default 는 업스트림 소스의
> appshell/preferences 영역과 기본 리소스에서 조정한다. 빌드에서 확인 후 확정.

### "Finale 워크스페이스" 기본 프로필
목표는 Finale UI **픽셀 복제(비현실적)**가 아니라, **다시 배울 게 거의 없는 익숙함**.
Finale **표준/기본 설정**을 기준으로 다음을 하나의 기본 워크스페이스로 묶어 첫 실행에
적용한다(최종 사용자가 Finale 내부를 알 필요 없음):
- Finale식 단축키(`presets/shortcuts/finale-like.xml`)
- 음표 입력 = Input by Duration(≈ Speedy Entry)
- 툴바·팔레트 배치, 테마, 한글 UI 를 Finale 기본 화면에 가깝게
- 국악 편성 템플릿을 새 악보 상단에

> 작성 주체: Finale 기본 구성은 **개발자가 보유 지식으로 세팅**(사용자 입력 불필요).
> 미세 조정은 빌드 후 **실사용자(작곡가)** 가 첫 실행에서 어색한 점만 피드백 →
> 워크스페이스/단축키를 보정(앱 내 AI 보조가 "Finale에선 이렇게 했는데?" 질문을 도움).
> MuseScore 워크스페이스는 앱에서 구성→export(.workspace) 해 `presets/` 에 동봉한다.
