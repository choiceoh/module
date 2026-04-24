package schema

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

var Columns = []struct {
	Key   string
	Label string
}{
	{"model_name", "모델명"},
	{"manufacturer", "제조사"},
	{"category", "카테고리"},
	{"voltage_rated", "정격전압"},
	{"current_rated", "정격전류"},
	{"interface", "인터페이스"},
	{"temp_range", "동작온도"},
	{"dimensions", "치수"},
	{"weight", "무게"},
	{"notes", "비고"},
}

const JSONSchema = `{
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
