// sojeongcompose — 창작국악 AI 작곡 보조 (독립 실행파일)
//
// index.html(작곡 지식 내장 UI)을 바이너리에 embed 해 localhost로 띄우고,
// AI 호출은 이 백엔드가 대신 한다(브라우저 직접 호출이 아니므로 CORS 문제 없음).
// 더블클릭하면 기본 브라우저로 열린다.
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

//go:embed index.html
var indexHTML []byte

// 프런트엔드가 보내는 요청: 공급자 정보 + 메시지(키는 localhost로만 전달)
type chatReq struct {
	BaseURL     string          `json:"baseUrl"`
	APIKey      string          `json:"apiKey"`
	Model       string          `json:"model"`
	Temperature float64         `json:"temperature"`
	Messages    json.RawMessage `json:"messages"`
}

var httpClient = &http.Client{Timeout: 180 * time.Second}

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

	// 빈 포트 자동 선택(127.0.0.1 만 바인딩 → 외부 노출 없음)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("포트 열기 실패: %v", err)
	}
	url := "http://" + ln.Addr().String()
	fmt.Println("sojeongcompose 실행 중 →", url)
	fmt.Println("(이 창을 닫으면 종료됩니다)")
	go func() {
		time.Sleep(300 * time.Millisecond)
		openBrowser(url)
	}()
	log.Fatal(http.Serve(ln, mux))
}

// OpenAI 호환 /chat/completions 로 그대로 전달하고 응답을 돌려준다.
func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req chatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.BaseURL == "" || req.Model == "" {
		http.Error(w, "baseUrl/model 필요", http.StatusBadRequest)
		return
	}
	if req.Temperature == 0 {
		req.Temperature = 0.8
	}
	fwd := map[string]any{
		"model":       req.Model,
		"temperature": req.Temperature,
		"messages":    req.Messages,
	}
	body, _ := json.Marshal(fwd)

	url := strings.TrimRight(req.BaseURL, "/") + "/chat/completions"
	preq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	preq.Header.Set("Content-Type", "application/json")
	if req.APIKey != "" {
		preq.Header.Set("Authorization", "Bearer "+req.APIKey)
	}
	resp, err := httpClient.Do(preq)
	if err != nil {
		http.Error(w, "공급자 호출 실패: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
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
