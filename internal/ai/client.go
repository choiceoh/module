package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	_ "image/jpeg"
	"io"
	"net/http"
	"time"

	"github.com/disintegration/imaging"

	"module-scanner/internal/schema"
)

type Client struct {
	BaseURL string
	Model   string
	HTTP    *http.Client
}

func New(baseURL, model string) *Client {
	return &Client{
		BaseURL: baseURL,
		Model:   model,
		HTTP:    &http.Client{Timeout: 120 * time.Second},
	}
}

// Ping hits /v1/models on the OpenAI-compatible server. Returns nil on
// success, an error describing the failure otherwise. Fast — 5s timeout
// is typical for callers.
func (c *Client) Ping(ctx context.Context) error {
	url := c.BaseURL + "/models"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

type chatMessage struct {
	Role    string    `json:"role"`
	Content []content `json:"content"`
}

type content struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    float64         `json:"temperature"`
	MaxTokens      int             `json:"max_tokens"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	GuidedJSON     json.RawMessage `json:"guided_json,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const systemPrompt = `너는 전자 부품 모듈의 사진이나 데이터시트 이미지에서 사양 정보를 정확히 추출하는 도우미다.
이미지에서 확인 가능한 값만 채우고, 알 수 없는 필드는 빈 문자열("")로 둔다. 추측하지 말 것.
단위는 이미지에 표기된 그대로 포함해라 (예: "5V", "3.3V/5V", "100mA", "-40~85°C", "25.4x17.8mm").
반드시 주어진 JSON 스키마에 맞는 JSON 하나만 출력한다.`

func (c *Client) ExtractModule(ctx context.Context, imageData []byte, mimeType string) (*schema.Module, error) {
	b64 := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

	req := chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: []content{{Type: "text", Text: systemPrompt}},
			},
			{
				Role: "user",
				Content: []content{
					{Type: "text", Text: "이 이미지의 모듈 정보를 스키마에 맞춰 추출해라."},
					{Type: "image_url", ImageURL: &imageURL{URL: dataURL}},
				},
			},
		},
		Temperature:    0.1,
		MaxTokens:      1024,
		ResponseFormat: &responseFormat{Type: "json_object"},
		GuidedJSON:     json.RawMessage(schema.ModuleJSONSchema),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("vllm request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("vllm %d: %s", resp.StatusCode, string(raw))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w (body=%s)", err, string(raw))
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("vllm error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	var mod schema.Module
	if err := json.Unmarshal([]byte(parsed.Choices[0].Message.Content), &mod); err != nil {
		return nil, fmt.Errorf("decode module json: %w (content=%s)", err, parsed.Choices[0].Message.Content)
	}
	return &mod, nil
}

type ModuleEntry struct {
	No     int    `json:"no"`
	Serial string `json:"serial"`
}

type SerialExtract struct {
	PalletSN string        `json:"pallet_sn"`
	Modules  []ModuleEntry `json:"modules"`
}

const serialExtractSchema = `{
  "type": "object",
  "properties": {
    "pallet_sn": {"type": "string"},
    "modules": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "no":     {"type": "integer"},
          "serial": {"type": "string"}
        },
        "required": ["no", "serial"],
        "additionalProperties": false
      }
    }
  },
  "required": ["pallet_sn", "modules"],
  "additionalProperties": false
}`

const serialExtractPrompt = `You are reading a solar module pallet packing list (e.g. Jinko, Trina).

The sheet is a grid with repeating "NO | SN" column pairs. Each small integer NO
(1, 2, 3, ...) sits next to exactly one module serial number. The SN is a long
alphanumeric string (15-24 chars) printed both as a barcode and as human-readable
text beneath it.

