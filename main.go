// sojeongcompose — 창작국악 AI 작곡 보조 (독립 실행파일, 에이전트 방식)
//
// index.html(슬림 UI)과 작곡 스킬 문서(skill/)를 바이너리에 embed 한다.
// localhost로 띄우고, /api/chat 에서 LLM 에이전트 루프를 돌린다:
// 모델에게 read_doc 도구를 주어, 질문에 맞는 참고 문서 1~3개만 스스로 읽고 답하게 한다
// (스킬이 의도한 "네이티브 lazy-loading"). 브라우저 직접 호출이 아니라 CORS 없음.
package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed index.html
var indexHTML []byte

//go:embed abcjs-basic-min.js
var abcjsJS []byte

//go:embed all:skill
var skillFS embed.FS

var httpClient = &http.Client{Timeout: 600 * time.Second}

// 에이전트 루프 최대 반복 수(= read_doc 라운드 + 마지막 답). 프로 작업용으로 넉넉히:
// 토리·악기·장르 문서가 작은 단위로 분할돼 있어, 깊은 질문은 여러 문서를 참고해야
// 한다 — 중간에 끊기지 않도록 상한을 높인다. 마지막 반복에서는 도구를 막아 반드시
// 완성된 답을 내게 한다(handleChat 참조). 총 소요는 httpClient 타임아웃(600s)이 가둔다.
const maxIters = 16

// 프런트엔드 → 백엔드 요청(원시 입력; 메시지/지식 구성은 백엔드가 함)
type uiReq struct {
	BaseURL     string    `json:"baseUrl"`
	APIKey      string    `json:"apiKey"`
	Model       string    `json:"model"`
	Intent      string    `json:"intent"`      // 곡 특성·배경(악상) — 참고 맥락
	Constraints string    `json:"constraints"` // 반드시 지켜야 할 제약 — 위반 금지
	Score       string    `json:"score"`       // MusicXML 요약(악기·조표·박자·음 나열)
	Question    string    `json:"question"`
	Extra       string    `json:"extra"`
	History     []histMsg `json:"history"` // 이전 답변·피드백(이어서 다듬기)
}

// 이전 대화 한 턴(역할은 user/assistant)
type histMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
	mux.HandleFunc("/abcjs.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Write(abcjsJS)
	})
	mux.HandleFunc("/api/config", handleConfig)
	mux.HandleFunc("/api/chat", handleChat)

	// 고정 포트로 띄운다 → 브라우저 origin(host:port)이 매 실행·매 빌드마다 같아져
	// localStorage에 저장한 설정(Base URL·모델·API 키)이 그대로 유지된다.
	// 포트가 이미 점유돼 있으면 후보를 차례로 시도하고, 모두 막히면 임의 포트로 폴백.
	ln := listen()
	url := "http://" + ln.Addr().String()
	fmt.Println("sojeongcompose 실행 중 →", url)
	if _, p, _ := net.SplitHostPort(ln.Addr().String()); p != "17324" {
		fmt.Println("⚠ 기본 포트(17324)가 사용 중이라 다른 포트로 실행됨 — 이 경우 저장된 설정이 안 보일 수 있습니다.")
	}
	fmt.Println("(이 창을 닫으면 종료됩니다)")
	go func() { time.Sleep(300 * time.Millisecond); openBrowser(url) }()
	log.Fatal(http.Serve(ln, mux))
}

// listen 은 고정 포트(설정 유지를 위해)를 우선 시도하고, 점유돼 있으면 후보를
// 차례로, 모두 막히면 임의 포트로 연다.
func listen() net.Listener {
	for _, p := range []string{"17324", "27324", "37324"} {
		if l, err := net.Listen("tcp", "127.0.0.1:"+p); err == nil {
			return l
		}
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("포트 열기 실패: %v", err)
	}
	return l
}

// ---- 설정(Base URL·모델·API 키) 앱 설정 파일 저장 ----
// 사용자 설정 디렉터리의 sojeongcompose/config.json 에 저장한다(파일 권한 0600).
// 브라우저 localStorage 와 무관하게 유지되어, 어떤 빌드/브라우저에서도 키가 보존된다.
type appConfig struct {
	BaseURL string `json:"baseUrl"`
	Model   string `json:"model"`
	APIKey  string `json:"apiKey"`
}

