import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App";

const theme = localStorage.getItem("loop_theme");
if (theme === "light" || theme === "dark") {
  document.documentElement.dataset.theme = theme;
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
