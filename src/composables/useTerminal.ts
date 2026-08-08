/**
 * Composable for managing an xterm.js terminal instance tied to a session.
 * Wires terminal I/O to the RPC client (terminal.data notifications + ssh.write calls).
 */
import { ref, onMounted, onBeforeUnmount, type Ref } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SearchAddon } from "@xterm/addon-search";
import "@xterm/xterm/css/xterm.css";
import { rpcClient } from "@/lib/rpc-client";

export function useTerminal(
  containerRef: Ref<HTMLElement | null>,
  sessionId: Ref<string>
) {
  const terminal = ref<Terminal | null>(null);
  const fitAddon = ref<FitAddon | null>(null);
  const searchAddon = ref<SearchAddon | null>(null);
  let unsubscribe: (() => void) | null = null;

  onMounted(() => {
    if (!containerRef.value) return;

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "JetBrains Mono, monospace",
      theme: {
        background: "#0d1117",
        foreground: "#c9d1d9",
      },
    });

    const fit = new FitAddon();
    const search = new SearchAddon();
    term.loadAddon(fit);
    term.loadAddon(search);
    term.open(containerRef.value);
    fit.fit();

    terminal.value = term;
    fitAddon.value = fit;
    searchAddon.value = search;

    // Show connection info
    term.writeln("\x1b[33m[DevTerm] Connected to session: " + sessionId.value + "\x1b[0m");
    term.writeln("\x1b[33m[DevTerm] Waiting for remote shell output...\x1b[0m");
    term.writeln("");

    // Send user keystrokes to the backend
    term.onData((data) => {
      rpcClient.call("ssh.write", {
        sessionId: sessionId.value,
        data,
      });
    });

    // Receive terminal output from the backend
    unsubscribe = rpcClient.subscribe("terminal.data", (params: unknown) => {
      const p = params as { sessionId: string; chunk: string };
      if (p.sessionId === sessionId.value) {
        term.write(p.chunk);
      }
    });

    // Handle resize with debounce for performance
    let resizeTimer: ReturnType<typeof setTimeout> | null = null;
    const observer = new ResizeObserver(() => {
      if (resizeTimer) clearTimeout(resizeTimer);
      resizeTimer = setTimeout(() => {
        fit.fit();
        rpcClient.call("ssh.resize", {
          sessionId: sessionId.value,
          cols: term.cols,
          rows: term.rows,
        });
      }, 50);
    });
    observer.observe(containerRef.value);
  });

  onBeforeUnmount(() => {
    unsubscribe?.();
    terminal.value?.dispose();
  });

  function searchText(query: string) {
    searchAddon.value?.findNext(query);
  }

  function searchPrevious(query: string) {
    searchAddon.value?.findPrevious(query);
  }

  return { terminal, searchText, searchPrevious };
}
