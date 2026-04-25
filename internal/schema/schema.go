package schema

// Box is the bounding box of an OCR detection in image-pixel
// coordinates (top-left origin). Set only for OCR-sourced rows so the
// frontend can show a zoomed preview around the serial.
type Box struct {
	X0 int `json:"x0"`
	Y0 int `json:"y0"`
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
}

type ScanResult struct {
	Filename string  `json:"filename"`
	PhotoNo  int     `json:"photo_no,omitempty"`
	Serial   string  `json:"serial"`
	Suffix   string  `json:"suffix"`
	Source   string  `json:"source"`
	Score    float32 `json:"score,omitempty"`
	Box      *Box    `json:"box,omitempty"`
	PalletSN string  `json:"pallet_sn,omitempty"`
	Notes    string  `json:"notes,omitempty"`
	Error    string  `json:"error,omitempty"`
}

type Column struct {
	Key   string
	Label string
	Width float64
}

var Columns = []Column{
	{"filename", "파일", 24},
	{"photo_no", "NO", 6},
	{"pallet_sn", "팔레트", 16},
	{"serial", "시리얼", 30},
	{"suffix", "접미사", 10},
	{"source", "출처", 10},
	{"notes", "비고", 24},
}

type Module struct {
	ModelName    string `json:"model_name"`
	Manufacturer string `json:"manufacturer"`
	Category     string `json:"category"`
	VoltageRated string `json:"voltage_rated"`
	CurrentRated string `json:"current_rated"`
	Interface    string `json:"interface"`
	TempRange    string `json:"temp_range"`
	Dimensions   string `json:"dimensions"`
	Weight       string `json:"weight"`
	Notes        string `json:"notes"`
}

const ModuleJSONSchema = `{
  "type": "object",
  "properties": {
    "model_name":    {"type": "string"},
    "manufacturer":  {"type": "string"},
    "category":      {"type": "string"},
    "voltage_rated": {"type": "string"},
    "current_rated": {"type": "string"},
    "interface":     {"type": "string"},
    "temp_range":    {"type": "string"},
    "dimensions":    {"type": "string"},
    "weight":        {"type": "string"},
    "notes":         {"type": "string"}
  },
  "required": ["model_name","manufacturer","category","voltage_rated","current_rated","interface","temp_range","dimensions","weight","notes"],
  "additionalProperties": false
}`
