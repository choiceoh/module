# Sidecar build instructions

The OCR sidecar is a PyInstaller-bundled Python program that exposes
`rapidocr_onnxruntime` (PaddleOCR ONNX) and PyMuPDF over a
stdin/stdout JSON protocol. The Go side spawns this exe once and
keeps it alive for the app's lifetime.

The sidecar runs **PP-OCRv5** detection + recognition with the
PP-OCRv4 angle-classifier. v5 ONNX models are not bundled with
`rapidocr_onnxruntime`, so they must be downloaded and packaged into
the exe via PyInstaller `--add-data`.

The compiled `sidecar.exe` is ~130MB and is **not** checked in. Build
it locally before running `wails build`.

## Prerequisites

- Python 3.12 (any 3.10+ works)
- Windows (the app is Windows-only)

## 1. Fetch ONNX models

Place these four files in `internal/ocr/sidecar-src/models/` (create
the directory). Filenames must match exactly — the sidecar resolves
them by name.

| Local filename | Source |
| --- | --- |
| `det.onnx` | PP-OCRv5 mobile detection (`PP-OCRv5_mobile_det_infer.onnx`) |
| `rec.onnx` | PP-OCRv5 mobile recognition (`PP-OCRv5_mobile_rec_infer.onnx`) |
| `cls.onnx` | PP-OCRv4 angle classifier (`ch_ppocr_mobile_v2.0_cls_infer.onnx` from `rapidocr_onnxruntime`'s default models is fine) |
| `keys.txt` | PP-OCRv5 recognition dictionary (`ppocrv5_dict.txt`) |

Recommended sources:
- HuggingFace: `RapidAI/RapidOCR` and `PaddlePaddle/PP-OCRv5_mobile_*`
- ModelScope: `RapidAI/RapidOCR`

If the recognition model and `keys.txt` come from different versions,
inference will produce garbage — they must be paired.

## 2. Build

```bash
pip install rapidocr-onnxruntime PyMuPDF pyinstaller
cd internal/ocr/sidecar-src
python -m PyInstaller --onefile --windowed --name rapidocr_sidecar \
    --collect-data rapidocr_onnxruntime \
    --collect-data onnxruntime \
    --collect-submodules fitz \
    --add-data "models;models" \
    rapidocr_main.py
cp dist/rapidocr_sidecar.exe ../assets/sidecar.exe
```

Notes:
- `--add-data "models;models"` (semicolon on Windows) bundles the four
  ONNX/dict files. At runtime PyInstaller extracts them under
  `sys._MEIPASS/models/`, which `rapidocr_main.py` resolves
  automatically.
- On macOS/Linux dev shells the separator is `:` instead of `;`. Use
  `--add-data "models:models"` there.

After this, `internal/ocr/assets/sidecar.exe` exists and `wails build`
(or `build.ps1`) will succeed.

## Why a sidecar instead of a Go port?

The pure-Go PaddleOCR port (PR #13, v0.6.0) topped out at ~30/33
recognition on tilted phone photos. The official Python pipeline uses
shapely/pyclipper for DB postprocessing (Vatti polygon offset) which
is non-trivial to reimplement in Go. Sidecar trades ~130MB of bundle
size for guaranteed parity with the upstream model.

## Why explicit v5 models instead of `rapidocr` 3.x?

`rapidocr` 3.x defaults to PP-OCRv5 but lazy-loads model weights from
ModelScope on first run, which breaks our offline single-exe
distribution. Staying on `rapidocr_onnxruntime` and pointing it at
v5 ONNX files via `det_model_path` / `rec_model_path` /
`rec_keys_path` keeps the bundle self-contained while still picking
up the v5 accuracy improvement (≈+13pp end-to-end on PaddleOCR's
benchmarks; notably better on Korean and tilted/handwritten text).

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
