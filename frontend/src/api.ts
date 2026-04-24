import type { ScanResult } from "./types";

export type MasterPreset = {
  module_type: string;
  cell_type: string;
};

export type MasterInfo = {
  path: string;
  filename: string;
  sheets: string[];
  total_rows: number;
  index_size: number;
  preset: MasterPreset;
  has_preset: boolean;
  recent_projects: string[];
};

export type BuildRequest = {
  serials: string[];
  module_type: string;
  cell_type: string;
  project_name: string;
  auto_number: boolean;
  save_preset: boolean;
};

export type BuildLine = {
  serial: string;
  found: boolean;
  warning?: string;
};

export type BuildResult = {
  path: string;
  matched: number;
  missing: number;
  lines: BuildLine[];
};

export type Settings = {
  vllm_base_url: string;
  vllm_model: string;
  use_vllm_fallback: boolean;
};

declare global {
  interface Window {
    go?: {
      main: {
        App: {
          ScanImage(filename: string, dataURL: string): Promise<ScanResult[]>;
          ExportExcel(rows: ScanResult[]): Promise<string>;
          PickAndLoadMaster(): Promise<MasterInfo | null>;
          LoadMaster(path: string): Promise<MasterInfo>;
          BuildReport(req: BuildRequest): Promise<BuildResult | null>;
          GetSettings(): Promise<Settings>;
          SaveSettings(s: Settings): Promise<void>;
        };
      };
    };
  }
}

function bridge() {
  if (!window.go?.main?.App) {
    throw new Error(
      "Wails 바인딩을 찾을 수 없습니다. 개발 중이면 `wails dev`로 실행하세요."
    );
  }
  return window.go.main.App;
}

function fileToDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}

export async function scanFile(file: File): Promise<ScanResult[]> {
  const dataURL = await fileToDataURL(file);
  return await bridge().ScanImage(file.name, dataURL);
}

export async function scanFiles(files: File[]): Promise<ScanResult[]> {
  const all: ScanResult[] = [];
  for (const f of files) {
    try {
      const rs = await scanFile(f);
      all.push(...rs);
    } catch (e) {
      all.push({
        filename: f.name,
        serial: "",
        suffix: "",
        source: "barcode",
        error: String(e),
      });
    }
  }
  return all;
}

export async function exportExcel(rows: ScanResult[]): Promise<string> {
  return await bridge().ExportExcel(rows);
}

export async function pickAndLoadMaster(): Promise<MasterInfo | null> {
  return await bridge().PickAndLoadMaster();
}

export async function buildReport(req: BuildRequest): Promise<BuildResult | null> {
  return await bridge().BuildReport(req);
}

export async function getSettings(): Promise<Settings> {
  return await bridge().GetSettings();
}

export async function saveSettings(s: Settings): Promise<void> {
  return await bridge().SaveSettings(s);
}
