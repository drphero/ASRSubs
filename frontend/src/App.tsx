import { useEffect } from "react";
import { AppShell } from "./features/shell/AppShell";

const selectedModel = "Qwen3-ASR-0.6B";

export default function App() {
  useEffect(() => {
    document.documentElement.dataset.theme = "dark";
  }, []);

  return <AppShell selectedModel={selectedModel} hasSelection={false} />;
}
