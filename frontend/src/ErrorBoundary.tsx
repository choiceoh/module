import { Component, type ReactNode } from "react";
import { Alert, Button, Space, Typography } from "antd";

type Props = { children: ReactNode };
type State = { error: Error | null };

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: { componentStack?: string | null }) {
    console.error("ErrorBoundary caught:", error, info);
  }

  reset = () => this.setState({ error: null });

  render() {
    if (!this.state.error) return this.props.children;
    return (
      <div style={{ padding: 24, maxWidth: 900, margin: "40px auto" }}>
        <Alert
          type="error"
          showIcon
          message="렌더링 오류가 발생했습니다"
          description={
            <Space direction="vertical" style={{ width: "100%" }}>
              <Typography.Text strong>{this.state.error.message}</Typography.Text>
              <Typography.Paragraph>
                <pre style={{ whiteSpace: "pre-wrap", fontSize: 11, margin: 0 }}>
                  {this.state.error.stack}
                </pre>
              </Typography.Paragraph>
              <Button onClick={this.reset}>다시 시도</Button>
            </Space>
          }
        />
      </div>
    );
  }
}
