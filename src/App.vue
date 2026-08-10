<script setup lang="ts">
import { RouterView } from "vue-router";
import { onMounted } from "vue";
import AppSidebar from "./components/AppSidebar.vue";
import NotificationToast from "./components/NotificationToast.vue";
import LockScreen from "./components/LockScreen.vue";
import { useSettingsStore } from "./stores/settings";
import { useKeyboardShortcuts } from "./composables/useKeyboardShortcuts";

const settingsStore = useSettingsStore();
useKeyboardShortcuts();

onMounted(() => {
  settingsStore.loadSettings();
  // Apply dark mode by default
  document.documentElement.classList.add("dark");
});
</script>

<template>
  <div class="flex h-screen w-screen overflow-hidden bg-background">
    <AppSidebar />
    <main class="flex-1 overflow-hidden">
      <RouterView v-slot="{ Component }">
        <Transition name="page" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
  </div>
  <NotificationToast />
  <LockScreen />
</template>

<style>
.page-enter-active,
.page-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.page-enter-from {
  opacity: 0;
  transform: translateY(4px);
}
.page-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
