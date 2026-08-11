/**
 * Composable for managing an xterm.js terminal instance.
 * Features: themes, copy/paste, search, resize, bell, links, zoom, bracket paste, batched writes.
 */
import { ref, onMounted, onActivated, onBeforeUnmount, watch, type Ref } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SearchAddon } from "@xterm/addon-search";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import { rpcClient } from "@/lib/rpc-client";
import { terminalThemes, defaultThemeName } from "@/lib/terminal-themes";
import { useSettingsStore } from "@/stores/settings";
import { playNotificationSound } from "@/lib/sounds";

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
    const webLinks = new WebLinksAddon((_, uri) => {
      window.open(uri, "_blank");
    });

    term.loadAddon(fit);
    term.loadAddon(search);
    term.loadAddon(webLinks);
    term.open(containerRef.value);
    fit.fit();

    terminal.value = term;
    fitAddon.value = fit;
    searchAddon.value = search;

    // Bell sound
    term.onBell(() => {
      playNotificationSound();
    });

    // Send keystrokes
    term.onData((data) => {
      rpcClient.call("ssh.write", { sessionId: sessionId.value, data }).catch(() => undefined);
    });

    term.onBinary((data) => {
      rpcClient.call("ssh.write", { sessionId: sessionId.value, data }).catch(() => undefined);
    });

    // Batched writes (16ms = 60fps)
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

    // Disconnect detection
    unsubDisconnect = rpcClient.subscribe("ssh.status", (params: unknown) => {
      const p = params as { sessionId: string; status: string };
      if (p.sessionId === sessionId.value && p.status === "disconnected") {
        term.writeln("");
        term.writeln("\x1b[31m[DevTerm] Connection lost.\x1b[0m");
      }
    });

    // Debounced resize
    let resizeTimer: ReturnType<typeof setTimeout> | null = null;
    resizeObserver = new ResizeObserver(() => {
      if (resizeTimer) clearTimeout(resizeTimer);
      resizeTimer = setTimeout(() => {
        fit.fit();
        rpcClient.call("ssh.resize", {
          sessionId: sessionId.value,
          cols: term.cols,
          rows: term.rows,
        }).catch(() => undefined);
      }, 50);
    });
    resizeObserver.observe(containerRef.value);

    // Right-click paste
    containerRef.value.addEventListener("contextmenu", async (e) => {
      e.preventDefault();
      try {
        const text = await navigator.clipboard.readText();
        if (text) rpcClient.call("ssh.write", { sessionId: sessionId.value, data: text }).catch(() => undefined);
      } catch { /* denied */ }
    });
  });

  // Re-fit on KeepAlive reactivation
  onActivated(() => {
    if (fitAddon.value) {
      setTimeout(() => fitAddon.value?.fit(), 20);
    }
  });

  // Live theme updates
  watch(() => settingsStore.settings.terminalTheme, (name) => {
    if (!terminal.value) return;
    const t = terminalThemes[name || defaultThemeName] || terminalThemes[defaultThemeName];
    terminal.value.options.theme = t.theme;
  });

  watch(() => settingsStore.settings.fontSize, (size) => {
    if (!terminal.value) return;
    terminal.value.options.fontSize = size;
    fitAddon.value?.fit();
  });

  watch(() => settingsStore.settings.fontFamily, (family) => {
    if (!terminal.value) return;
    terminal.value.options.fontFamily = family;
    fitAddon.value?.fit();
  });

  onBeforeUnmount(() => {
    unsubscribe?.();
    unsubDisconnect?.();
    resizeObserver?.disconnect();
    terminal.value?.dispose();
  });

  function searchText(query: string) { searchAddon.value?.findNext(query); }
  function searchPrevious(query: string) { searchAddon.value?.findPrevious(query); }

  function copySelection() {
    const sel = terminal.value?.getSelection();
    if (sel) navigator.clipboard.writeText(sel);
  }

  function paste() {
    navigator.clipboard.readText().then((text) => {
      if (text) rpcClient.call("ssh.write", { sessionId: sessionId.value, data: text }).catch(() => undefined);
    });
  }

  function clear() { terminal.value?.clear(); }

  function zoomIn() {
    if (!terminal.value) return;
    const size = Math.min((terminal.value.options.fontSize || 14) + 1, 32);
    terminal.value.options.fontSize = size;
    settingsStore.saveSettings({ fontSize: size });
    fitAddon.value?.fit();
  }

  function zoomOut() {
    if (!terminal.value) return;
    const size = Math.max((terminal.value.options.fontSize || 14) - 1, 8);
    terminal.value.options.fontSize = size;
    settingsStore.saveSettings({ fontSize: size });
    fitAddon.value?.fit();
  }

  function zoomReset() {
    if (!terminal.value) return;
    terminal.value.options.fontSize = 14;
    settingsStore.saveSettings({ fontSize: 14 });
    fitAddon.value?.fit();
  }

  return { terminal, fitAddon, searchText, searchPrevious, copySelection, paste, clear, zoomIn, zoomOut, zoomReset };
}
