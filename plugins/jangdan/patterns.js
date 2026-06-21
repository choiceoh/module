// 장단(長短) 패턴 정의 — sojeongcompose
//
// 각 장단은 박자(time)와, 한 주기(한 장단) 동안의 타점(strokes)으로 기술한다.
// stroke.dur 은 분할 단위(8 = 8분음표, 4 = 4분음표 …) 기준 음가,
// stroke.type 은 장구 주법(아래 STROKE). type "rest" 는 쉼표.
//
// ⚠ 장구 드럼셋 음고 매핑(STROKE → MIDI)은 docs/jangdan.md 에서 확정한다.
//   아래 값은 일반적 General MIDI Percussion 근사 placeholder 이며,
//   포크의 Drumset 정의와 일치시켜 최종 조정한다.
.pragma library

// 장구 주법(구음) → placeholder MIDI 음고
var STROKE = {
  deong: 36,   // 덩 — 북편+채편 동시(합장단)
  kung: 35,    // 쿵 — 북편(왼손)
  deok: 38,    // 덕 — 채편(오른손)
  gideok: 40,  // 기덕 — 겹채(꾸밈)
  deoreoreo: 42 // 더러러러 — 굴림
};

// dur 은 "분음표 분모". 예) 8 → 8분음표, 4 → 4분음표, 2 → 2분음표.
function s(type, dur) { return { type: type, dur: dur }; }

var PATTERNS = {
  // 굿거리 — 12/8, 흥겹고 유연한 4박
  gutgeori: {
    name: "굿거리",
    time: { num: 12, den: 8 },
    strokes: [
      s("deong", 8), s("rest", 8), s("deok", 8),
      s("kung", 8),  s("deok", 8), s("deok", 8),
      s("deong", 8), s("rest", 8), s("deok", 8),
      s("gideok", 8), s("deok", 8), s("deok", 8)
    ]
  },

  // 자진모리 — 12/8, 빠르고 몰아가는 4박
  jajinmori: {
    name: "자진모리",
    time: { num: 12, den: 8 },
    strokes: [
      s("deong", 8), s("deok", 8), s("deok", 8),
      s("kung", 8),  s("deok", 8), s("deok", 8),
      s("deong", 8), s("deok", 8), s("deok", 8),
      s("kung", 8),  s("deok", 8), s("gideok", 8)
    ]
  },

  // 세마치 — 9/8, 3박(아리랑 등)
  semachi: {
    name: "세마치",
    time: { num: 9, den: 8 },
    strokes: [
      s("deong", 8), s("rest", 8), s("deok", 8),
      s("kung", 8),  s("deok", 8), s("rest", 8),
      s("deok", 8),  s("gideok", 8), s("deok", 8)
    ]
  },

  // 중모리 — 12/4, 느긋한 12박
  jungmori: {
    name: "중모리",
    time: { num: 12, den: 4 },
    strokes: [
      s("deong", 4), s("deok", 4), s("deok", 4), s("kung", 4),
      s("deok", 4),  s("rest", 4), s("kung", 4), s("deok", 4),
      s("deok", 4),  s("gideok", 4), s("deok", 4), s("rest", 4)
    ]
  }
};

function list() {
  var out = [];
  for (var key in PATTERNS) out.push({ key: key, name: PATTERNS[key].name });
  return out;
}
