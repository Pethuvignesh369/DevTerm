import { defineStore } from "pinia";
import { ref, watch } from "vue";
import { rpcClient } from "@/lib/rpc-client";

export type ThemeMode = "light" | "dark" | "system";

export interface AppSettings {
  theme: ThemeMode;
  fontFamily: string;
  fontSize: number;
  monitorPollInterval: number;
}

export const useSettingsStore = defineStore("settings", () => {
  const settings = ref<AppSettings>({
    theme: "dark",
    fontFamily: "JetBrains Mono, Fira Code, monospace",
    fontSize: 14,
    monitorPollInterval: 3000,
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
        // system
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
      if (result.fontFamily) settings.value.fontFamily = result.fontFamily;
      if (result.fontSize) settings.value.fontSize = parseInt(result.fontSize);
      if (result.monitorPollInterval) settings.value.monitorPollInterval = parseInt(result.monitorPollInterval);
    } catch {
      // Use defaults if backend not ready
    }
  }

  async function saveSettings(partial: Partial<AppSettings>) {
    Object.assign(settings.value, partial);
    try {
      await rpcClient.call("settings.set", partial);
    } catch {
      // Persist locally even if backend fails
    }
  }

  return { settings, loadSettings, saveSettings };
});
