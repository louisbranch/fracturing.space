import React from "react";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import { canonicalizeWindowLocation } from "./app_mode";
import { readShellConfig } from "./shell_config";
import "./styles.css";

const rootElement = document.getElementById("root");

if (!rootElement) {
  throw new Error("missing root element");
}

const mode = canonicalizeWindowLocation();
const shellConfig = readShellConfig();

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <App mode={mode} shellConfig={shellConfig} />
  </React.StrictMode>,
);
