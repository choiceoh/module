import { useEffect, useMemo, useState, type ReactNode } from "react";
import {
  Button,
  Card,
  Layout,
  Space,
  Table,
  Typography,
  Upload,
  message,
  Popconfirm,
  Input,
  Alert,
  Tag,
  Descriptions,
  Form,
  Checkbox,
  AutoComplete,
  Divider,
  Statistic,
  Row as AntRow,
  Col,
  Modal,
  Select,
} from "antd";
import type { UploadFile } from "antd/es/upload/interface";
import {
  InboxOutlined,
  FileExcelOutlined,
  DeleteOutlined,
  PlusOutlined,
  FileSearchOutlined,
  ExportOutlined,
  EyeOutlined,
} from "@ant-design/icons";
import {
  emptyRow,
  type ScanResult,
  type ScanSource,
  type Row,
} from "./types";
import {
  buildReport,
  exportExcel,
  getVersion,
  pickAndLoadMaster,
  scanFiles,
  type BuildResult,
  type MasterInfo,
} from "./api";
import { ImagePreviewModal, canPreview } from "./ImagePreviewModal";

const { Header, Content } = Layout;
const { Title, Text } = Typography;

let rowSeq = 0;
const newRowId = () => `row-${Date.now()}-${++rowSeq}`;

const sourceColor: Record<ScanSource, string> = {
  barcode: "green",
  manual: "blue",
  ocr: "orange",
};

const sourceLabel: Record<ScanSource, string> = {
  barcode: "바코드",
  manual: "수동",
  ocr: "OCR",
};

// OCR rows below this confidence are flagged for review.
const OCR_LOW_CONFIDENCE = 0.85;

