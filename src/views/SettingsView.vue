<script setup lang="ts">
import { useSettingsStore } from "@/stores/settings";
import { Input } from "@/components/ui/input";

const settingsStore = useSettingsStore();

function setTheme(theme: "light" | "dark" | "system") {
  settingsStore.saveSettings({ theme });
}

function setFontSize(event: Event) {
  const value = parseInt((event.target as HTMLInputElement).value);
  if (value >= 8 && value <= 32) {
    settingsStore.saveSettings({ fontSize: value });
  }
}

function setFontFamily(event: Event) {
  const value = (event.target as HTMLInputElement).value;
  settingsStore.saveSettings({ fontFamily: value });
}
</script>

<template>
  <div class="flex h-full flex-col overflow-auto p-6 scrollbar-thin">
    <h2 class="text-2xl font-bold tracking-tight">Settings</h2>
    <p class="mt-0.5 text-sm text-muted-foreground">Configure appearance and behavior.</p>

    <div class="mt-8 max-w-lg space-y-8">
      <!-- Theme -->
      <section class="animate-fade-in rounded-xl border border-border bg-card p-5">
        <h3 class="font-semibold">Appearance</h3>
        <p class="mt-0.5 text-xs text-muted-foreground">Choose your preferred color scheme.</p>
        <div class="mt-4 flex gap-2">
          <button
            v-for="theme in [
              { key: 'light' as const, label: 'Light', icon: 'M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z' },
              { key: 'dark' as const, label: 'Dark', icon: 'M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z' },
              { key: 'system' as const, label: 'System', icon: 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z' }
            ]"
            :key="theme.key"
            class="flex flex-1 flex-col items-center gap-2 rounded-lg border-2 p-3 transition-smooth"
            :class="settingsStore.settings.theme === theme.key ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/30'"
            @click="setTheme(theme.key)"
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" :d="theme.icon" />
            </svg>
            <span class="text-xs font-medium">{{ theme.label }}</span>
          </button>
        </div>
      </section>

      <!-- Font -->
      <section class="animate-fade-in rounded-xl border border-border bg-card p-5">
        <h3 class="font-semibold">Terminal Font</h3>
        <p class="mt-0.5 text-xs text-muted-foreground">Configure the font used in terminal sessions.</p>
        <div class="mt-4 space-y-4">
          <div>
            <label class="mb-1.5 block text-xs font-medium text-muted-foreground">Font Family</label>
            <Input
              :model-value="settingsStore.settings.fontFamily"
              @change="setFontFamily"
              placeholder="JetBrains Mono, monospace"
            />
          </div>
          <div>
            <div class="flex items-center justify-between mb-1.5">
              <label class="text-xs font-medium text-muted-foreground">Font Size</label>
              <span class="text-xs font-mono text-primary">{{ settingsStore.settings.fontSize }}px</span>
            </div>
            <input
              type="range"
              min="8"
              max="32"
              :value="settingsStore.settings.fontSize"
              class="w-full accent-primary h-1.5 rounded-full appearance-none bg-secondary cursor-pointer"
              @input="setFontSize"
            />
            <div class="flex justify-between mt-1">
              <span class="text-[10px] text-muted-foreground">8px</span>
              <span class="text-[10px] text-muted-foreground">32px</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Monitor -->
      <section class="animate-fade-in rounded-xl border border-border bg-card p-5">
        <h3 class="font-semibold">Monitoring</h3>
        <p class="mt-0.5 text-xs text-muted-foreground">Configure dashboard refresh intervals.</p>
        <div class="mt-4">
          <label class="mb-1.5 block text-xs font-medium text-muted-foreground">Poll Interval</label>
          <select
            :value="settingsStore.settings.monitorPollInterval"
            class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm transition-smooth focus:border-primary focus:ring-1 focus:ring-primary/50 outline-none"
            @change="settingsStore.saveSettings({ monitorPollInterval: parseInt(($event.target as HTMLSelectElement).value) })"
          >
            <option :value="1000">1 second (high load)</option>
            <option :value="3000">3 seconds (default)</option>
            <option :value="5000">5 seconds</option>
            <option :value="10000">10 seconds</option>
            <option :value="30000">30 seconds (low load)</option>
          </select>
        </div>
      </section>
    </div>
  </div>
</template>
