/**
 * Composable for managing an xterm.js terminal instance tied to a session.
 * Features: themes, copy/paste, search, resize debounce, batched writes, reactivation fit.
 */
import { ref, onMounted, onActivated, onBeforeUnmount, watch, type Ref } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SearchAddon } from "@xterm/addon-search";
import "@xterm/xterm/css/xterm.css";
import { rpcClient } from "@/lib/rpc-client";
import { terminalThemes, defaultThemeName } from "@/lib/terminal-themes";
import { useSettingsStore } from "@/stores/settings";

// Dynamic scrollback: cap at this to prevent memory issues
const MAX_SCROLLBACK = 5000;

export function useTerminal(
  containerRef: Ref<HTMLElement | null>,
  sessionId: Ref<string>
) {
  const terminal = ref<Terminal | null>(null);
  const fitAddon = ref<FitAddon | null>(null);
  const searchAddon = ref<SearchAddon | null>(null);
  const settingsStore = useSettingsStore();
  let unsubscribe: (() => void) | null = null;
  let unsubDisconnect: (() => void) | null = null;
  let resizeObserver: ResizeObserver | null = null;

  onMounted(() => {
    if (!containerRef.value) return;

    const themeName = settingsStore.settings.terminalTheme || defaultThemeName;
    const themeConfig = terminalThemes[themeName] || terminalThemes[defaultThemeName];

    const term = new Terminal({
      cursorBlink: true,
      fontSize: settingsStore.settings.fontSize,
      fontFamily: settingsStore.settings.fontFamily,
      theme: themeConfig.theme,
      allowProposedApi: true,
      scrollback: MAX_SCROLLBACK,
      convertEol: true,
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

    // Send user keystrokes to the backend
    term.onData((data) => {
      rpcClient.call("ssh.write", {
        sessionId: sessionId.value,
        data,
      });
    });

    // Handle binary data
    term.onBinary((data) => {
      rpcClient.call("ssh.write", {
        sessionId: sessionId.value,
        data,
      });
    });

    // Receive terminal output (batched for performance)
    let writeBuf = "";
    let writeTimer: ReturnType<typeof setTimeout> | null = null;

    unsubscribe = rpcClient.subscribe("terminal.data", (params: unknown) => {
      const p = params as { sessionId: string; chunk: string };
      if (p.sessionId === sessionId.value) {
        writeBuf += p.chunk;
        if (!writeTimer) {
          writeTimer = setTimeout(() => {
            if (writeBuf) {
              term.write(writeBuf);
              writeBuf = "";
            }
            writeTimer = null;
          }, 16);
        }
      }
    });

    // Detect unexpected disconnects
    unsubDisconnect = rpcClient.subscribe("ssh.status", (params: unknown) => {
      const p = params as { sessionId: string; status: string };
      if (p.sessionId === sessionId.value && p.status === "disconnected") {
        term.writeln("");
        term.writeln("\x1b[31m[DevTerm] Connection lost.\x1b[0m");
      }
    });

    // Handle resize with debounce
    let resizeTimer: ReturnType<typeof setTimeout> | null = null;
    resizeObserver = new ResizeObserver(() => {
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
    resizeObserver.observe(containerRef.value);

    // Right-click paste
    containerRef.value.addEventListener("contextmenu", async (e) => {
      e.preventDefault();
      try {
        const text = await navigator.clipboard.readText();
        if (text) {
          rpcClient.call("ssh.write", { sessionId: sessionId.value, data: text });
        }
      } catch {
        // Clipboard denied
      }
    });
  });

  // Fix #1: Re-fit terminal when KeepAlive reactivates this component
  onActivated(() => {
    if (fitAddon.value && terminal.value) {
      // Small delay to let the DOM layout settle
      setTimeout(() => {
        fitAddon.value?.fit();
      }, 20);
    }
  });

  // Live theme update (#5)
  watch(
    () => settingsStore.settings.terminalTheme,
    (themeName) => {
      if (!terminal.value) return;
      const name = themeName || defaultThemeName;
      const themeConfig = terminalThemes[name] || terminalThemes[defaultThemeName];
      terminal.value.options.theme = themeConfig.theme;
    }
  );

  watch(
    () => settingsStore.settings.fontSize,
    (size) => {
      if (!terminal.value) return;
      terminal.value.options.fontSize = size;
      fitAddon.value?.fit();
    }
  );

  watch(
    () => settingsStore.settings.fontFamily,
    (family) => {
      if (!terminal.value) return;
      terminal.value.options.fontFamily = family;
      fitAddon.value?.fit();
    }
  );

  onBeforeUnmount(() => {
    unsubscribe?.();
    unsubDisconnect?.();
    resizeObserver?.disconnect();
    terminal.value?.dispose();
  });

  function searchText(query: string) {
    searchAddon.value?.findNext(query);
  }

  function searchPrevious(query: string) {
    searchAddon.value?.findPrevious(query);
  }

  function copySelection() {
    if (!terminal.value) return;
    const selection = terminal.value.getSelection();
    if (selection) {
      navigator.clipboard.writeText(selection);
    }
  }

  function paste() {
    navigator.clipboard.readText().then((text) => {
      if (text) {
        rpcClient.call("ssh.write", { sessionId: sessionId.value, data: text });
      }
    });
  }

  function clear() {
    terminal.value?.clear();
  }

  return { terminal, fitAddon, searchText, searchPrevious, copySelection, paste, clear };
}
