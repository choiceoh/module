# Sidecar build instructions

The OCR sidecar is a PyInstaller-bundled Python program that exposes
`rapidocr_onnxruntime` (PaddleOCR ONNX) and PyMuPDF over a
stdin/stdout JSON protocol. The Go side spawns this exe once and
keeps it alive for the app's lifetime.

The compiled `sidecar.exe` is ~108MB and is **not** checked in. Build
it locally before running `wails build`.

## Prerequisites

- Python 3.12 (any 3.10+ works)
- Windows (the app is Windows-only)

## Build

```bash
pip install rapidocr-onnxruntime PyMuPDF pyinstaller
cd internal/ocr/sidecar-src
python -m PyInstaller --onefile --windowed --name rapidocr_sidecar \
    --collect-data rapidocr_onnxruntime \
    --collect-data onnxruntime \
    --collect-submodules fitz \
    rapidocr_main.py
cp dist/rapidocr_sidecar.exe ../assets/sidecar.exe
```

After this, `internal/ocr/assets/sidecar.exe` exists and `wails build`
(or `build.ps1`) will succeed.

## Why a sidecar instead of a Go port?

The pure-Go PaddleOCR port (PR #13, v0.6.0) topped out at ~30/33
recognition on tilted phone photos. The official Python pipeline uses
shapely/pyclipper for DB postprocessing (Vatti polygon offset) which
is non-trivial to reimplement in Go. Sidecar trades ~80MB of bundle
size for guaranteed parity with the upstream model — 33/33 and 36/36
on the test packing lists.

## Protocol

Daemon mode (current): one path per stdin line, one JSON response per
stdout line. Model loaded once on startup, reused for every call.

```
> /path/to/image.jpg
< {"raw":[{"text":"E2FXJ...","score":0.99,"x0":12,"y0":34,...}]}
> /path/to/multi.pdf
< {"raw":[ ...all pages, Y-offset per page... ]}
```

The first stdout line is `{"ready": true}` — the Go side waits for
this before sending requests.
