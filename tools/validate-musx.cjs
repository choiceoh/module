#!/usr/bin/env node
/*
 * .musx 변환기 회귀 검증 하니스 (개발용).
 *
 * index.html 의 디코드/변환 함수를 떼어내, rpatters1/musxdom 의 실제 테스트 샘플
 * (.musx + 레퍼런스 .enigmaxml 쌍 95개)로 검증한다:
 *   1) 디코드:  .musx → EnigmaXML  를 레퍼런스 .enigmaxml 과 byte 비교
 *   2) 변환:    .enigmaxml → MusicXML  변환 시 오류/빈 마디 0 인지
 *
 * 사용법:
 *   git clone --depth 1 https://github.com/rpatters1/musxdom /tmp/musxdom
 *   npm i @xmldom/xmldom            # (2번 변환 테스트에만 필요)
 *   node tools/validate-musx.cjs /tmp/musxdom/tests/data
 *
 * 결과(2025-06 기준): 디코드 95쌍 중 91 byte-완전일치(나머지는 버전 마이그레이션 차이),
 * 변환은 96개 샘플 오류 0·빈 마디 0. 옥타브 기준은 musxdom 의 calcNoteProperties
 * 테스트(enharmonic_unlinked·hidden_keysigs)와 대조해 C4=가온다로 확정.
 */
const fs = require("fs"), path = require("path");

const dataDir = process.argv[2];
if (!dataDir || !fs.existsSync(dataDir)) {
  console.error("사용법: node tools/validate-musx.cjs <musxdom/tests/data 경로>");
  process.exit(2);
}
const html = fs.readFileSync(path.join(__dirname, "..", "index.html"), "utf8");

// index.html 의 인라인 스크립트에서 함수 블록을 anchor 로 떼어낸다.
function slice(from, toBefore) {
  const a = html.indexOf(from);
  const b = html.indexOf(toBefore, a);
  if (a < 0 || b < 0) throw new Error("anchor 를 찾지 못함: " + from);
  return html.slice(a, b);
}
const decodeSrc = slice("function zipList", "// ── EnigmaXML → MusicXML");
// 변환부: mx 헬퍼들 ~ enigmaToMusicXml 끝(중괄호 카운팅으로 함수 끝을 찾는다).
const convSrc = (() => {
  const a = html.indexOf("function mxKids");
  const f = html.indexOf("function enigmaToMusicXml", a);
  let i = html.indexOf("{", f), depth = 0, end = -1;
  for (; i < html.length; i++) { const c = html[i]; if (c === "{") depth++; else if (c === "}") { if (--depth === 0) { end = i + 1; break; } } }
  if (a < 0 || end < 0) throw new Error("convert anchor 실패");
  return html.slice(a, end);
})();

const { musxToEnigmaXml } = new Function(decodeSrc + "\nreturn { musxToEnigmaXml };")();

let DOMParser = null;
try { DOMParser = require("@xmldom/xmldom").DOMParser; } catch { /* 변환 테스트 스킵 */ }
let enigmaToMusicXml = null;
if (DOMParser) {
  global.DOMParser = DOMParser;
  // esc() 는 index.html 의 다른 곳(ABC 변환부)에 정의돼 브라우저에선 스코프에 있다.
  // 추출 평가에선 동등 정의를 주입한다.
  const escDef = 'function esc(s){return String(s).replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;");}\n';
  enigmaToMusicXml = new Function(escDef + convSrc + "\nreturn { enigmaToMusicXml };")().enigmaToMusicXml;
}

(async () => {
  const bases = fs.readdirSync(dataDir).filter(f => f.endsWith(".musx")).map(f => f.slice(0, -5));

  // 1) 디코드 검증
  let exact = 0, diff = 0, derr = 0, pairs = 0;
  for (const b of bases) {
    const ref = path.join(dataDir, b + ".enigmaxml");
    if (!fs.existsSync(ref)) continue;
    pairs++;
    try {
      const buf = fs.readFileSync(path.join(dataDir, b + ".musx"));
      const ab = buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength);
      const got = (await musxToEnigmaXml(ab)).replace(/\r\n/g, "\n").trimEnd();
      const want = fs.readFileSync(ref, "utf8").replace(/\r\n/g, "\n").trimEnd();
      got === want ? exact++ : diff++;
    } catch { derr++; }
  }
  console.log(`디코드: 쌍=${pairs}  byte-완전일치=${exact}  불일치=${diff}  오류=${derr}`);

  // 2) 변환 검증
  if (enigmaToMusicXml) {
    let files = 0, measures = 0, empty = 0, cerr = 0;
    for (const f of fs.readdirSync(dataDir).filter(f => f.endsWith(".enigmaxml"))) {
      files++;
      try {
        const xml = enigmaToMusicXml(fs.readFileSync(path.join(dataDir, f), "utf8")); // 브라우저판은 문자열 입력
        const out = new DOMParser().parseFromString(xml, "text/xml");
        for (const m of Array.from(out.getElementsByTagName("measure"))) { measures++; if (!m.getElementsByTagName("note").length) empty++; }
      } catch { cerr++; }
    }
    console.log(`변환: 파일=${files}  마디=${measures}  빈마디=${empty}  오류=${cerr}`);
  } else {
    console.log("변환 테스트 스킵(@xmldom/xmldom 미설치 — npm i @xmldom/xmldom).");
  }
})();
