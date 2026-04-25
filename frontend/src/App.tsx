import { useEffect, useMemo, useState } from "react";
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
  Switch,
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
  SettingOutlined,
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
  getSettings,
  getVersion,
  pickAndLoadMaster,
  saveSettings,
  scanFiles,
  type BuildResult,
  type MasterInfo,
  type Settings,
} from "./api";

const { Header, Content } = Layout;
const { Title, Text } = Typography;

let rowSeq = 0;
const newRowId = () => `row-${Date.now()}-${++rowSeq}`;

const sourceColor: Record<ScanSource, string> = {
  barcode: "green",
  manual: "blue",
  vllm: "orange",
};

const sourceLabel: Record<ScanSource, string> = {
  barcode: "바코드",
  manual: "수동",
  vllm: "VLM",
};

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

  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settings, setSettings] = useState<Settings>({
    vllm_base_url: "",
    vllm_model: "",
    use_vllm_fallback: false,
  });
  const [settingsDraft, setSettingsDraft] = useState<Settings>(settings);
  const [version, setVersion] = useState<string>("");

  useEffect(() => {
    getVersion()
      .then(setVersion)
      .catch(() => {});
  }, []);

  useEffect(() => {
    getSettings()
      .then((s) => {
        setSettings(s);
        setSettingsDraft(s);
      })
      .catch(() => {});
  }, []);

  const openSettings = () => {
    setSettingsDraft(settings);
    setSettingsOpen(true);
  };

  const handleSaveSettings = async () => {
    try {
      await saveSettings(settingsDraft);
      setSettings(settingsDraft);
      setSettingsOpen(false);
      message.success("설정 저장됨");
    } catch (e) {
      message.error(String(e));
    }
  };

  useEffect(() => {
    if (master?.has_preset) {
      setModuleType(master.preset.module_type);
      setCellType(master.preset.cell_type);
    }
  }, [master]);

  const duplicateSerials = useMemo(() => {
    const counts = new Map<string, number>();
    for (const r of rows) {
      if (!r.serial) continue;
      counts.set(r.serial, (counts.get(r.serial) ?? 0) + 1);
    }
    const dups = new Set<string>();
    counts.forEach((c, s) => {
      if (c > 1) dups.add(s);
    });
    return dups;
  }, [rows]);

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

  const handleScan = async () => {
    const files = fileList
      .map((f) => f.originFileObj as File | undefined)
      .filter((f): f is File => !!f);
    if (files.length === 0) {
      message.warning("먼저 이미지를 업로드해주세요.");
      return;
    }
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

  const clearAll = () => setRows([]);

  const cellStyle = { fontSize: 12 };
  const columns = [
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
      render: (v: string, r: Row) => (
        <Input
          value={v}
          size="small"
          variant="borderless"
          style={{ fontSize: 15, fontFamily: "ui-monospace, 'Cascadia Mono', Consolas, monospace" }}
          onChange={(e) => updateCell(r._rowId, "serial", e.target.value)}
          status={v && duplicateSerials.has(v) ? "warning" : undefined}
          suffix={
            v && duplicateSerials.has(v) ? (
              <Tag color="orange" style={{ margin: 0, fontSize: 10 }}>
                중복
              </Tag>
            ) : null
          }
        />
      ),
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
          justifyContent: "space-between",
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
        <Button icon={<SettingOutlined />} onClick={openSettings}>
          설정
        </Button>
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
        title="설정"
        open={settingsOpen}
        onCancel={() => setSettingsOpen(false)}
        onOk={handleSaveSettings}
        okText="저장"
        cancelText="취소"
        width={560}
      >
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <Alert
            type="info"
            showIcon
            message="vLLM (OCR 폴백용) — 현재 미호출. 추후 바코드 디코딩 실패 시 이미지의 하단 시리얼 텍스트를 vLLM으로 읽는 폴백 기능에 사용될 값."
          />
          <Form layout="vertical">
            <Form.Item label="vLLM Base URL" tooltip="OpenAI 호환 엔드포인트 (/v1)">
              <Input
                value={settingsDraft.vllm_base_url}
                onChange={(e) =>
                  setSettingsDraft({ ...settingsDraft, vllm_base_url: e.target.value })
                }
                placeholder="https://.../v1 또는 http://localhost:8000/v1"
              />
            </Form.Item>
            <Form.Item label="모델명">
              <Input
                value={settingsDraft.vllm_model}
                onChange={(e) =>
                  setSettingsDraft({ ...settingsDraft, vllm_model: e.target.value })
                }
                placeholder="gemma4"
              />
            </Form.Item>
            <Form.Item
              label="vLLM 자동 사용 (테이블일 때만)"
              tooltip="바코드가 5개 이상 검출되면 포장 명세서로 판단해 vLLM을 추가 호출하고 결과를 합칩니다. 단일 라벨 사진은 빠른 바코드만 사용."
            >
              <Switch
                checked={settingsDraft.use_vllm_fallback}
                onChange={(v) =>
                  setSettingsDraft({ ...settingsDraft, use_vllm_fallback: v })
                }
              />
              <Text type="secondary" style={{ marginLeft: 8 }}>
                {settingsDraft.use_vllm_fallback
                  ? "테이블 감지 시 vLLM 추가 호출"
                  : "바코드만 (빠름)"}
              </Text>
            </Form.Item>
          </Form>
          <Text type="secondary" style={{ fontSize: 12 }}>
            저장 위치: <code>~/.module-scanner/presets.json</code>
          </Text>
        </Space>
      </Modal>
    </Layout>
  );
}
