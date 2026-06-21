// sojeongcompose — 창작국악 AI 작곡 보조 (독립 실행파일, 에이전트 방식)
//
// index.html(슬림 UI)과 작곡 스킬 문서(skill/)를 바이너리에 embed 한다.
// localhost로 띄우고, /api/chat 에서 LLM 에이전트 루프를 돌린다:
// 모델에게 read_doc 도구를 주어, 질문에 맞는 참고 문서 1~3개만 스스로 읽고 답하게 한다
// (스킬이 의도한 "네이티브 lazy-loading"). 브라우저 직접 호출이 아니라 CORS 없음.
package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os/exec"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"
)

//go:embed index.html
var indexHTML []byte

//go:embed abcjs-basic-min.js
var abcjsJS []byte

//go:embed all:skill
var skillFS embed.FS

var httpClient = &http.Client{Timeout: 600 * time.Second}

const maxIters = 8

// 프런트엔드 → 백엔드 요청(원시 입력; 메시지/지식 구성은 백엔드가 함)
type uiReq struct {
	BaseURL  string `json:"baseUrl"`
	APIKey   string `json:"apiKey"`
	Model    string `json:"model"`
	Intent   string `json:"intent"` // 곡 특성·나타내려는 악상(작곡가 서술)
	Score    string `json:"score"`  // MusicXML 요약(악기·조표·박자·음 나열)
	Question string `json:"question"`
	Extra    string `json:"extra"`
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
	b.WriteString("=== 작곡 스킬 핵심 지침 (SKILL.md) ===\n")
	b.Write(skill)
	b.WriteString("\n\n=== 읽을 수 있는 문서 목록 (read_doc 의 path 로 사용) ===\n")
	b.WriteString(strings.Join(docList(), "\n"))
	return b.String()
}

func userPrompt(r uiReq) string {
	var p []string
	if r.Intent != "" {
		p = append(p, "곡 특성·나타내려는 악상(작곡가 서술): "+r.Intent)
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

// 공급자 응답(필요 부분만)
type provResp struct {
	Choices []struct {
		Message json.RawMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}
type asstMsg struct {
	Content   string `json:"content"`
	ToolCalls []struct {
		ID       string `json:"id"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	} `json:"tool_calls"`
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

	raw := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
	messages := []json.RawMessage{
		raw(map[string]any{"role": "system", "content": systemPrompt()}),
		raw(map[string]any{"role": "user", "content": userPrompt(req)}),
	}
	var readDocs []string

	for i := 0; i < maxIters; i++ {
		body, _ := json.Marshal(map[string]any{
			"model": req.Model, "temperature": 0.8,
			"messages": messages, "tools": tools, "tool_choice": "auto",
		})
		respBytes, status, err := callProvider(req, body)
		if err != nil {
			msg := err.Error()
			hint := ""
			if strings.Contains(msg, "deadline") || strings.Contains(strings.ToLower(msg), "timeout") {
				hint = "\n\n응답이 너무 느려 시간이 초과됐습니다. ① 잠시 후 다시 시도, ② 질문을 더 좁게, " +
					"③ Base URL 엔드포인트가 맞는지 확인하세요(예: https://api.z.ai/api/paas/v4). " +
					"공급자가 혼잡하거나 모델이 매우 길게 답하는 경우 발생할 수 있습니다."
			}
			writeJSON(w, map[string]any{"error": "공급자 호출 실패: " + msg + hint})
			return
		}
		var pr provResp
		if e := json.Unmarshal(respBytes, &pr); e != nil {
			writeJSON(w, map[string]any{"error": fmt.Sprintf("응답 파싱 실패(HTTP %d): %.300s", status, respBytes)})
			return
		}
		if pr.Error != nil {
			writeJSON(w, map[string]any{"error": pr.Error.Message})
			return
		}
		if len(pr.Choices) == 0 {
			writeJSON(w, map[string]any{"error": fmt.Sprintf("빈 응답(HTTP %d): %.300s", status, respBytes)})
			return
		}
		am := pr.Choices[0].Message
		messages = append(messages, am) // assistant 메시지(원형 그대로) 재투입
		var parsed asstMsg
		json.Unmarshal(am, &parsed)
		if len(parsed.ToolCalls) == 0 {
			writeJSON(w, map[string]any{"content": parsed.Content, "docs": readDocs})
			return
		}
		// 도구 호출 처리
		for _, tc := range parsed.ToolCalls {
			var args struct {
				Path string `json:"path"`
			}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			content := readDoc(args.Path)
			readDocs = append(readDocs, args.Path)
			messages = append(messages, raw(map[string]any{
				"role": "tool", "tool_call_id": tc.ID, "content": content,
			}))
		}
	}
	writeJSON(w, map[string]any{"error": "도구 호출이 너무 많아 중단(질문을 좁혀보세요)."})
}

func callProvider(req uiReq, body []byte) ([]byte, int, error) {
	url := strings.TrimRight(req.BaseURL, "/") + "/chat/completions"
	hr, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("Authorization", "Bearer "+req.APIKey)
	resp, err := httpClient.Do(hr)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return b, resp.StatusCode, err
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
