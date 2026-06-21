//=============================================================================
//  sojeongcompose — 장단(長短) 입력 플러그인
//
//  선택한 타악기 보표(장구 파트)에 굿거리·자진모리·세마치·중모리 등
//  장단 패턴을 지정한 마디 수만큼 삽입한다.
//
//  설치: 이 폴더(jangdan)를 MuseScore Plugins 폴더로 복사 → Home → Plugins 에서
//        활성화 후, Plugins → Composing/arranging tools → "장단 입력 (sojeongcompose)"
//        로 실행. (categoryCode 가 Composing/arranging tools 하위 메뉴에 등록)
//
//  ⚠ 장구 드럼셋 음고 매핑은 patterns.js 의 STROKE 값과 포크의 Drumset 정의를
//    일치시켜야 한다(docs/jangdan.md). 본 파일은 동작하는 스캐폴드이며,
//    실제 장치에서 음고 매핑·커서 거동을 검증해 확정한다.
//=============================================================================
import QtQuick 2.9
import QtQuick.Controls 2.2
import MuseScore 3.0
import "patterns.js" as Jangdan

MuseScore {
  version: "0.1.0"
  description: "선택한 장구 보표에 장단 패턴을 삽입합니다."
  title: "장단 입력 (sojeongcompose)"
  // MuseScore 4.4+ 규격: menuPath 대신 title + categoryCode + pluginType 사용.
  // categoryCode 유효값: composing-arranging-tools / color-notes / playback / lyrics
  categoryCode: "composing-arranging-tools"
  pluginType: "dialog"
  width: 320
  height: 200

  property var patternList: Jangdan.list()

  Column {
    anchors.fill: parent
    anchors.margins: 12
    spacing: 10

    Text { text: "장단을 선택하고 삽입할 마디 수를 정하세요."; wrapMode: Text.WordWrap; width: parent.width }

    Row {
      spacing: 8
      Text { text: "장단:"; anchors.verticalCenter: parent.verticalCenter }
      ComboBox {
        id: jangdanBox
        width: 160
        model: patternList.map(function (p) { return p.name; })
      }
    }

    Row {
      spacing: 8
      Text { text: "마디 수:"; anchors.verticalCenter: parent.verticalCenter }
      SpinBox { id: cycleBox; from: 1; to: 64; value: 4 }
    }

    Row {
      spacing: 8
      Button { text: "삽입"; onClicked: insertJangdan() }
      Button { text: "닫기"; onClicked: Qt.quit() }
    }
  }

  // 선택 위치(또는 곡 처음)부터 장단을 cycleBox.value 마디만큼 삽입한다.
  function insertJangdan() {
    if (!curScore) { Qt.quit(); return; }

    var chosen = Jangdan.PATTERNS[patternList[jangdanBox.currentIndex].key];
    var cycles = cycleBox.value;

    curScore.startCmd();
    var cursor = curScore.newCursor();
    cursor.rewind(Cursor.SELECTION_START);
    if (!cursor.segment) cursor.rewind(Cursor.SCORE_START); // 선택 없으면 처음부터

    for (var c = 0; c < cycles; c++) {
      for (var i = 0; i < chosen.strokes.length; i++) {
        var st = chosen.strokes[i];
        cursor.setDuration(1, st.dur); // 1/dur 음표(예: 1/8 = 8분음표)
        if (st.type === "rest") {
          cursor.addRest(); // setDuration 으로 정한 음가만큼 쉼표 삽입
        } else {
          cursor.addNote(Jangdan.STROKE[st.type]);
        }
      }
    }

    curScore.endCmd();
    Qt.quit();
  }

  onRun: {
    if (!curScore) {
      console.log("열린 악보가 없습니다.");
      Qt.quit();
    }
  }
}
