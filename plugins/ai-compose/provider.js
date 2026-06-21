// sojeongcompose — AI 작곡 보조: OpenAI 호환 API 클라이언트
//
// 공급자 독립: base URL / 모델명 / 키만 바꾸면 OpenAI · GLM(Zhipu) · DeepSeek ·
// 로컬(Ollama/LM Studio/vLLM) 등 OpenAI 호환 엔드포인트 어디든 사용.
//   - OpenAI : https://api.openai.com/v1        모델 예: gpt-4o-mini
//   - GLM    : https://api.z.ai/api/paas/v4     모델 예: glm-4.6   (중국: open.bigmodel.cn/api/paas/v4)
//   - 로컬   : http://localhost:11434/v1        모델 예: (설치한 모델)
// ⚠ 정확한 base URL/모델명은 각 공급자 문서에서 확인. 키는 레포에 커밋하지 않는다.
.pragma library

// OpenAI 호환 Chat Completions 호출. QML의 XMLHttpRequest 사용.
// cfg = { baseUrl, apiKey, model }
// onDone(text|null, errString|null) 콜백으로 결과 전달.
function chat(cfg, messages, onDone) {
  try {
    var url = (cfg.baseUrl || "").replace(/\/+$/, "") + "/chat/completions";
    var xhr = new XMLHttpRequest();
    xhr.open("POST", url, true);
    xhr.setRequestHeader("Content-Type", "application/json");
    if (cfg.apiKey) xhr.setRequestHeader("Authorization", "Bearer " + cfg.apiKey);

    xhr.onreadystatechange = function () {
      if (xhr.readyState !== XMLHttpRequest.DONE) return;
      if (xhr.status < 200 || xhr.status >= 300) {
        onDone(null, "HTTP " + xhr.status + ": " + xhr.responseText);
        return;
      }
      try {
        var data = JSON.parse(xhr.responseText);
        var content = data.choices && data.choices[0] &&
                      data.choices[0].message && data.choices[0].message.content;
        if (!content) { onDone(null, "응답에 content가 없습니다."); return; }
        onDone(content, null);
      } catch (e) {
        onDone(null, "응답 파싱 실패: " + e);
      }
    };

    var body = {
      model: cfg.model,
      messages: messages,
      temperature: 0.8,
      // OpenAI 호환: JSON만 받도록 강제(미지원 공급자면 무시될 수 있음 → 파서가 폴백)
      response_format: { type: "json_object" }
    };
    xhr.send(JSON.stringify(body));
  } catch (e) {
    onDone(null, "요청 생성 실패: " + e);
  }
}

// 모델 응답 텍스트에서 음표 JSON을 추출(코드펜스/잡텍스트 섞여도 최대한 복구).
// 기대 형식: { "notes": [ { "dur": 8, "midi": 67 }, { "dur": 8, "rest": true }, ... ] }
//   dur  = 음가 분모(4=4분음표, 8=8분음표 …), midi = MIDI 음높이, rest = 쉼표
function parseNotes(text) {
  if (!text) return null;
  var s = text.trim();
  // ```json ... ``` 펜스 제거
  var fence = s.match(/```(?:json)?\s*([\s\S]*?)```/i);
  if (fence) s = fence[1].trim();
  // 첫 { 부터 마지막 } 까지
  var a = s.indexOf("{"), b = s.lastIndexOf("}");
  if (a >= 0 && b > a) s = s.substring(a, b + 1);
  try {
    var obj = JSON.parse(s);
    if (obj && obj.notes && obj.notes.length) return obj.notes;
  } catch (e) { /* fallthrough */ }
  return null;
}
