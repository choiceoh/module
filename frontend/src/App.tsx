import { useState } from "react";
import {
  Button,
  Layout,
  Space,
  Table,
  Typography,
  Upload,
  message,
  Popconfirm,
  Input,
  Alert,
} from "antd";
import type { UploadFile } from "antd/es/upload/interface";
import {
  InboxOutlined,
  FileExcelOutlined,
  DeleteOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import {
  MODULE_COLUMNS,
  emptyModule,
  type Module,
  type Row,
} from "./types";
import { downloadBlob, exportExcel, extractModules } from "./api";

const { Header, Content } = Layout;
const { Title, Text } = Typography;

let rowSeq = 0;
const newRowId = () => `row-${Date.now()}-${++rowSeq}`;

export default function App() {
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [rows, setRows] = useState<Row[]>([]);
  const [extracting, setExtracting] = useState(false);
  const [exporting, setExporting] = useState(false);

  const handleExtract = async () => {
    const files = fileList
      .map((f) => f.originFileObj as File | undefined)
      .filter((f): f is File => !!f);
    if (files.length === 0) {
      message.warning("먼저 이미지를 업로드해주세요.");
      return;
    }
    setExtracting(true);
    try {
      const results = await extractModules(files);
      const newRows: Row[] = results.map((r) => ({
        ...(r.module ?? emptyModule()),
        _rowId: newRowId(),
        _filename: r.filename,
        _error: r.error,
      }));
      setRows((prev) => [...prev, ...newRows]);
      const okCount = results.filter((r) => r.module).length;
      message.success(`${okCount}/${results.length}건 추출 완료`);
      setFileList([]);
    } catch (e) {
      message.error(String(e));
    } finally {
      setExtracting(false);
    }
  };

  const handleExport = async () => {
    if (rows.length === 0) {
      message.warning("내보낼 데이터가 없습니다.");
      return;
    }
    setExporting(true);
    try {
      const modules: Module[] = rows.map(({ _rowId, _filename, _error, ...m }) => {
        void _rowId; void _filename; void _error;
        return m;
      });
      const blob = await exportExcel(modules);
      downloadBlob(blob, `modules-${new Date().toISOString().slice(0, 10)}.xlsx`);
    } catch (e) {
      message.error(String(e));
    } finally {
      setExporting(false);
    }
  };

  const updateCell = (rowId: string, key: keyof Module, value: string) => {
    setRows((prev) =>
      prev.map((r) => (r._rowId === rowId ? { ...r, [key]: value } : r))
    );
  };

  const addEmptyRow = () => {
    setRows((prev) => [
      ...prev,
      { ...emptyModule(), _rowId: newRowId(), _filename: "(수동 입력)" },
    ]);
  };

  const deleteRow = (rowId: string) => {
    setRows((prev) => prev.filter((r) => r._rowId !== rowId));
  };

  const clearAll = () => setRows([]);

  const columns = [
    {
      title: "파일",
      dataIndex: "_filename",
      key: "_filename",
      width: 160,
      fixed: "left" as const,
      render: (v: string, r: Row) =>
        r._error ? (
          <Text type="danger" title={r._error}>
            {v} ⚠
          </Text>
        ) : (
          <Text>{v}</Text>
        ),
    },
    ...MODULE_COLUMNS.map((c) => ({
      title: c.label,
      dataIndex: c.key,
      key: c.key,
      width: 140,
      render: (v: string, r: Row) => (
        <Input
          value={v}
          size="small"
          variant="borderless"
          onChange={(e) => updateCell(r._rowId, c.key, e.target.value)}
        />
      ),
    })),
    {
      title: "",
      key: "_actions",
      width: 60,
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
      <Header style={{ background: "#fff", borderBottom: "1px solid #f0f0f0" }}>
        <Title level={3} style={{ margin: 0, lineHeight: "64px" }}>
          모듈 정보 추출기
        </Title>
      </Header>
      <Content style={{ padding: 24, maxWidth: 1400, margin: "0 auto", width: "100%" }}>
        <Space direction="vertical" size="large" style={{ width: "100%" }}>
          <Upload.Dragger
            multiple
            accept="image/*"
            fileList={fileList}
            beforeUpload={() => false}
            onChange={(info) => setFileList(info.fileList)}
            style={{ padding: 12 }}
          >
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">
              모듈 사진/데이터시트 이미지를 이 영역에 끌어놓거나 클릭해서 업로드
            </p>
            <p className="ant-upload-hint">여러 장을 한꺼번에 올릴 수 있습니다 (각 20MB 이하).</p>
          </Upload.Dragger>

          <Space>
            <Button
              type="primary"
              onClick={handleExtract}
              loading={extracting}
              disabled={fileList.length === 0}
            >
              추출 실행 ({fileList.length})
            </Button>
            <Button icon={<PlusOutlined />} onClick={addEmptyRow}>
              빈 행 추가
            </Button>
            <Popconfirm title="모든 결과를 지울까요?" onConfirm={clearAll} okText="지우기" cancelText="취소">
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
              엑셀 다운로드
            </Button>
          </Space>

          {rows.some((r) => r._error) && (
            <Alert
              type="warning"
              showIcon
              message="일부 항목에서 오류가 발생했습니다. 파일 열의 ⚠ 마크에 마우스를 올리면 원인을 볼 수 있습니다."
            />
          )}

          <Table<Row>
            rowKey="_rowId"
            columns={columns}
            dataSource={rows}
            pagination={false}
            scroll={{ x: "max-content" }}
            bordered
            size="small"
            locale={{ emptyText: "아직 결과가 없습니다. 이미지를 업로드하고 '추출 실행'을 눌러주세요." }}
          />
        </Space>
      </Content>
    </Layout>
  );
}
