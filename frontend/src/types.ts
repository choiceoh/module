export type ScanSource = "barcode" | "manual" | "vllm";

export type ScanResult = {
  filename: string;
  photo_no?: number;
  serial: string;
  suffix: string;
  source: ScanSource;
  pallet_sn?: string;
  notes?: string;
  error?: string;
};

export const SCAN_COLUMNS: { key: keyof ScanResult; label: string; width: number }[] = [
  { key: "filename", label: "파일", width: 180 },
  { key: "photo_no", label: "NO", width: 50 },
  { key: "pallet_sn", label: "팔레트", width: 140 },
  { key: "serial", label: "시리얼", width: 230 },
  { key: "suffix", label: "접미사", width: 50 },
  { key: "source", label: "출처", width: 60 },
  { key: "notes", label: "비고", width: 260 },
];

export const emptyRow = (): ScanResult => ({
  filename: "(수동 입력)",
  serial: "",
  suffix: "",
  source: "manual",
  notes: "",
});

export type Row = ScanResult & {
  _rowId: string;
};