Task:
- "pallet_sn": the pallet/tray number (short, ~12 chars, usually once, labeled
  "Pallet Number" / "托盘号" / shown at a corner; it's NOT one of the grid modules).
- "modules": for every NO you can read in the grid, output {"no": <int>,
  "serial": <printed text>}. Preserve the NO order as shown on the sheet.

Rules:
- Transcribe ONLY the printed text; do NOT guess characters.
- If a row has a NO but the serial beneath is illegible, include it with serial="".
- If a NO is not printed or clearly absent, omit that entry entirely.
- Never emit duplicates, header labels, or product codes (e.g. "TSM-720NEG21C.20K").
- Never emit the pallet_sn as a module entry.
- Output JSON only, matching the given schema.`

// ExtractSerials splits the image into overlapping vertical slices and
// sends each to the VLM. Vision models typically downscale inputs to
// ~1024x1024, so a 3000px-tall packing list has tiny barcode text after
// resize. Tiling gives the model a larger effective resolution per call.
func (c *Client) ExtractSerials(ctx context.Context, imageData []byte, mimeType string) (*SerialExtract, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("decode image for tiling: %w", err)
	}
	bounds := img.Bounds()
	h := bounds.Dy()
	w := bounds.Dx()

	// Decide tile count based on height. >2000px → 3 tiles, >1200 → 2, else 1.
	tiles := 1
	switch {
	case h > 2000:
		tiles = 3
	case h > 1200:
		tiles = 2
	}

	var tilePNGs [][]byte
	if tiles == 1 {
		tilePNGs = [][]byte{imageData}
	} else {
		sliceH := h / tiles
		overlap := sliceH / 4
		for i := 0; i < tiles; i++ {
			y0 := i*sliceH - overlap
			y1 := (i+1)*sliceH + overlap
			if y0 < 0 {
				y0 = 0
			}
			if y1 > h {
				y1 = h
			}
			if i == tiles-1 {
				y1 = h
			}
			crop := imaging.Crop(img, image.Rect(0, y0, w, y1))
			var buf bytes.Buffer
			if err := png.Encode(&buf, crop); err != nil {
				continue
			}
			tilePNGs = append(tilePNGs, buf.Bytes())
		}
	}

	seen := map[string]struct{}{}
	merged := &SerialExtract{}
	for i, tile := range tilePNGs {
		mime := mimeType
		if tiles > 1 {
			mime = "image/png"
		}
		ext, err := c.extractSerialsOnce(ctx, tile, mime, i+1, tiles)
		if err != nil {
			if i == 0 && len(tilePNGs) == 1 {
				return nil, err
			}
			continue
		}
		if merged.PalletSN == "" && ext.PalletSN != "" {
			merged.PalletSN = ext.PalletSN
		}
		for _, m := range ext.Modules {
			s := m.Serial
			if s == "" {
				continue
			}
			if _, dup := seen[s]; dup {
				continue
			}
			seen[s] = struct{}{}
			merged.Modules = append(merged.Modules, m)
		}
	}

	if len(merged.Modules) == 0 && merged.PalletSN == "" {
		return nil, fmt.Errorf("no serials extracted from %d tile(s)", tiles)
	}
	return merged, nil
}

func (c *Client) extractSerialsOnce(ctx context.Context, imageData []byte, mimeType string, tileIdx, tileCount int) (*SerialExtract, error) {
	b64 := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

	userMsg := "Extract all module serials and the pallet number from this packing list image. A single sheet typically has 30-40 modules — do NOT stop early. Read EVERY row you can see."
	if tileCount > 1 {
		userMsg = fmt.Sprintf("This is tile %d of %d (vertical slice) from a packing list. Read every module serial in this slice. Do NOT stop early.", tileIdx, tileCount)
	}

	req := chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: []content{{Type: "text", Text: serialExtractPrompt}},
			},
			{
				Role: "user",
				Content: []content{
					{Type: "text", Text: userMsg},
					{Type: "image_url", ImageURL: &imageURL{URL: dataURL}},
				},
			},
		},
		Temperature:    0.0,
		MaxTokens:      8192,
		ResponseFormat: &responseFormat{Type: "json_object"},
		GuidedJSON:     json.RawMessage(serialExtractSchema),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("vllm request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("vllm %d: %s", resp.StatusCode, string(raw))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w (body=%s)", err, string(raw))
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("vllm error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	var ext SerialExtract
	if err := json.Unmarshal([]byte(parsed.Choices[0].Message.Content), &ext); err != nil {
		return nil, fmt.Errorf("decode serials json: %w (content=%s)", err, parsed.Choices[0].Message.Content)
	}
	return &ext, nil
}
