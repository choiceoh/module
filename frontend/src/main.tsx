import React from "react";
import ReactDOM from "react-dom/client";
import { ConfigProvider } from "antd";
import koKR from "antd/locale/ko_KR";
import App from "./App";
import { ErrorBoundary } from "./ErrorBoundary";
import "antd/dist/reset.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ConfigProvider locale={koKR}>
      <ErrorBoundary>
        <App />
      </ErrorBoundary>
    </ConfigProvider>
  </React.StrictMode>
);
