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

//go:embed all:skill
var skillFS embed.FS

var httpClient = &http.Client{Timeout: 180 * time.Second}

const maxIters = 8

// 프런트엔드 → 백엔드 요청(원시 입력; 메시지/지식 구성은 백엔드가 함)
type uiReq struct {
	BaseURL    string `json:"baseUrl"`
	APIKey     string `json:"apiKey"`
	Model      string `json:"model"`
	Instrument string `json:"instrument"`
	Mode       string `json:"mode"`
	Jangdan    string `json:"jangdan"`
	Score      string `json:"score"`
	Question   string `json:"question"`
	Extra      string `json:"extra"`
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
	mux.HandleFunc("/api/chat", handleChat)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("포트 열기 실패: %v", err)
	}
	url := "http://" + ln.Addr().String()
	fmt.Println("sojeongcompose 실행 중 →", url)
	fmt.Println("(이 창을 닫으면 종료됩니다)")
	go func() { time.Sleep(300 * time.Millisecond); openBrowser(url) }()
	log.Fatal(http.Serve(ln, mux))
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
	b.WriteString("시김새 적용 지점을 구체적으로. 모르면 솔직히. 한국어로 답한다.\n\n")
	b.WriteString("당신에게는 작곡 스킬 문서를 읽는 read_doc(path) 도구가 있습니다.\n")
	b.WriteString("먼저 references/00-navigation.md 를 읽어 어떤 문서가 필요한지 판단하고,\n")
	b.WriteString("질문에 직접 관련된 문서 1~3개만 읽은 뒤, 그 가이드에 근거해 답하세요\n")
	b.WriteString("(전부 읽지 말 것 — 5개 이상이면 질문이 너무 넓은 것).\n\n")
	b.WriteString("=== 작곡 스킬 핵심 지침 (SKILL.md) ===\n")
	b.Write(skill)
	b.WriteString("\n\n=== 읽을 수 있는 문서 목록 (read_doc 의 path 로 사용) ===\n")
	b.WriteString(strings.Join(docList(), "\n"))
	return b.String()
}

func userPrompt(r uiReq) string {
	var p []string
	if r.Instrument != "" {
		p = append(p, "악기: "+r.Instrument)
	}
	if r.Mode != "" {
		p = append(p, "조: "+r.Mode)
	}
	if r.Jangdan != "" {
		p = append(p, "장단: "+r.Jangdan)
	}
	if r.Score != "" {
		p = append(p, "악보(MusicXML 요약):\n"+r.Score)
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
			writeJSON(w, map[string]any{"error": "공급자 호출 실패: " + err.Error()})
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