var cfgMu sync.Mutex

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		if h, e := os.UserHomeDir(); e == nil {
			dir = h
		} else {
			dir = "."
		}
	}
	return filepath.Join(dir, "sojeongcompose", "config.json")
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	p := configPath()
	switch r.Method {
	case http.MethodGet:
		cfgMu.Lock()
		b, err := os.ReadFile(p)
		cfgMu.Unlock()
		var c appConfig
		if err == nil {
			json.Unmarshal(b, &c)
		}
		writeJSON(w, c)
	case http.MethodPost:
		var c appConfig
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeJSON(w, map[string]any{"error": "bad request: " + err.Error()})
			return
		}
		cfgMu.Lock()
		defer cfgMu.Unlock()
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		b, _ := json.MarshalIndent(c, "", "  ")
		if err := os.WriteFile(p, b, 0600); err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "path": p})
	default:
		http.Error(w, "GET/POST only", 405)
	}
}

// 임베드된 스킬 문서 목록(skill/ 기준 상대경로, .md만)
func docList() []string {
	var out []string
	fs.WalkDir(skillFS, "skill", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") {
			out = append(out, strings.TrimPrefix(p, "skill/"))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// read_doc 도구 실행: skill/ 내부 문서를 읽어 반환(경로 탈출 차단)
func readDoc(rel string) string {
	clean := path.Clean("/" + rel)[1:] // 선행 / 제거 + .. 정리
	b, err := skillFS.ReadFile("skill/" + clean)
	if err != nil {
		return "오류: 그런 문서가 없음 (" + rel + "). list 에 있는 경로만 사용."
	}
	return string(b)
}

func systemPrompt() string {
	skill, _ := skillFS.ReadFile("skill/SKILL.md")
	var b strings.Builder
	b.WriteString("당신은 창작국악(현대 한국 창작음악) 작곡을 돕는 전문 보조자입니다.\n")
	b.WriteString("창작국악은 서양 화성·작곡 기법 위에서 오선보·평균율로, 국악기와 국악 어법(장단,\n")
	b.WriteString("평조/계면조, 시김새: 요성·퇴성·추성 등)을 살려 작곡합니다.\n")
	b.WriteString("규칙: 빈말·과장된 칭찬 금지. 음 이름(C4 등)·화성 진행·도약/순차·장단 정렬·악기 음역·\n")
	b.WriteString("시김새 적용 지점을 구체적으로. 모르면 솔직히. 한국어로 답한다.\n")
	b.WriteString("제약 규칙: 사용자가 '반드시 지켜야 할 제약'을 제시하면 어떤 경우에도 위반하지 마라.\n")
	b.WriteString("(예: 특정 악기 음역, 조/선법 유지, 장단 고정, 마디 수 등) 제안·악보가 그 제약을 모두\n")
	b.WriteString("만족하는지 스스로 점검하라. 제약과 충돌이 불가피하면 먼저 그 사실과 이유를 밝히고\n")
	b.WriteString("제약을 지키는 대안을 제시한다. '곡 특성·배경(악상)'은 참고 맥락이니 융통성 있게 쓴다.\n")
	b.WriteString("피드백 규칙: 이전 답변에 대한 사용자 피드백(수정 요청)이 오면, 그 피드백을 최우선으로\n")
	b.WriteString("반영해 직전 답을 수정·발전시켜라(처음부터 다시 쓰지 말고 이어서 다듬되, 제약은 계속 준수).\n")
	b.WriteString("편성 규칙: 악보 요약에 있는 파트(악기)만 다뤄라. 파트가 하나면 솔로곡이다 —\n")
	b.WriteString("임의로 '가야금1·가야금2'처럼 성부를 늘리지 마라(편성 확장은 사용자가 요청할 때만).\n")
	b.WriteString("악기 규칙: 국악기 주법·특성·편성을 말하기 전에 반드시\n")
	b.WriteString("references/instrument-idiom/korean/00-index.md 와 해당 악기 문서를 읽어\n")
	b.WriteString("발음 수단(가야금=맨손가락/거문고=술대/해금·아쟁=활/대금=입김)과 농현 방식을 확인하라.\n")
	b.WriteString("예: 가야금은 술대를 쓰지 않는다(술대는 거문고 전용). 추측으로 주법을 지어내지 마라.\n\n")
	b.WriteString("악보 출력: 구체적인 선율·화성·진행을 제안할 때는 말로만 하지 말고 ABC 표기로\n")
	b.WriteString("악보를 함께 그려라. ```abc 코드블록에 넣으면 화면에 오선보로 렌더된다(반드시 그 블록 사용).\n")
	b.WriteString("ABC 형식: 헤더 X:1 / M:박자(예 3/4) / L:기본음길이(예 1/8) / K:조표(예 K:Gm, K:D)\n")
	b.WriteString("를 먼저 쓰고 다음 줄에 음표. 음높이는 C D E F G A B(가온음역)·c d e(한 옥타브 위)·\n")
	b.WriteString("C,(한 옥타브 아래), 임시표는 ^(샵)·_(플랫)·=(제자리), 마디는 |, 여러 성부는 V:1 V:2.\n")
	b.WriteString("한 번에 2~8마디로 짧게. 헤더(X·M·L·K)를 빠뜨리면 렌더가 깨진다.\n")
	b.WriteString("이명동음(enharmonic) 표기 규율: 한 악구 안에서 플랫과 샵을 섞지 마라. 플랫 계열\n")
	b.WriteString("조·맥락이면 플랫(_)으로, 샵 계열이면 샵(^)으로 일관되게 적고, 조표(K:)와 어긋나지 않게.\n")
	b.WriteString("음정 철자도 바르게(증2도 vs 단3도 등). 헷갈리면\n")
	b.WriteString("assets/scale-degree-spelling-cheatsheet.md·assets/chord-symbol-conventions.md 를 읽어라.\n")
	b.WriteString("국악 시김새(농현·꺾음·미분음)는 평균율 ABC로 다 못 담는다 — 골격만 악보로 그리고\n")
	b.WriteString("시김새는 글로 지시하라(예: 'A음에 넓고 느린 농현', '도→시 꺾음').\n\n")
	b.WriteString("당신에게는 작곡 스킬 문서를 읽는 read_doc(path) 도구가 있습니다.\n")
	b.WriteString("먼저 references/00-navigation.md 를 읽어 어떤 문서가 필요한지 판단하고,\n")
	b.WriteString("질문에 직접 관련된 문서 1~3개만 읽은 뒤, 그 가이드에 근거해 답하세요\n")
	b.WriteString("(전부 읽지 말 것 — 5개 이상이면 질문이 너무 넓은 것).\n")
	b.WriteString("read_doc 의 path 는 문서 목록에 적힌 그대로(references/... 또는 assets/...) 써라.\n\n")
	b.WriteString("=== 응답 절차 (품질 프로토콜 — 매 답변에 적용) ===\n")
	b.WriteString("1) 요구 분해: 질문이 (a) 사실·이론 질문인지 (b) 창작 과제(선율·화성·편성·장단을 짓거나\n")
	b.WriteString("   고치는 일)인지 (c) 혼합인지 먼저 판별하고, 지켜야 할 제약을 추려라.\n")
	b.WriteString("2) 근거 충분성 점검: 답에 필요한 문서를 충분히 읽었는지 스스로 물어라. 토리·구음·유파·\n")
	b.WriteString("   연도처럼 자료마다 갈리는 사실은 한 문서로 단정하지 말고 관련 문서를 교차 확인하라\n")
	b.WriteString("   (부족하면 더 읽되 5개 초과는 금지 — 그땐 범위를 좁혀 답하라).\n")
	b.WriteString("3) 형식을 요청에 맞춰라 — '항상 장문 조사'가 아니라 요청에 맞는 깊이를 고르는 것이다:\n")
	b.WriteString("   - 창작 과제: 장황한 설명 대신 곧바로 쓸 구체값(음 이름·진행·도약/순차·장단 정렬·\n")
	b.WriteString("     시김새 지점)과 ABC 악보 중심으로, 왜 그 소리가 나는지 근거만 짧게. 보고서처럼 늘이지 마라.\n")
	b.WriteString("   - 사실·이론 질문: 핵심을 먼저 답하고, 근거가 갈리거나 미분음·지역차로 불확실하면\n")
	b.WriteString("     그 사실을 함께 밝혀라. 화려한 단정보다 정확이 우선.\n")
	b.WriteString("4) 겸손·검증 게이트(출력 직전 자문): ① 단일 출처로 단정하거나 과장하지 않았나\n")
	b.WriteString("   ② 모르는/불확실한 것을 솔직히 표시했나 ③ 제약을 모두 만족하나 ④ 빈말·추상론 없이\n")
	b.WriteString("   구체적인가. 하나라도 어기면 고쳐서 출력하라.\n\n")
	b.WriteString("=== 작곡 스킬 핵심 지침 (SKILL.md) ===\n")
	b.Write(skill)
	b.WriteString("\n\n=== 읽을 수 있는 문서 목록 (read_doc 의 path 로 사용) ===\n")
	b.WriteString(strings.Join(docList(), "\n"))
	return b.String()
}

func userPrompt(r uiReq) string {
	var p []string
	if r.Constraints != "" {
		p = append(p, "[반드시 지켜야 할 제약 — 어떤 경우에도 위반 금지]\n"+r.Constraints)
	}
	if r.Intent != "" {
		p = append(p, "곡 특성·배경(악상 — 참고 맥락, 융통성 있게 활용): "+r.Intent)
	}
	if r.Score != "" {
		p = append(p, "악보(MusicXML 요약 — 악기·조표·박자는 여기서 읽어라):\n"+r.Score)
	}
	if r.Extra != "" {
		p = append(p, "[사용자 추가 지식 — 우선 적용]\n"+r.Extra)
	}
	q := r.Question
	if q == "" {
		q = "이 곡의 다음 선율 아이디어를 제안해줘."
	}
	p = append(p, "질문: "+q)
	return strings.Join(p, "\n\n")
}

var tools = []map[string]any{{
	"type": "function",
	"function": map[string]any{
		"name":        "read_doc",
		"description": "작곡 스킬의 참고 문서를 읽는다. path 는 제공된 문서 목록의 상대경로(예: references/genres/korean-traditional.md).",
		"parameters": map[string]any{
			"type":       "object",
			"properties": map[string]any{"path": map[string]any{"type": "string", "description": "문서 상대경로"}},
			"required":   []string{"path"},
		},
	},
}}

// 공급자 응답(에러 파싱용 — 스트리밍 실패 시 본문에서 메시지 추출)
type provResp struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var req uiReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"error": "bad request: " + err.Error()})
		return
	}
	if req.BaseURL == "" || req.Model == "" || req.APIKey == "" {
		writeJSON(w, map[string]any{"error": "Base URL·모델·키가 필요합니다."})
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, map[string]any{"error": "이 서버는 스트리밍을 지원하지 않습니다."})
		return
	}
	// Server-Sent Events 로 진행 상황과 답을 토막내어 보낸다(빈 대기 제거).
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	send := func(v any) { b, _ := json.Marshal(v); fmt.Fprintf(w, "data: %s\n\n", b); flusher.Flush() }

	raw := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
	messages := []json.RawMessage{
		raw(map[string]any{"role": "system", "content": systemPrompt()}),
		raw(map[string]any{"role": "user", "content": userPrompt(req)}),
	}
	// 이전 대화(직전 답변·사용자 피드백)를 이어 붙여 수정·발전을 가능하게 한다.
	for _, m := range req.History {
		if (m.Role == "user" || m.Role == "assistant") && m.Content != "" {
			messages = append(messages, raw(map[string]any{"role": m.Role, "content": m.Content}))
		}
	}
	var readDocs []string

	for i := 0; i < maxIters; i++ {
		payload := map[string]any{
			"model": req.Model, "temperature": 0.8, "messages": messages, "stream": true,
		}
		// 마지막 반복에서는 도구를 빼 모델이 반드시 최종 답을 내게 한다 — 반복 한도에
		// 걸려도 '중단 에러' 대신 지금까지 읽은 문서로 답을 완성하도록(프로 작업 보호).
		if i < maxIters-1 {
			payload["tools"] = tools
			payload["tool_choice"] = "auto"
		}
		body, _ := json.Marshal(payload)
		send(map[string]any{"type": "progress", "msg": "생각 중…"})
		// 최종 답 토큰은 도착하는 대로 delta 로 흘려보낸다. 도구 호출 턴은 보통 content 가
		// 비어 있어 흘려보낼 게 없다.
		content, tcs, err := streamProvider(req, body, func(s string) {
			send(map[string]any{"type": "delta", "text": s})
		})
		if err != nil {
			msg := err.Error()
			hint := ""
			if strings.Contains(msg, "deadline") || strings.Contains(strings.ToLower(msg), "timeout") {
				hint = "\n\n응답이 너무 느려 시간이 초과됐습니다. ① 잠시 후 다시 시도, ② 질문을 더 좁게, " +
					"③ Base URL 엔드포인트가 맞는지 확인하세요(예: https://api.z.ai/api/paas/v4). " +
					"공급자가 혼잡하거나 모델이 매우 길게 답하는 경우 발생할 수 있습니다."
			}
			send(map[string]any{"type": "error", "error": "공급자 호출 실패: " + msg + hint})
			return
		}
		if len(tcs) == 0 { // 도구 호출이 없으면 = 최종 답(이미 delta 로 전송됨)
			send(map[string]any{"type": "done", "content": content, "docs": readDocs})
			return
		}
		// assistant 메시지(tool_calls 포함)를 재구성해 재투입
		var tcList []map[string]any
		for _, t := range tcs {
			tcList = append(tcList, map[string]any{
				"id": t.ID, "type": "function",
				"function": map[string]any{"name": t.Name, "arguments": t.Args},
			})
		}
		messages = append(messages, raw(map[string]any{
			"role": "assistant", "content": content, "tool_calls": tcList,
		}))
		// 도구 호출 처리(읽는 문서를 실시간 진행 표시)
		for _, t := range tcs {
			var args struct {
				Path string `json:"path"`
			}
			json.Unmarshal([]byte(t.Args), &args)
			docContent := readDoc(args.Path)
			readDocs = append(readDocs, args.Path)
			send(map[string]any{"type": "progress", "msg": "문서 읽는 중", "doc": args.Path})
			messages = append(messages, raw(map[string]any{
				"role": "tool", "tool_call_id": t.ID, "content": docContent,
			}))
		}
	}
	send(map[string]any{"type": "error", "error": "도구 호출이 너무 많아 중단(질문을 좁혀보세요)."})
}

