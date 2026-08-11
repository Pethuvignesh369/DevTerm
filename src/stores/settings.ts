import { defineStore } from "pinia";
import { ref, watch } from "vue";
import { rpcClient } from "@/lib/rpc-client";
import { debounce } from "@/lib/debounce";

export type ThemeMode = "light" | "dark" | "system";

export interface AppSettings {
  theme: ThemeMode;
  terminalTheme: string;
  fontFamily: string;
  fontSize: number;
  monitorPollInterval: number;
  connectionTimeout: number;
}

export const useSettingsStore = defineStore("settings", () => {
  const settings = ref<AppSettings>({
    theme: "dark",
    terminalTheme: "devterm",
    fontFamily: "JetBrains Mono, Fira Code, monospace",
    fontSize: 14,
    monitorPollInterval: 3000,
    connectionTimeout: 30000,
  });

  // Apply theme to document
  watch(
    () => settings.value.theme,
    (theme) => {
      const root = document.documentElement;
      if (theme === "dark") {
        root.classList.add("dark");
      } else if (theme === "light") {
        root.classList.remove("dark");
      } else {
        if (window.matchMedia("(prefers-color-scheme: dark)").matches) {
          root.classList.add("dark");
        } else {
          root.classList.remove("dark");
        }
      }
    },
    { immediate: true }
  );

  async function loadSettings() {
    try {
      const result = await rpcClient.call<object, Record<string, string>>("settings.getAll", {});
      if (result.theme) settings.value.theme = result.theme as ThemeMode;
      if (result.terminalTheme) settings.value.terminalTheme = result.terminalTheme;
      if (result.fontFamily) settings.value.fontFamily = result.fontFamily;
      if (result.fontSize) settings.value.fontSize = parseInt(result.fontSize);
      if (result.monitorPollInterval) settings.value.monitorPollInterval = parseInt(result.monitorPollInterval);
      if (result.connectionTimeout) settings.value.connectionTimeout = parseInt(result.connectionTimeout);
    } catch {
      // Use defaults if backend not ready
    }
  }

  let pendingSettings: Partial<AppSettings> = {};
  const persistSettings = debounce(() => {
    const changes = pendingSettings;
    pendingSettings = {};
    rpcClient.call("settings.set", changes).catch(() => {
      // Settings remain applied for this session when the sidecar is unavailable.
    });
  }, 250);

  function saveSettings(partial: Partial<AppSettings>) {
    Object.assign(settings.value, partial);
    Object.assign(pendingSettings, partial);
    persistSettings();
  }

  return { settings, loadSettings, saveSettings };
});
