package schema

type ScanResult struct {
	Filename  string  `json:"filename"`
	PhotoNo   int     `json:"photo_no,omitempty"`
	Serial    string  `json:"serial"`
	Suffix    string  `json:"suffix"`
	Source    string  `json:"source"`
	Score     float32 `json:"score,omitempty"`
	PalletSN  string  `json:"pallet_sn,omitempty"`
	Corrected bool    `json:"corrected,omitempty"`
	RawText   string  `json:"raw_text,omitempty"`
	Notes     string  `json:"notes,omitempty"`
	Error     string  `json:"error,omitempty"`
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
