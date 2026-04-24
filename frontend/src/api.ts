import type { ExtractResult, Module } from "./types";

export async function extractModules(files: File[]): Promise<ExtractResult[]> {
  const fd = new FormData();
  for (const f of files) fd.append("images", f);

  const res = await fetch("/api/extract", { method: "POST", body: fd });
  if (!res.ok) {
    const txt = await res.text();
    throw new Error(`추출 실패 (${res.status}): ${txt}`);
  }
  const body = (await res.json()) as { results: ExtractResult[] };
  return body.results;
}

export async function exportExcel(modules: Module[]): Promise<Blob> {
  const res = await fetch("/api/export", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ modules }),
  });
  if (!res.ok) {
    const txt = await res.text();
    throw new Error(`엑셀 생성 실패 (${res.status}): ${txt}`);
  }
  return await res.blob();
}

export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
