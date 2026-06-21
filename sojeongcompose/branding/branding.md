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
