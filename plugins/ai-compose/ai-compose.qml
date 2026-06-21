//=============================================================================
//  sojeongcompose — AI 작곡 보조 플러그인 (OpenAI 호환 API)
//
//  자연어 지시 또는 선택 구간 이어쓰기로 가락을 생성해 악보에 삽입한다.
//  공급자 독립: 설정에서 base URL / 모델 / 키만 바꾸면 OpenAI · GLM · 로컬 등 사용.
//
//  설치: 이 폴더(ai-compose)를 MuseScore Plugins 폴더로 복사 → Home → Plugins 에서
//        활성화 → Plugins → Composing/arranging tools → "AI 작곡 보조".
//
//  ⚠ 동작하는 스캐폴드다. 실제 장치(빌드된 sojeongcompose)에서 네트워크·커서 삽입·
//    음역 검증을 확인해 다듬는다(아래 TODO). 키는 로컬에만 저장되며 레포에 없다.
//=============================================================================
import QtQuick 2.9
import QtQuick.Controls 2.2
import Qt.labs.settings 1.0
import FileIO 3.0
import MuseScore 3.0
import "provider.js" as Provider
import "prompt.js" as Prompt

MuseScore {
  version: "0.1.0"
  description: "AI(OpenAI 호환)로 국악 가락을 생성해 삽입합니다."
  title: "AI 작곡 보조 (sojeongcompose)"
  categoryCode: "composing-arranging-tools"
  pluginType: "dialog"
  width: 460
  height: 460

  // 로컬에만 저장되는 설정(레포에 키 미포함)
  Settings {
    id: cfg
    category: "sojeongcompose-ai"
    property string baseUrl: "https://api.z.ai/api/paas/v4"  // GLM 예시
    property string model: "glm-4.6"
    property string apiKey: ""
    property string instrument: "해금"
    property string jangdan: "굿거리"
    property string timeSig: "12/8"
    property string mode: "계면조"
    property int measures: 4
    // music-composition 스킬의 레퍼런스 파일 경로(선택). 예:
    // .../music-composition/references/genres/korean-traditional.md
    property string skillRefPath: ""
  }

  // 선택적 레퍼런스 파일 읽기(MuseScore 내장 FileIO)
  FileIO { id: skillFile }

  Column {
    anchors.fill: parent
    anchors.margins: 12
    spacing: 8

    Text { text: "OpenAI 호환 API로 국악 가락 생성 (GLM/OpenAI/로컬)"; font.bold: true }

    Grid {
      columns: 2; columnSpacing: 8; rowSpacing: 6; width: parent.width
      Text { text: "Base URL" }
      TextField { id: baseUrlF; width: 300; text: cfg.baseUrl; placeholderText: "https://api.z.ai/api/paas/v4" }
      Text { text: "모델" }
      TextField { id: modelF; width: 300; text: cfg.model; placeholderText: "glm-4.6 / gpt-4o-mini …" }
      Text { text: "API 키" }
      TextField { id: keyF; width: 300; text: cfg.apiKey; echoMode: TextInput.Password; placeholderText: "sk-… (로컬 저장)" }
      Text { text: "악기 / 조" }
      Row { spacing: 6
        TextField { id: instF; width: 120; text: cfg.instrument }
        TextField { id: modeF; width: 120; text: cfg.mode; placeholderText: "평조/계면조" }
      }
      Text { text: "장단 / 박자" }
      Row { spacing: 6
        TextField { id: jangF; width: 120; text: cfg.jangdan }
        TextField { id: tsF; width: 120; text: cfg.timeSig; placeholderText: "12/8" }
      }
      Text { text: "마디 수" }
      SpinBox { id: measF; from: 1; to: 64; value: cfg.measures }
      Text { text: "스킬 레퍼런스" }
      TextField { id: skillRefF; width: 300; text: cfg.skillRefPath
        placeholderText: ".../references/genres/korean-traditional.md (선택)" }
    }

    Text { text: "요청(자연어). 비우면 선택 구간을 '이어쓰기' 합니다." }
    TextArea {
      id: promptF; width: parent.width; height: 70
      placeholderText: "예) 굿거리 장단으로 해금 8마디, 애절한 계면조 가락"
    }

    Row {
      spacing: 8
      Button { id: genBtn; text: "생성·삽입"; onClicked: generate() }
      Button { text: "닫기"; onClicked: Qt.quit() }
    }
    Text { id: statusT; text: ""; width: parent.width; wrapMode: Text.WordWrap; color: "#555" }
  }

  // 현재 선택의 마지막 음들을 seed(MIDI 배열)로 수집(이어쓰기용).
  function collectSeed() {
    var seed = [];
    if (!curScore) return seed;
    var c = curScore.newCursor();
    c.rewind(Cursor.SELECTION_START);
    if (!c.segment) return seed;
    var end = curScore.newCursor(); end.rewind(Cursor.SELECTION_END);
    while (c.segment && (!end.segment || c.tick < end.tick)) {
      if (c.element && c.element.type === Element.CHORD) {
        var ns = c.element.notes;
        if (ns && ns.length) seed.push(ns[ns.length - 1].pitch);
      }
      c.next();
    }
    return seed.slice(-8); // 마지막 8음만
  }

  function saveCfg() {
    cfg.baseUrl = baseUrlF.text; cfg.model = modelF.text; cfg.apiKey = keyF.text;
    cfg.instrument = instF.text; cfg.mode = modeF.text;
    cfg.jangdan = jangF.text; cfg.timeSig = tsF.text; cfg.measures = measF.value;
    cfg.skillRefPath = skillRefF.text;
  }

  // 스킬 레퍼런스 파일이 지정돼 있으면 본문을 읽어 반환(없으면 "").
  function loadSkillRef() {
    if (!skillRefF.text) return "";
    skillFile.source = skillRefF.text;
    var t = skillFile.read();
    return t ? t : "";
  }

  function generate() {
    if (!curScore) { statusT.text = "열린 악보가 없습니다."; return; }
    if (!keyF.text) { statusT.text = "API 키를 입력하세요(로컬에만 저장)."; return; }
    saveCfg();
    genBtn.enabled = false;
    statusT.text = "생성 중…";

    var ctx = {
      instrument: instF.text, jangdan: jangF.text, timeSig: tsF.text,
      mode: modeF.text, measures: measF.value, userText: promptF.text,
      seedNotes: promptF.text ? [] : collectSeed()
    };
    var conf = { baseUrl: baseUrlF.text, apiKey: keyF.text, model: modelF.text };

    Provider.chat(conf, Prompt.messages(ctx, loadSkillRef()), function (content, err) {
      if (err) { statusT.text = "오류: " + err; genBtn.enabled = true; return; }
      var notes = Provider.parseNotes(content);
      if (!notes) { statusT.text = "음표 JSON 파싱 실패. 응답: " + content.substring(0, 120); genBtn.enabled = true; return; }
      insertNotes(notes);
      statusT.text = "삽입 완료: " + notes.length + "개";
      genBtn.enabled = true;
    });
  }

  // 생성된 음표를 선택 위치(없으면 곡 처음)부터 삽입.
  function insertNotes(notes) {
    var r0 = RANGES_LOOKUP(instF.text);
    curScore.startCmd();
    var cur = curScore.newCursor();
    cur.rewind(Cursor.SELECTION_START);
    if (!cur.segment) cur.rewind(Cursor.SCORE_START);
    for (var i = 0; i < notes.length; i++) {
      var n = notes[i];
      var dur = n.dur ? n.dur : 4;
      cur.setDuration(1, dur);
      if (n.rest) {
        cur.addRest();
      } else {
        var p = n.midi;
        // 음역 클램프(악기 범위 밖이면 옥타브 보정)
        if (r0) { while (p < r0[0]) p += 12; while (p > r0[1]) p -= 12; }
        cur.addNote(p);
      }
    }
    curScore.endCmd();
  }

  function RANGES_LOOKUP(name) { return Prompt.RANGES[name] || null; }

  onRun: { if (!curScore) { console.log("열린 악보가 없습니다."); } }
}
