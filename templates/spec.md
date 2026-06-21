# 국악 템플릿 사양 (즉시 작곡)

"새 악보 → 템플릿 선택 → 바로 작곡"을 위해 동봉할 `.mscz` 템플릿 명세.
각 템플릿은 **편성 + 장단(장구 보표) + 조선 시리즈 음색 사전 연결 + 합리적 기본
스타일(조표/박자/템포)** 을 갖춘다.

> `.mscz`는 zip(압축 XML) 포맷이라 손으로 만들기 어렵다. **포크 빌드 후 앱에서
> 각 편성을 만들어 "템플릿으로 저장" → `share/templates/`에 커밋**한다. 이 문서는
> 그 작성 기준이다. (악기 id는 `overlay/share/instruments/sojeong_gukak.xml` 기준)

## 공통 규칙
- 파트마다 믹서에서 Decent Sampler(조선 시리즈) 음색을 연결(`docs/decent-sampler-josun.md`).
- 장구 보표는 `plugins/jangdan` 패턴으로 채울 수 있게 빈 장단 1주기를 예시로 배치.
- 기본 박자/템포는 장단에 맞춤(아래). 조표는 곡에 맞게 비워두거나 평조/계면조 관습.

## 템플릿 목록

### 1) 산조 (독주 + 장단)
- 편성: `sojeong-gayageum-sanjo`(또는 거문고/대금/해금 택1) + `sojeong-janggu`
- 장단: 진양조 → 중모리 → 중중모리 → 자진모리(악장 진행). 시작 템플릿은 **중모리**.
- 박자/템포: 중모리 12/4, ♩≈90.

### 2) 정악 합주 (관현악 축소)
- 편성: `sojeong-piri` ×2, `sojeong-daegeum`, `sojeong-haegeum`,
  `sojeong-gayageum-jeongak`, `sojeong-geomungo`, `sojeong-ajaeng`, `sojeong-janggu`
- 박자/템포: 한 박이 긴 정악 관습(예: 20/♩ 매우 느림). 기본 6/4, ♩≈60.

### 3) 사물놀이 (타악 4)
- 편성: `sojeong-kkwaenggwari`, `sojeong-jing`, `sojeong-janggu`, `sojeong-buk`
- 장단: 굿거리/자진모리. 기본 **자진모리** 12/8, ♩.≈100.

### 4) 시나위 (즉흥 합주)
- 편성: `sojeong-daegeum`, `sojeong-piri`, `sojeong-haegeum`, `sojeong-ajaeng`,
  `sojeong-gayageum-sanjo`, `sojeong-janggu`
- 장단: 살풀이/자진모리 계열. 기본 굿거리 12/8, ♩.≈80.

### 5) 가야금 + 장구 (레슨/스케치)
- 편성: `sojeong-gayageum-sanjo` + `sojeong-janggu`
- 가장 가벼운 시작 템플릿. 굿거리 12/8.

## 작성 체크리스트(빌드 후)
- [ ] 각 편성 새 악보 생성(국악기 그룹에서 선택)
- [ ] 파트 순서/이름(한글) 확인, 장구 보표에 장단 1주기 예시 입력
- [ ] 믹서에서 조선 시리즈 음색 연결 후 재생 확인
- [ ] "템플릿으로 저장" → `share/templates/<이름>.mscz` 커밋
- [ ] 새 악보 마법사 템플릿 목록에 노출되는지 확인
