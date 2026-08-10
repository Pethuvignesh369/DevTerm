/**
 * Composable for managing an xterm.js terminal instance tied to a session.
 * Features: themes, copy/paste, search, resize debounce, reconnect.
 */
import { ref, onMounted, onBeforeUnmount, watch, type Ref } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SearchAddon } from "@xterm/addon-search";
import "@xterm/xterm/css/xterm.css";
import { rpcClient } from "@/lib/rpc-client";
import { terminalThemes, defaultThemeName } from "@/lib/terminal-themes";
import { useSettingsStore } from "@/stores/settings";

export function useTerminal(
  containerRef: Ref<HTMLElement | null>,
  sessionId: Ref<string>
) {
  const terminal = ref<Terminal | null>(null);
  const fitAddon = ref<FitAddon | null>(null);
  const searchAddon = ref<SearchAddon | null>(null);
  const settingsStore = useSettingsStore();
  let unsubscribe: (() => void) | null = null;
  let resizeObserver: ResizeObserver | null = null;

  onMounted(() => {
    if (!containerRef.value) return;

    const themeName = (settingsStore.settings as { terminalTheme?: string }).terminalTheme || defaultThemeName;
    const themeConfig = terminalThemes[themeName] || terminalThemes[defaultThemeName];

    const term = new Terminal({
      cursorBlink: true,
      fontSize: settingsStore.settings.fontSize,
      fontFamily: settingsStore.settings.fontFamily,
      theme: themeConfig.theme,
      allowProposedApi: true,
      scrollback: 10000,
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

    // Handle binary data (for copy/paste with special chars)
    term.onBinary((data) => {
      rpcClient.call("ssh.write", {
        sessionId: sessionId.value,
        data,
      });
    });

    // Receive terminal output from the backend (batched for performance)
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
          }, 16); // ~60fps batch
        }
      }
    });

    // Handle resize with debounce for performance
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
          rpcClient.call("ssh.write", {
            sessionId: sessionId.value,
            data: text,
          });
        }
      } catch {
        // Clipboard access denied
      }
    });
  });

  // Watch for theme changes
  watch(
    () => (settingsStore.settings as { terminalTheme?: string }).terminalTheme,
    (themeName) => {
      if (!terminal.value) return;
      const name = themeName || defaultThemeName;
      const themeConfig = terminalThemes[name] || terminalThemes[defaultThemeName];
      terminal.value.options.theme = themeConfig.theme;
    }
  );

  // Watch for font changes
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
        rpcClient.call("ssh.write", {
          sessionId: sessionId.value,
          data: text,
        });
      }
    });
  }

  function clear() {
    terminal.value?.clear();
  }

  return { terminal, searchText, searchPrevious, copySelection, paste, clear };
}