// 누적되는 도구 호출(스트리밍 델타로 조각조각 도착)
type tcAccum struct{ ID, Name, Args string }

// 공급자 스트리밍 청크(OpenAI 호환 SSE delta 형식)
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// streamProvider 는 stream:true 로 공급자를 호출하고, content 델타를 onContent 로
// 흘려보내며, 누적된 (content, tool_calls) 를 돌려준다.
func streamProvider(req uiReq, body []byte, onContent func(string)) (string, []tcAccum, error) {
	url := strings.TrimRight(req.BaseURL, "/") + "/chat/completions"
	hr, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("Authorization", "Bearer "+req.APIKey)
	hr.Header.Set("Accept", "text/event-stream")
	resp, err := httpClient.Do(hr)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4000))
		var pe provResp
		if json.Unmarshal(b, &pe) == nil && pe.Error != nil {
			return "", nil, fmt.Errorf("%s", pe.Error.Message)
		}
		return "", nil, fmt.Errorf("HTTP %d: %.300s", resp.StatusCode, b)
	}
	var content strings.Builder
	var tcs []tcAccum
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var ch streamChunk
		if json.Unmarshal([]byte(data), &ch) != nil {
			continue
		}
		if ch.Error != nil {
			return content.String(), tcs, fmt.Errorf("%s", ch.Error.Message)
		}
		if len(ch.Choices) == 0 {
			continue
		}
		d := ch.Choices[0].Delta
		if d.Content != "" {
			content.WriteString(d.Content)
			onContent(d.Content)
		}
		for _, t := range d.ToolCalls {
			for len(tcs) <= t.Index {
				tcs = append(tcs, tcAccum{})
			}
			if t.ID != "" {
				tcs[t.Index].ID = t.ID
			}
			if t.Function.Name != "" {
				tcs[t.Index].Name = t.Function.Name
			}
			tcs[t.Index].Args += t.Function.Arguments
		}
	}
	if err := sc.Err(); err != nil {
		return content.String(), tcs, err
	}
	return content.String(), tcs, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd, args = "open", []string{url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}
