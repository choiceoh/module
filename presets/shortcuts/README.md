# Finale식 단축키 프리셋

`finale-like.xml` — Finale Speedy/Simple Entry에 익숙한 손을 위해 키를 맞춘
MuseScore 단축키 세트. 음가 숫자 키는 MuseScore 기본이 이미 Finale와 같으므로
(5=4분음표) 그것을 명시·고정하고, 나머지 명령을 점진적으로 맞춘다.

## 적용 방법

**방법 A — 파일 교체**: MuseScore를 끈 상태에서 `finale-like.xml`을 설정 폴더의
`shortcuts.xml`로 복사.
- Windows: `%LOCALAPPDATA%\MuseScore\MuseScore4\shortcuts.xml`
- macOS: `~/Library/Application Support/MuseScore/MuseScore4/shortcuts.xml`
- Linux: `${XDG_DATA_HOME:-~/.local/share}/MuseScore/MuseScore4/shortcuts.xml`

**방법 B — 앱에서 가져오기**: Edit → Preferences → Shortcuts 에서 가져오기.

> 포크 빌드에서는 이 파일을 **factory default**로 동봉해 첫 실행부터 적용되게 한다
> (`branding/branding.md`의 기본 환경설정 항목 참고).

## 확정 절차 (빌드 후)

1. sojeongcompose 빌드 실행 → Preferences → Shortcuts.
2. `docs/finale-to-sojeong.md`의 매핑 표대로 비음가 명령(아티큘레이션·이동·삽입
   등)의 키를 점검·보정.
3. 보정 후 **export** 하여 이 `finale-like.xml`을 갱신·커밋.

액션ID는 MuseScore 버전마다 다를 수 있어, 빌드에서 검증한 값으로 확정한다.
