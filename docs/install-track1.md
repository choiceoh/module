# sojeongcompose 설치 가이드 (트랙 1 — 빌드 없이 오늘 바로)

공식 **MuseScore Studio 4**(무료)를 설치하고, 우리 **국악 자산**을 얹어
**창작국악 작곡 환경**을 만든다. **컴파일/빌드 불필요**, 클릭 설치 위주.
Windows·macOS 모두 안내.

> 한 번만 세팅하면 끝. 막히면 husband가 이 문서 단계 번호를 알려주면 도와줄 수 있음.

---

## 0. 준비물
- 인터넷 연결
- (선택) **Decent Sampler** + 구매한 **조선 시리즈** — 실제 국악기 음색용
- (선택) **OpenAI 호환 API 키**(GLM 등) — AI 작곡 보조용

---

## 1. MuseScore Studio 4 설치 (5분)
1. https://musescore.org/download 접속 → 운영체제에 맞는 설치 파일 내려받기.
2. 설치 후 실행. (Finale 사용자라면: MuseScore가 Finale 공식 이주처라 `.musx`/
   MusicXML 가져오기를 지원 — 기존 곡도 불러올 수 있음.)

## 2. 우리 국악 자산 내려받기 (2분)
1. 레포 페이지에서 **Code → Download ZIP** (또는 `git clone`).
2. 압축 풀면 `overlay/`, `plugins/`, `presets/`, `docs/` 등이 보임.
   아래 단계에서 이 폴더들을 사용한다. (편한 곳에 두기, 예: 바탕화면)

## 3. 한글 UI (30초)
- MuseScore 상단 메뉴 **Edit → Preferences → General → Language → 한국어** 선택 →
  MuseScore 재시작.

## 4. 국악기 편성 추가 (2분)
1. **편집 → 환경 설정 → 악보** 탭의 **"악기 목록(Instrument list)"** 칸에,
   내려받은 `overlay/share/instruments/sojeong_gukak.xml` 경로를 지정.
2. MuseScore 재시작.
3. **새 악보 만들기** → 악기 추가 화면에서 **"국악 (Korean Traditional)"** 그룹
   확인(가야금·거문고·해금·대금·피리·장구 등 14종).

## 5. 장단 플러그인 설치 (2분)
1. 내려받은 `plugins/jangdan` 폴더를 **MuseScore Plugins 폴더**로 복사:
   - **Windows**: `C:\Users\<사용자이름>\Documents\MuseScore4\Plugins\`
   - **macOS**: `~/Documents/MuseScore4/Plugins/`
2. MuseScore **홈 → 플러그인**에서 목록 새로고침 후 **활성화**.
3. 사용: **플러그인 → Composing/arranging tools → "장단 입력"** → 굿거리·자진모리·
   세마치·중모리 등 삽입.

## 6. AI 작곡 보조 플러그인 설치 (선택, 3분)
1. `plugins/ai-compose` 폴더를 5번과 같은 Plugins 폴더로 복사 → 활성화.
2. 실행: **플러그인 → Composing/arranging tools → "AI 작곡 보조"**.
3. 설정:
   - **Base URL / 모델 / 키**: 예) GLM = `https://api.z.ai/api/paas/v4`, 모델
     `glm-4.6`, 본인 키. (OpenAI·로컬도 가능 — `docs/ai-compose.md` 참고)
   - (선택) **스킬 레퍼런스**: `npx skills add SJY051/music-composition` 설치 후
     `.../references/genres/korean-traditional.md` 경로 지정 → 창작국악 품질↑.
4. 악기·조·장단·마디 수 입력하고 자연어 요청("계면조로 해금 8마디") → 생성·삽입.
> 키는 이 PC에만 저장됨(공용 PC면 사용 후 키 삭제 권장).

## 7. Finale 단축키 + 입력 방식 (3분)
1. **단축키**: `presets/shortcuts/finale-like.xml` 적용
   - **편집 → 환경 설정 → 단축키**에서 가져오기, 또는 파일을 아래 위치의
     `shortcuts.xml`로 복사(MuseScore 끈 상태):
     - Windows: `%LOCALAPPDATA%\MuseScore\MuseScore4\shortcuts.xml`
     - macOS: `~/Library/Application Support/MuseScore/MuseScore4/shortcuts.xml`
   - 음가 숫자(5=4분음표 등)는 Finale와 동일.
2. **음표 입력 방식**: 음표 입력 시 **"Input by Duration"** 선택 → Finale
   **Speedy Entry**와 거의 동일하게 동작.
3. 적응이 필요할 때: [`docs/finale-to-sojeong.md`](finale-to-sojeong.md) 치트시트.

## 8. 조선 시리즈(Decent Sampler) 음색 연결 (선택, 5분)
1. [Decent Sampler **VST3**](https://www.decentsamples.com/product/decent-sampler-plugin/)
   설치 → **조선 시리즈** 라이브러리 로드.
2. MuseScore에서 VST를 찾도록: **편집 → 환경 설정 → 오디오/MIDI(또는 VST)** 에서
   VST3 폴더 확인.
3. 악보 열고 **보기 → 믹서(F10)** → 각 국악기 파트의 음원을 **Decent Sampler**로
   지정 → 조선 시리즈 악기 로드.
4. 자세히: [`docs/decent-sampler-josun.md`](decent-sampler-josun.md).
> 한 번 연결해 두고 **템플릿으로 저장**하면, 다음부터 "새 악보 → 그 템플릿"으로
> 바로 조선 시리즈 음색으로 시작(자세히 `templates/spec.md`).

## 9. 첫 곡 확인
1. 새 악보 → 국악 편성 선택 → 박자/조 지정.
2. 음표 몇 개 입력(N → 숫자 음가 → 음이름/MIDI) → 재생(Space)으로 소리 확인.
3. 장단 플러그인으로 장구 보표 채우기 → AI로 가락 제안 받아보기.

---

## 부록 — 자주 막히는 곳
- **국악기 그룹이 안 보임**: 4번에서 악기 목록 경로 지정 후 **재시작**했는지,
  새 악보 화면에서 카테고리가 "국악"인지 확인.
- **플러그인이 메뉴에 없음**: Plugins 폴더 위치 정확한지, 홈 → 플러그인에서
  **활성화** 했는지.
- **AI 오류**: Base URL/모델/키 정확한지, 네트워크 되는지. 응답 형식 문제면
  모델을 더 큰 것으로.
- **소리가 서양 악기 같음**: 8번 미적용 상태(GM 폴백). Decent Sampler 연결 필요.
