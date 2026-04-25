import { useEffect, useState } from "react";
import { Button, Modal, Space, Tag, Typography } from "antd";
import { LeftOutlined, RightOutlined } from "@ant-design/icons";
import type { Row } from "./types";

const { Text } = Typography;

// Image extensions we can show in the preview. PDFs are excluded —
// the OCR sidecar stitches PDF pages and the bounding boxes use
// a synthetic Y-offset, so they don't map to a single source image.
const PREVIEWABLE_EXTS = new Set([
  "jpg",
  "jpeg",
  "png",
  "webp",
  "bmp",
  "gif",
]);

export function canPreview(row: Row): boolean {
  if (!row.box) return false;
  const dot = row.filename.lastIndexOf(".");
  if (dot < 0) return false;
  return PREVIEWABLE_EXTS.has(row.filename.slice(dot + 1).toLowerCase());
}

type Props = {
  row: Row;
  imageUrl: string | undefined;
  hasPrev: boolean;
  hasNext: boolean;
  onPrev: () => void;
  onNext: () => void;
  onClose: () => void;
};

export function ImagePreviewModal({
  row,
  imageUrl,
  hasPrev,
  hasNext,
  onPrev,
  onNext,
  onClose,
}: Props) {
  const [zoomed, setZoomed] = useState(true);
  const [imgDim, setImgDim] = useState<{ w: number; h: number } | null>(null);

  // Reset zoom + image dimensions whenever the row (and therefore the
  // source image) changes. Without this, navigating between rows from
  // different files would keep stale dimensions.
  useEffect(() => {
    setZoomed(true);
    setImgDim(null);
  }, [row.filename, imageUrl]);

  // Arrow keys navigate between rows; Esc closes (Modal default).
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "ArrowLeft" && hasPrev) {
        e.preventDefault();
        onPrev();
      } else if (e.key === "ArrowRight" && hasNext) {
        e.preventDefault();
        onNext();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [hasPrev, hasNext, onPrev, onNext]);

  const box = row.box;
  let viewBox = "0 0 100 100";
  if (imgDim) {
    if (zoomed && box) {
      // Pad the bounding box by 30% of its larger dimension so the
      // serial doesn't sit flush against the viewport edge.
      const bw = box.x1 - box.x0;
      const bh = box.y1 - box.y0;
      const pad = Math.max(bw, bh) * 0.3;
      const x = Math.max(0, box.x0 - pad);
      const y = Math.max(0, box.y0 - pad);
      const w = Math.min(imgDim.w - x, bw + pad * 2);
      const h = Math.min(imgDim.h - y, bh + pad * 2);
      viewBox = `${x} ${y} ${w} ${h}`;
    } else {
      viewBox = `0 0 ${imgDim.w} ${imgDim.h}`;
    }
  }

  return (
    <Modal
      open
      onCancel={onClose}
      footer={null}
      width={900}
      destroyOnClose
      title={
        <Space size="middle" wrap>
          <Text strong style={{ fontFamily: "ui-monospace, Consolas, monospace" }}>
            {row.serial || "(빈 시리얼)"}
          </Text>
          {typeof row.score === "number" && row.score > 0 && (
            <Tag color="gold">신뢰도 {(row.score * 100).toFixed(0)}%</Tag>
          )}
          <Text type="secondary" style={{ fontSize: 12 }}>
            {row.filename}
          </Text>
        </Space>
      }
    >
      {!imageUrl ? (
        <Text type="secondary">
          이 행의 원본 이미지를 찾지 못했습니다. 새로 스캔한 행만 미리보기 가능합니다.
        </Text>
      ) : !box ? (
        <Text type="secondary">바운딩 박스 정보가 없는 행입니다 (수동 입력 등).</Text>
      ) : (
        <>
          {/* Hidden loader to capture natural dimensions. */}
          <img
            src={imageUrl}
            alt=""
            style={{ display: "none" }}
            onLoad={(e) => {
              const img = e.currentTarget;
              setImgDim({ w: img.naturalWidth, h: img.naturalHeight });
            }}
          />
          {imgDim && (
            <div
              style={{
                width: "100%",
                maxHeight: "70vh",
                display: "flex",
                justifyContent: "center",
                background: "#fafafa",
              }}
            >
              <svg
                viewBox={viewBox}
                style={{ width: "100%", height: "auto", maxHeight: "70vh" }}
                preserveAspectRatio="xMidYMid meet"
              >
                <image
                  href={imageUrl}
                  width={imgDim.w}
                  height={imgDim.h}
                />
                <rect
                  x={box.x0}
                  y={box.y0}
                  width={box.x1 - box.x0}
                  height={box.y1 - box.y0}
                  fill="none"
                  stroke="#ff4d4f"
                  strokeWidth={3}
                  vectorEffect="non-scaling-stroke"
                />
              </svg>
            </div>
          )}
          <Space style={{ marginTop: 12, width: "100%", justifyContent: "space-between" }}>
            <Button onClick={() => setZoomed((z) => !z)}>
              {zoomed ? "전체 보기" : "확대 보기"}
            </Button>
            <Space>
              <Button
                icon={<LeftOutlined />}
                disabled={!hasPrev}
                onClick={onPrev}
              >
                이전
              </Button>
              <Button
                icon={<RightOutlined />}
                disabled={!hasNext}
                onClick={onNext}
              >
                다음
              </Button>
            </Space>
          </Space>
          <Text type="secondary" style={{ fontSize: 11, display: "block", marginTop: 8 }}>
            ←/→ 키로 이전/다음 행 이동, Esc 닫기
          </Text>
        </>
      )}
    </Modal>
  );
}