export default function App() {
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [rows, setRows] = useState<Row[]>([]);
  const [scanning, setScanning] = useState(false);
  const [exporting, setExporting] = useState(false);

  const [master, setMaster] = useState<MasterInfo | null>(null);
  const [loadingMaster, setLoadingMaster] = useState(false);

  const [moduleType, setModuleType] = useState("");
  const [cellType, setCellType] = useState("");
  const [projectName, setProjectName] = useState("");
  const [autoNumber, setAutoNumber] = useState(true);
  const [savePreset, setSavePreset] = useState(true);

  const [building, setBuilding] = useState(false);
  const [lastBuild, setLastBuild] = useState<BuildResult | null>(null);

  const [diffOpen, setDiffOpen] = useState(false);
  const [expectedText, setExpectedText] = useState("");

  // Object URLs for the source images, keyed by filename. Held only
  // for the current session — losing them on refresh is acceptable.
  const [imageUrls, setImageUrls] = useState<Map<string, string>>(new Map());
  const [previewIdx, setPreviewIdx] = useState<number | null>(null);

  const [version, setVersion] = useState<string>("");

  useEffect(() => {
    getVersion()
      .then(setVersion)
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (master?.has_preset) {
      setModuleType(master.preset.module_type);
      setCellType(master.preset.cell_type);
    }
  }, [master]);

  const duplicateSerials = useMemo(() => {
    const counts = new Map<string, number>();
    for (const r of rows) {
      const s = r.serial?.trim();
      if (!s) continue;
      counts.set(s, (counts.get(s) ?? 0) + 1);
    }
    const dups = new Set<string>();
    counts.forEach((c, s) => {
      if (c > 1) dups.add(s);
    });
    return dups;
  }, [rows]);

  const lowConfidenceCount = useMemo(
    () =>
      rows.filter(
        (r) =>
          r.source === "ocr" &&
          typeof r.score === "number" &&
          r.score > 0 &&
          r.score < OCR_LOW_CONFIDENCE
      ).length,
    [rows]
  );

  const palletStats = useMemo(() => {
    const m = new Map<string, number>();
    for (const r of rows) {
      if (!r.serial) continue;
      const key = r.pallet_sn || "(미지정)";
      m.set(key, (m.get(key) ?? 0) + 1);
    }
    return Array.from(m.entries()).map(([pallet, count]) => ({ pallet, count }));
  }, [rows]);

  const palletColors = useMemo(() => {
    const palette = [
      "magenta",
      "volcano",
      "gold",
      "lime",
      "green",
      "cyan",
      "blue",
      "geekblue",
      "purple",
    ];
    const m = new Map<string, string>();
    let i = 0;
    for (const { pallet } of palletStats) {
      if (pallet === "(미지정)") {
        m.set(pallet, "default");
        continue;
      }
      m.set(pallet, palette[i % palette.length]);
      i++;
    }
    return m;
  }, [palletStats]);

  const validSerials = useMemo(
    () => rows.map((r) => r.serial).filter((s) => !!s),
    [rows]
  );

  const expectedSerials = useMemo(() => {
    // Accept newlines, commas, tabs, or whitespace as separators.
    const tokens = expectedText
      .split(/[\s,;]+/)
      .map((t) => t.trim())
      .filter(Boolean);
    return Array.from(new Set(tokens));
  }, [expectedText]);

  const diffStats = useMemo(() => {
    const scanned = new Set(
      rows.map((r) => r.serial?.trim()).filter((s): s is string => !!s)
    );
    const expected = new Set(expectedSerials);
    const matched: string[] = [];
    const missing: string[] = [];
    const extra: string[] = [];
    expected.forEach((s) => {
      if (scanned.has(s)) matched.push(s);
      else missing.push(s);
    });
    scanned.forEach((s) => {
      if (!expected.has(s)) extra.push(s);
    });
    return { matched, missing, extra };
  }, [rows, expectedSerials]);

  const missingSet = useMemo(() => new Set(diffStats.missing), [diffStats]);
  const extraSet = useMemo(() => new Set(diffStats.extra), [diffStats]);
  const diffActive = expectedSerials.length > 0;

  const handleScan = async () => {
    const files = fileList
      .map((f) => f.originFileObj as File | undefined)
      .filter((f): f is File => !!f);
    if (files.length === 0) {
      message.warning("먼저 이미지를 업로드해주세요.");
      return;
    }
    // Capture object URLs before scan so the preview modal can show
    // the source image even after fileList is cleared. Same-filename
    // re-uploads override (last write wins; rare in practice).
    setImageUrls((prev) => {
      const next = new Map(prev);
      for (const f of files) {
        const old = next.get(f.name);
        if (old) URL.revokeObjectURL(old);
        next.set(f.name, URL.createObjectURL(f));
      }
      return next;
    });
    setScanning(true);
    try {
      const results = await scanFiles(files);
      const newRows: Row[] = results.map((r) => ({
        ...r,
        _rowId: newRowId(),
      }));
      setRows((prev) => [...prev, ...newRows]);
      const ok = results.filter((r) => !r.error && r.serial).length;
      const err = results.filter((r) => r.error).length;
      message.success(`${ok}개 시리얼 추출 (실패 ${err})`);
      setFileList([]);
    } catch (e) {
      message.error(String(e));
    } finally {
      setScanning(false);
    }
  };

  const handleExport = async () => {
    if (rows.length === 0) {
      message.warning("내보낼 데이터가 없습니다.");
      return;
    }
    setExporting(true);
    try {
      const clean: ScanResult[] = rows.map(({ _rowId, ...r }) => {
        void _rowId;
        return r;
      });
      const path = await exportExcel(clean);
      if (path) message.success(`저장됨: ${path}`);
    } catch (e) {
      message.error(String(e));
    } finally {
      setExporting(false);
    }
  };

  const handleLoadMaster = async () => {
    setLoadingMaster(true);
    try {
      const info = await pickAndLoadMaster();
      if (info) {
        setMaster(info);
        message.success(
          `마스터 로드: ${info.filename} (${info.index_size.toLocaleString()}개 시리얼)`
        );
      }
    } catch (e) {
      message.error(String(e));
    } finally {
      setLoadingMaster(false);
    }
  };

  const handleBuild = async () => {
    if (!master) {
      message.warning("먼저 마스터 엑셀을 불러오세요.");
      return;
    }
    if (validSerials.length === 0) {
      message.warning("시리얼이 없습니다.");
      return;
    }
    if (!moduleType || !cellType || !projectName) {
      message.warning("Module type, Cell Type, 프로젝트명을 모두 입력하세요.");
      return;
    }
    setBuilding(true);
    try {
      const res = await buildReport({
        serials: validSerials,
        module_type: moduleType,
        cell_type: cellType,
        project_name: projectName,
        auto_number: autoNumber,
        save_preset: savePreset,
      });
      if (res) {
        setLastBuild(res);
        message.success(
          `생성 완료: ${res.matched}건 매칭, ${res.missing}건 누락 → ${res.path}`
        );
      }
    } catch (e) {
      message.error(String(e));
    } finally {
      setBuilding(false);
    }
  };

  const updateCell = (rowId: string, key: keyof ScanResult, value: string) => {
    setRows((prev) =>
      prev.map((r) => (r._rowId === rowId ? { ...r, [key]: value } : r))
    );
  };

  const addEmptyRow = () => {
    setRows((prev) => [...prev, { ...emptyRow(), _rowId: newRowId() }]);
  };

  const deleteRow = (rowId: string) => {
    setRows((prev) => prev.filter((r) => r._rowId !== rowId));
  };

  const clearAll = () => {
    setRows([]);
    setImageUrls((prev) => {
      prev.forEach((url) => URL.revokeObjectURL(url));
      return new Map();
    });
    setPreviewIdx(null);
  };

  const cellStyle = { fontSize: 12 };
  const columns = [
    {
      title: "",
      key: "_preview",
      width: 36,
      fixed: "left" as const,
      render: (_: unknown, r: Row, idx: number) => {
        const previewable = canPreview(r) && imageUrls.has(r.filename);
        return (
          <Button
            type="text"
            size="small"
            icon={<EyeOutlined />}
            disabled={!previewable}
            title={previewable ? "원본 이미지 미리보기" : "미리보기 불가"}
            onClick={() => setPreviewIdx(idx)}
          />
        );
      },
    },
    {
      title: "NO",
      dataIndex: "photo_no",
      key: "photo_no",
      width: 50,
      fixed: "left" as const,
      render: (v: number | undefined, r: Row) => (
        <Text type={r.error ? "danger" : undefined} title={r.error} style={cellStyle}>
          {v && v > 0 ? v : r.error ? "⚠" : ""}
        </Text>
      ),
    },
    {
      title: "팔레트",
      dataIndex: "pallet_sn",
      key: "pallet_sn",
      width: 130,
      render: (v: string | undefined, r: Row) => {
        const color = palletColors.get(v || "(미지정)") ?? "default";
        return (
          <Input
            value={v ?? ""}
            size="small"
            variant="borderless"
            style={cellStyle}
            onChange={(e) => updateCell(r._rowId, "pallet_sn", e.target.value)}
            placeholder="—"
            addonBefore={
              v ? (
                <Tag color={color} style={{ margin: 0 }}>
                  •
                </Tag>
              ) : null
            }
          />
        );
      },
    },
    {
      title: "시리얼",
      dataIndex: "serial",
      key: "serial",
      width: 320,
      render: (v: string, r: Row) => {
        const trimmed = v?.trim() ?? "";
        const isDup = !!trimmed && duplicateSerials.has(trimmed);
        const isLowConf =
          r.source === "ocr" &&
          typeof r.score === "number" &&
          r.score > 0 &&
          r.score < OCR_LOW_CONFIDENCE;
        const isExtra = diffActive && !!trimmed && extraSet.has(trimmed);
        const isMatched = diffActive && !!trimmed && !isExtra;
        const tags: ReactNode[] = [];
        if (isDup) {
          tags.push(
            <Tag key="dup" color="orange" style={{ margin: 0, fontSize: 10 }}>
              중복
            </Tag>
          );
        }
        if (isLowConf) {
          tags.push(
            <Tag
              key="lowconf"
              color="gold"
              style={{ margin: 0, fontSize: 10 }}
              title={`OCR 신뢰도 ${(r.score! * 100).toFixed(0)}% — 검수 권장`}
            >
              {`${(r.score! * 100).toFixed(0)}%`}
            </Tag>
          );
        }
        if (isExtra) {
          tags.push(
            <Tag key="extra" color="red" style={{ margin: 0, fontSize: 10 }}>
              잉여
            </Tag>
          );
        } else if (isMatched) {
          tags.push(
            <Tag key="ok" color="green" style={{ margin: 0, fontSize: 10 }}>
              ✓
            </Tag>
          );
        }
        const status = isDup || isExtra ? "warning" : undefined;
        return (
          <Input
            value={v}
            size="small"
            variant="borderless"
            style={{
              fontSize: 15,
              fontFamily:
                "ui-monospace, 'Cascadia Mono', Consolas, monospace",
              background: isLowConf && !isDup && !isExtra ? "#fffbe6" : undefined,
            }}
            onChange={(e) => updateCell(r._rowId, "serial", e.target.value)}
            status={status}
            suffix={tags.length > 0 ? <Space size={2}>{tags}</Space> : null}
          />
        );
      },
    },
    {
      title: "접미사",
      dataIndex: "suffix",
      key: "suffix",
      width: 50,
      render: (v: string, r: Row) => (
        <Input
          value={v}
          size="small"
          variant="borderless"
          style={cellStyle}
          onChange={(e) => updateCell(r._rowId, "suffix", e.target.value)}
        />
      ),
    },
    {
      title: "출처",
      dataIndex: "source",
      key: "source",
      width: 60,
      render: (v: ScanSource) => (
        <Tag color={sourceColor[v]} style={{ fontSize: 10 }}>
          {sourceLabel[v]}
        </Tag>
      ),
    },
    {
      title: "비고",
      dataIndex: "notes",
      key: "notes",
      width: 320,
      render: (v: string | undefined, r: Row) => (
        <Input.TextArea
          value={v ?? ""}
          autoSize={{ minRows: 1, maxRows: 3 }}
          variant="borderless"
          style={{ ...cellStyle, padding: "2px 4px" }}
          onChange={(e) => updateCell(r._rowId, "notes", e.target.value)}
        />
      ),
    },
    {
      title: "",
      key: "_actions",
      width: 40,
      fixed: "right" as const,
      render: (_: unknown, r: Row) => (
        <Button
          type="text"
          size="small"
          danger
          icon={<DeleteOutlined />}
          onClick={() => deleteRow(r._rowId)}
        />
      ),
    },
  ];

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Header
        style={{
          background: "#fff",
          borderBottom: "1px solid #f0f0f0",
          display: "flex",
          alignItems: "center",
        }}
      >
        <Space align="baseline">
          <Title level={3} style={{ margin: 0, lineHeight: "64px" }}>
            모듈 시리얼 스캐너
          </Title>
          {version && (
            <Text type="secondary" style={{ fontSize: 13 }}>
              v{version}
            </Text>
          )}
        </Space>
      </Header>
      <Content style={{ padding: 24, maxWidth: 1400, margin: "0 auto", width: "100%" }}>
        <Space direction="vertical" size="large" style={{ width: "100%" }}>
          <Card size="small" title="1. 사진에서 시리얼 추출">
            <Space direction="vertical" size="middle" style={{ width: "100%" }}>
              <Upload.Dragger
                multiple
                accept="image/*,application/pdf,.pdf"
                fileList={fileList}
                beforeUpload={() => false}
                onChange={(info) => setFileList(info.fileList)}
                style={{ padding: 12 }}
              >
                <p className="ant-upload-drag-icon">
                  <InboxOutlined />
                </p>
                <p className="ant-upload-text">
                  모듈 측면 라벨 사진을 이 영역에 끌어놓거나 클릭해서 업로드
                </p>
                <p className="ant-upload-hint">
                  한 장에 여러 바코드가 있어도 모두 추출합니다.
                </p>
              </Upload.Dragger>

              <Space wrap>
                <Button
                  type="primary"
                  onClick={handleScan}
                  loading={scanning}
                  disabled={fileList.length === 0}
                >
                  스캔 실행 ({fileList.length})
                </Button>
                <Button icon={<PlusOutlined />} onClick={addEmptyRow}>
                  빈 행 추가
                </Button>
                <Popconfirm
                  title="모든 결과를 지울까요?"
                  onConfirm={clearAll}
                  okText="지우기"
                  cancelText="취소"
                >
                  <Button danger disabled={rows.length === 0}>
                    전체 초기화
                  </Button>
                </Popconfirm>
                <Button
                  icon={<FileExcelOutlined />}
                  onClick={handleExport}
                  loading={exporting}
                  disabled={rows.length === 0}
                >
                  원본 스캔 엑셀
                </Button>
                <Button onClick={() => setDiffOpen(true)}>
                  기대 리스트 비교
                  {diffActive && (
                    <Tag
                      color={diffStats.missing.length > 0 ? "red" : "green"}
                      style={{ marginLeft: 6, fontSize: 11 }}
                    >
                      {diffStats.matched.length}/{expectedSerials.length}
                    </Tag>
                  )}
                </Button>
              </Space>

              {rows.some((r) => r.error) && (
                <Alert
                  type="warning"
                  showIcon
                  message="일부 이미지에서 바코드를 찾지 못했습니다. 파일 열의 ⚠에 마우스를 올리면 원인을 볼 수 있습니다."
                />
              )}
              {duplicateSerials.size > 0 && (
                <Alert
                  type="warning"
                  showIcon
                  message={`중복 시리얼 ${duplicateSerials.size}건 발견.`}
                />
              )}
              {lowConfidenceCount > 0 && (
                <Alert
                  type="warning"
                  showIcon
                  message={`OCR 신뢰도 낮은 행 ${lowConfidenceCount}건 — 시리얼 칸이 노란색으로 강조됩니다.`}
                />
              )}
              {diffActive && (
                <Alert
                  type={diffStats.missing.length > 0 ? "warning" : "success"}
                  showIcon
                  message={
                    `기대 ${expectedSerials.length}건 중 ${diffStats.matched.length}건 매칭, ` +
                    `누락 ${diffStats.missing.length}건, 잉여 ${diffStats.extra.length}건.`
                  }
                  action={
                    <Button size="small" onClick={() => setDiffOpen(true)}>
                      자세히
                    </Button>
                  }
                />
              )}
              {palletStats.length > 0 && (
                <Space wrap size={[8, 8]}>
                  <Text strong>팔레트별 모듈:</Text>
                  {palletStats.map(({ pallet, count }) => (
                    <Tag
                      key={pallet}
                      color={palletColors.get(pallet) ?? "default"}
                      style={{ fontSize: 12 }}
                    >
                      {pallet} — {count}개
                    </Tag>
                  ))}
                </Space>
              )}

              <Table<Row>
                rowKey="_rowId"
                columns={columns}
                dataSource={rows}
                pagination={false}
                scroll={{ x: "max-content" }}
                bordered
                size="small"
                locale={{
                  emptyText:
                    "아직 결과가 없습니다. 이미지를 업로드하고 '스캔 실행'을 눌러주세요.",
                }}
              />
            </Space>
          </Card>

          <Card
            size="small"
            title="2. 마스터 엑셀 매칭 & 리포트 생성"
            extra={
              <Button
                icon={<FileSearchOutlined />}
                onClick={handleLoadMaster}
                loading={loadingMaster}
              >
                마스터 엑셀 선택
              </Button>
            }
          >
            {master ? (
              <Space direction="vertical" size="middle" style={{ width: "100%" }}>
                <Descriptions size="small" column={2} bordered>
                  <Descriptions.Item label="파일">{master.filename}</Descriptions.Item>
                  <Descriptions.Item label="시트">
                    {master.sheets.length}개
                  </Descriptions.Item>
                  <Descriptions.Item label="총 행">
                    {master.total_rows.toLocaleString()}
                  </Descriptions.Item>
                  <Descriptions.Item label="시리얼 인덱스">
                    {master.index_size.toLocaleString()}
                  </Descriptions.Item>
                </Descriptions>

                {master.has_preset && (
                  <Alert
                    type="info"
                    showIcon
                    message="저장된 프리셋을 자동으로 채웠습니다. 필요하면 아래에서 수정하세요."
                  />
                )}

                {(master.recent_presets ?? []).length > 0 && (
                  <Form layout="vertical" style={{ marginBottom: -8 }}>
                    <Form.Item
                      label="이전 조합 불러오기"
                      tooltip="과거에 썼던 Module type / Cell Type 조합을 골라 두 필드를 한번에 채울 수 있습니다."
                    >
                      <Select
                        placeholder="이전 조합 선택 (선택사항)"
                        allowClear
                        options={(master.recent_presets ?? []).map((p, i) => ({
                          value: `${p.module_type}|${p.cell_type}|${i}`,
                          label: `${p.module_type || "(비어있음)"} / ${p.cell_type || "(비어있음)"}`,
                        }))}
                        onChange={(v) => {
                          if (!v) return;
                          const idx = Number(String(v).split("|").pop());
                          const chosen = (master.recent_presets ?? [])[idx];
                          if (chosen) {
                            setModuleType(chosen.module_type);
                            setCellType(chosen.cell_type);
                          }
                        }}
                      />
                    </Form.Item>
                  </Form>
                )}

                <Form layout="vertical">
                  <AntRow gutter={16}>
                    <Col span={8}>
                      <Form.Item
                        label="Module type"
                        required
                        tooltip="예: JKM640N-78HL4-BDV-S1"
                      >
                        <AutoComplete
                          value={moduleType}
                          onChange={(v) => setModuleType(v)}
                          options={Array.from(
                            new Set(
                              (master.recent_presets ?? [])
                                .map((p) => p.module_type)
                                .filter(Boolean)
                            )
                          ).map((v) => ({ value: v }))}
                          placeholder="JKM640N-78HL4-BDV-S1"
                          filterOption={(input, option) =>
                            String(option?.value ?? "")
                              .toLowerCase()
                              .includes(input.toLowerCase())
                          }
                        />
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item
                        label="Cell Type"
                        required
                        tooltip="예: M183R-16-C"
                      >
                        <AutoComplete
                          value={cellType}
                          onChange={(v) => setCellType(v)}
                          options={Array.from(
                            new Set(
                              (master.recent_presets ?? [])
                                .map((p) => p.cell_type)
                                .filter(Boolean)
                            )
                          ).map((v) => ({ value: v }))}
                          placeholder="M183R-16-C"
                          filterOption={(input, option) =>
                            String(option?.value ?? "")
                              .toLowerCase()
                              .includes(input.toLowerCase())
                          }
                        />
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item label="프로젝트명" required>
                        <AutoComplete
                          value={projectName}
                          onChange={(v) => setProjectName(v)}
                          options={(master.recent_projects ?? []).map((p) => ({
                            value: p,
                          }))}
                          placeholder="진달래2"
                        />
                      </Form.Item>
                    </Col>
                  </AntRow>
                  <Space>
                    <Checkbox
                      checked={autoNumber}
                      onChange={(e) => setAutoNumber(e.target.checked)}
                    >
                      NO 열 자동 순번
                    </Checkbox>
                    <Checkbox
                      checked={savePreset}
                      onChange={(e) => setSavePreset(e.target.checked)}
                    >
                      이 마스터에 Module/Cell Type 저장 (다음에 자동 채움)
                    </Checkbox>
                  </Space>
                </Form>

                <Divider style={{ margin: "8px 0" }} />

                <Space>
                  <Button
                    type="primary"
                    icon={<ExportOutlined />}
                    onClick={handleBuild}
                    loading={building}
                    disabled={validSerials.length === 0}
                  >
                    리포트 생성 ({validSerials.length}건)
                  </Button>
                  <Text type="secondary">
                    매칭 안 되는 시리얼은 노란색 빈 행으로 포함됩니다.
                  </Text>
                </Space>

                {lastBuild && (
                  <AntRow gutter={16}>
                    <Col span={8}>
                      <Statistic title="매칭" value={lastBuild.matched} />
                    </Col>
                    <Col span={8}>
                      <Statistic
                        title="누락"
                        value={lastBuild.missing}
                        valueStyle={{
                          color: lastBuild.missing > 0 ? "#d46b08" : undefined,
                        }}
                      />
                    </Col>
                    <Col span={8}>
                      <Statistic title="총 행" value={lastBuild.lines.length} />
                    </Col>
                  </AntRow>
                )}
                {lastBuild && lastBuild.missing > 0 && (
                  <Alert
                    type="warning"
                    showIcon
                    message={`마스터에 없는 시리얼 ${lastBuild.missing}건`}
                    description={lastBuild.lines
                      .filter((l) => !l.found)
                      .map((l) => l.serial)
                      .join(", ")}
                  />
                )}
              </Space>
            ) : (
              <Text type="secondary">
                마스터 엑셀을 선택하면 시리얼을 자동 매칭하여 샘플 포맷으로
                리포트를 생성합니다.
              </Text>
            )}
          </Card>
        </Space>
      </Content>

      <Modal
        title="기대 시리얼 비교"
        open={diffOpen}
        onCancel={() => setDiffOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setExpectedText("")} disabled={!expectedText}>
              비우기
            </Button>
            <Button type="primary" onClick={() => setDiffOpen(false)}>
              닫기
            </Button>
          </Space>
        }
        width={720}
      >
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            기대 시리얼을 한 줄에 하나씩 붙여넣으세요. 쉼표·탭·공백 구분도 됩니다. 입력은 모달을 닫아도 유지됩니다.
          </Text>
          <Input.TextArea
            value={expectedText}
            onChange={(e) => setExpectedText(e.target.value)}
            placeholder={"PC2025...\nPC2025...\nPC2025..."}
            autoSize={{ minRows: 6, maxRows: 14 }}
            style={{ fontFamily: "ui-monospace, 'Cascadia Mono', Consolas, monospace", fontSize: 13 }}
          />
          {diffActive ? (
            <>
              <AntRow gutter={16}>
                <Col span={6}>
                  <Statistic title="기대" value={expectedSerials.length} />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="매칭"
                    value={diffStats.matched.length}
                    valueStyle={{ color: "#389e0d" }}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="누락"
                    value={diffStats.missing.length}
                    valueStyle={{
                      color: diffStats.missing.length > 0 ? "#d4380d" : undefined,
                    }}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="잉여"
                    value={diffStats.extra.length}
                    valueStyle={{
                      color: diffStats.extra.length > 0 ? "#d46b08" : undefined,
                    }}
                  />
                </Col>
              </AntRow>
              {diffStats.missing.length > 0 && (
                <Alert
                  type="error"
                  showIcon
                  message={`누락 ${diffStats.missing.length}건 — 스캔 결과에 없음`}
                  description={
                    <Input.TextArea
                      readOnly
                      value={diffStats.missing.join("\n")}
                      autoSize={{ minRows: 2, maxRows: 8 }}
                      style={{
                        fontFamily:
                          "ui-monospace, 'Cascadia Mono', Consolas, monospace",
                        fontSize: 12,
                        marginTop: 4,
                      }}
                    />
                  }
                />
              )}
              {diffStats.extra.length > 0 && (
                <Alert
                  type="warning"
                  showIcon
                  message={`잉여 ${diffStats.extra.length}건 — 기대 리스트에 없음`}
                  description={
                    <Input.TextArea
                      readOnly
                      value={diffStats.extra.join("\n")}
                      autoSize={{ minRows: 2, maxRows: 8 }}
                      style={{
                        fontFamily:
                          "ui-monospace, 'Cascadia Mono', Consolas, monospace",
                        fontSize: 12,
                        marginTop: 4,
                      }}
                    />
                  }
                />
              )}
              {missingSet.size === 0 && extraSet.size === 0 && (
                <Alert
                  type="success"
                  showIcon
                  message="모든 시리얼이 기대 리스트와 정확히 일치합니다."
                />
              )}
            </>
          ) : (
            <Text type="secondary">
              위에 시리얼을 붙여넣으면 매칭/누락/잉여 통계가 표시됩니다.
            </Text>
          )}
        </Space>
      </Modal>

      {previewIdx !== null && rows[previewIdx] && (() => {
        // Skip rows that have no preview (PDFs, manual entries, errors)
        // so arrow nav jumps to the next/prev *previewable* row.
        const isPreviewable = (r: Row) =>
          canPreview(r) && imageUrls.has(r.filename);
        const findStep = (from: number, dir: 1 | -1): number | null => {
          for (let i = from + dir; i >= 0 && i < rows.length; i += dir) {
            if (isPreviewable(rows[i])) return i;
          }
          return null;
        };
        const prevIdx = findStep(previewIdx, -1);
        const nextIdx = findStep(previewIdx, 1);
        return (
          <ImagePreviewModal
            row={rows[previewIdx]}
            imageUrl={imageUrls.get(rows[previewIdx].filename)}
            hasPrev={prevIdx !== null}
            hasNext={nextIdx !== null}
            onPrev={() => prevIdx !== null && setPreviewIdx(prevIdx)}
            onNext={() => nextIdx !== null && setPreviewIdx(nextIdx)}
            onClose={() => setPreviewIdx(null)}
          />
        );
      })()}
    </Layout>
  );
}
