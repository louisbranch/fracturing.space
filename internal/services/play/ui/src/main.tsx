import React from "react";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import { canonicalizeWindowLocation } from "./app_mode";
import "./styles.css";

const rootElement = document.getElementById("root");

if (!rootElement) {
  throw new Error("missing root element");
}

const mode = canonicalizeWindowLocation();

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <App mode={mode} />
  </React.StrictMode>,
);
