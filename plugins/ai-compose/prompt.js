// sojeongcompose — AI 작곡 보조: 프롬프트 빌더
//
// 국악 제약(악기 음역·장단·조)을 프롬프트로 인코딩해 LLM이 그 틀 안에서
// 오선보(평균율) 음표를 생성하게 한다. 출력은 엄격한 JSON으로 강제한다.
.pragma library

// 국악기 음역(MIDI). overlay/share/instruments/sojeong_gukak.xml 과 일치시킨다.
var RANGES = {
  "가야금": [43, 74], "거문고": [36, 67], "해금": [55, 88], "아쟁": [34, 67],
  "대금": [57, 88], "소금": [72, 96], "피리": [55, 82], "단소": [70, 93],
  "태평소": [57, 84], "양금": [55, 88]
};

// 조(調) 설명 — 평조/계면조의 일반적 음 구성(평균율 근사).
var MODES = {
  "평조": "평조: 솔라도레미(장조 느낌)의 5음 음계, 밝고 평온.",
  "계면조": "계면조: 라도레미솔(단조 느낌)의 5음 음계, 애절·구슬픔."
};

// 시스템 프롬프트: 역할·제약·출력 형식.
function system() {
  return [
    "당신은 한국 전통음악(국악) 작곡을 돕는 보조자입니다.",
    "오선보·평균율(12-TET) 기준으로 작곡합니다.",
    "반드시 주어진 악기 음역과 장단(박자), 조(평조/계면조)를 지킵니다.",
    "장단의 강박/맺음 등 한국 음악의 호흡을 고려해 자연스러운 가락을 만듭니다.",
    "",
    "출력은 설명 없이 JSON 객체 하나만:",
    '{ "notes": [ {"dur": 8, "midi": 67}, {"dur": 8, "rest": true} ] }',
    "- dur = 음가 분모(1=온음표,2=2분,4=4분,8=8분,16=16분).",
    "- midi = MIDI 음높이 번호(60=가온다 C4). rest:true 면 쉼표(midi 생략).",
    "- 총 길이가 요청한 마디 수×박자에 맞도록 합니다."
  ].join("\n");
}

// 사용자 프롬프트: 맥락 + 지시.
//   ctx = { instrument, jangdan, timeSig, mode, measures, userText, seedNotes }
function user(ctx) {
  var lines = [];
  var r = RANGES[ctx.instrument];
  lines.push("악기: " + (ctx.instrument || "(미지정)") +
             (r ? " (음역 MIDI " + r[0] + "~" + r[1] + ")" : ""));
  if (ctx.jangdan) lines.push("장단: " + ctx.jangdan);
  if (ctx.timeSig) lines.push("박자: " + ctx.timeSig);
  if (ctx.mode) lines.push("조: " + (MODES[ctx.mode] || ctx.mode));
  if (ctx.measures) lines.push("길이: " + ctx.measures + "마디");
  if (ctx.seedNotes && ctx.seedNotes.length)
    lines.push("이어쓸 기존 가락(MIDI): [" + ctx.seedNotes.join(", ") + "] — 이 뒤를 자연스럽게 잇기.");
  if (ctx.userText) lines.push("요청: " + ctx.userText);
  lines.push("위 제약을 지켜 JSON으로만 답하세요.");
  return lines.join("\n");
}

// skillRef: 선택적 작곡 레퍼런스 본문(예: music-composition 스킬의
// references/genres/korean-traditional.md). 있으면 system 프롬프트에 주입해
// 답변 품질을 쿡북 기반으로 격상한다.
function messages(ctx, skillRef) {
  var sys = system();
  if (skillRef && skillRef.length) {
    sys += "\n\n[참고 자료 — 아래 가이드를 우선 적용해 작곡하라.\n" +
           "출처: SJY051/music-composition (CC BY 4.0)]\n" + skillRef;
  }
  return [
    { role: "system", content: sys },
    { role: "user", content: user(ctx) }
  ];
}
