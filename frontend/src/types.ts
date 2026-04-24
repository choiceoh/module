export type Module = {
  model_name: string;
  manufacturer: string;
  category: string;
  voltage_rated: string;
  current_rated: string;
  interface: string;
  temp_range: string;
  dimensions: string;
  weight: string;
  notes: string;
};

export const MODULE_COLUMNS: { key: keyof Module; label: string }[] = [
  { key: "model_name", label: "모델명" },
  { key: "manufacturer", label: "제조사" },
  { key: "category", label: "카테고리" },
  { key: "voltage_rated", label: "정격전압" },
  { key: "current_rated", label: "정격전류" },
  { key: "interface", label: "인터페이스" },
  { key: "temp_range", label: "동작온도" },
  { key: "dimensions", label: "치수" },
  { key: "weight", label: "무게" },
  { key: "notes", label: "비고" },
];

export const emptyModule = (): Module => ({
  model_name: "",
  manufacturer: "",
  category: "",
  voltage_rated: "",
  current_rated: "",
  interface: "",
  temp_range: "",
  dimensions: "",
  weight: "",
  notes: "",
});

export type ExtractResult = {
  filename: string;
  module?: Module;
  error?: string;
};

export type Row = Module & {
  _rowId: string;
  _filename: string;
  _error?: string;
};
