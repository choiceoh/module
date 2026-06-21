#!/usr/bin/env bash
#
# sojeongcompose 포크 부트스트랩 (macOS / Linux)
#
#   1) musescore/MuseScore 를 .work/MuseScore 로 클론(또는 fetch)
#   2) 검증된 업스트림 태그로 체크아웃 후 sojeongcompose 브랜치 생성
#   3) overlay/ 의 국악 자산을 업스트림 트리에 복사
#
# 환경변수로 조정:
#   UPSTREAM_TAG  : 고정할 업스트림 릴리스 태그 (기본: 최신 안정 추정값)
#   WORK_DIR      : 작업 폴더 (기본: ./.work)
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

UPSTREAM_URL="https://github.com/musescore/MuseScore.git"
UPSTREAM_TAG="${UPSTREAM_TAG:-v4.4.4}"
WORK_DIR="${WORK_DIR:-${ROOT_DIR}/.work}"
SRC_DIR="${WORK_DIR}/MuseScore"
BRANCH="sojeongcompose"

echo "==> sojeongcompose 부트스트랩"
echo "    업스트림 : ${UPSTREAM_URL} @ ${UPSTREAM_TAG}"
echo "    작업폴더 : ${SRC_DIR}"

mkdir -p "${WORK_DIR}"

if [ ! -d "${SRC_DIR}/.git" ]; then
  echo "==> 클론 중(시간이 걸립니다)..."
  git clone "${UPSTREAM_URL}" "${SRC_DIR}"
else
  echo "==> 기존 클론에서 fetch..."
  git -C "${SRC_DIR}" fetch --tags origin
fi

echo "==> ${UPSTREAM_TAG} 체크아웃 + ${BRANCH} 브랜치 생성"
if git -C "${SRC_DIR}" rev-parse -q --verify "refs/tags/${UPSTREAM_TAG}" >/dev/null; then
  git -C "${SRC_DIR}" checkout -B "${BRANCH}" "tags/${UPSTREAM_TAG}"
else
  echo "    ⚠ 태그 ${UPSTREAM_TAG} 를 찾지 못함 → 기본 브랜치 사용."
  echo "      UPSTREAM_TAG 를 실제 릴리스 태그로 지정하세요. (git -C ${SRC_DIR} tag 로 확인)"
  git -C "${SRC_DIR}" checkout -B "${BRANCH}"
fi

echo "==> overlay/ 적용"
# overlay 트리를 그대로 업스트림 소스에 덮어쓴다.
cp -Rv "${ROOT_DIR}/overlay/." "${SRC_DIR}/"

echo "==> 완료."
echo "    국악기 정의가 다음 위치에 복사됨:"
echo "      ${SRC_DIR}/share/instruments/sojeong_gukak.xml"
echo
echo "  다음 단계:"
echo "   1) 국악기 그룹을 기본 악기 목록에 포함하려면 docs 의 안내대로"
echo "      share/instruments/instruments.xml 에 그룹을 병합하거나,"
echo "      MuseScore Preferences → Score → Instrument list 에서 위 파일을 지정."
echo "   2) 빌드:  cd ${SRC_DIR} && cmake -S . -B build -DCMAKE_BUILD_TYPE=Release && cmake --build build -j"
echo "   3) 장단 플러그인: ${ROOT_DIR}/plugins/jangdan 를 MuseScore Plugins 폴더로 복사"
