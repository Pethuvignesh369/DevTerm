<script setup lang="ts">
import { RouterView } from "vue-router";
import { onMounted } from "vue";
import AppSidebar from "./components/AppSidebar.vue";
import StatusBar from "./components/StatusBar.vue";
import NotificationToast from "./components/NotificationToast.vue";
import LockScreen from "./components/LockScreen.vue";
import CommandPalette from "./components/CommandPalette.vue";
import { useSettingsStore } from "./stores/settings";
import { useKeyboardShortcuts } from "./composables/useKeyboardShortcuts";

const settingsStore = useSettingsStore();
useKeyboardShortcuts();

onMounted(() => {
  settingsStore.loadSettings();
  document.documentElement.classList.add("dark");
});
</script>

<template>
  <div class="flex h-screen w-screen flex-col overflow-hidden bg-background">
    <div class="flex flex-1 min-h-0">
      <AppSidebar />
      <main class="flex-1 overflow-hidden">
        <RouterView v-slot="{ Component }">
          <KeepAlive>
            <component :is="Component" />
          </KeepAlive>
        </RouterView>
      </main>
    </div>
    <StatusBar />
  </div>
  <NotificationToast />
  <LockScreen />
  <CommandPalette />
</template>
