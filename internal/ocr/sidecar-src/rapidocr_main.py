"""Sidecar OCR daemon. Reads image/PDF paths from stdin (one per line),
writes JSON responses (one per line) to stdout. Model loaded once.

Protocol:
  stdin:  <path>\n
  stdout: {"raw":[...]}\n  or  {"error":"..."}\n

Exits when stdin closes.
"""
import sys
import os
import json
import tempfile
from rapidocr_onnxruntime import RapidOCR


def _models_dir() -> str:
    # PyInstaller --onefile extracts bundled data under sys._MEIPASS.
    base = getattr(sys, "_MEIPASS", os.path.dirname(os.path.abspath(__file__)))
    return os.path.join(base, "models")


def _build_reader() -> RapidOCR:
    d = _models_dir()
    det = os.path.join(d, "det.onnx")
    rec = os.path.join(d, "rec.onnx")
    cls = os.path.join(d, "cls.onnx")
    keys = os.path.join(d, "keys.txt")
    missing = [p for p in (det, rec, cls, keys) if not os.path.isfile(p)]
    if missing:
        raise FileNotFoundError(
            "missing OCR model files: " + ", ".join(missing)
            + " (see internal/ocr/sidecar-src/BUILD.md)"
        )
    return RapidOCR(
        det_model_path=det,
        rec_model_path=rec,
        cls_model_path=cls,
        rec_keys_path=keys,
    )


def ocr_image(reader, path, y_offset=0):
    result, _ = reader(path)
    raws = []
    if result:
        for box, text, score in result:
            xs = [p[0] for p in box]
            ys = [p[1] for p in box]
            raws.append({
                "text": text,
                "score": float(score),
                "x0": int(min(xs)),
                "y0": int(min(ys)) + y_offset,
                "x1": int(max(xs)),
                "y1": int(max(ys)) + y_offset,
            })
    return raws


def ocr_pdf(reader, path):
    import fitz
    doc = fitz.open(path)
    all_raws = []
    y_offset = 0
    try:
        with tempfile.TemporaryDirectory() as td:
            for page_num in range(len(doc)):
                page = doc.load_page(page_num)
                pix = page.get_pixmap(dpi=200)
                img_path = os.path.join(td, f"page_{page_num}.png")
                pix.save(img_path)
                all_raws.extend(ocr_image(reader, img_path, y_offset))
                y_offset += pix.height + 50
    finally:
        doc.close()
    return all_raws


def write_response(payload: dict) -> None:
    sys.stdout.buffer.write((json.dumps(payload, ensure_ascii=True) + "\n").encode())
    sys.stdout.flush()


def main():
    reader = _build_reader()
    # Signal ready so the parent can know init is done.
    write_response({"ready": True})

    # Read stdin as raw bytes (UTF-8) — Windows default cp949 mangles
    # non-ASCII paths.
    for raw_line in sys.stdin.buffer:
        path = raw_line.decode("utf-8", errors="replace").strip()
        if not path:
            continue
        try:
            ext = os.path.splitext(path)[1].lower()
            if ext == ".pdf":
                raws = ocr_pdf(reader, path)
            else:
                raws = ocr_image(reader, path)
            write_response({"raw": raws})
        except Exception as e:
            write_response({"error": str(e)})


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        pass
