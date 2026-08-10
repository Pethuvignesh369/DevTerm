import { onMounted, onBeforeUnmount } from "vue";
import { useRouter } from "vue-router";
import { useSessionsStore } from "@/stores/sessions";

/**
 * Global keyboard shortcuts for the application.
 * Ctrl+T: New connection (navigate to connections)
 * Ctrl+W: Close current tab
 * Ctrl+1-9: Switch to tab by index
 * Ctrl+Tab: Next tab
 * Ctrl+Shift+Tab: Previous tab
 * Ctrl+,: Open settings
 */
export function useKeyboardShortcuts() {
  const router = useRouter();
  const sessionsStore = useSessionsStore();

  function handleKeydown(event: KeyboardEvent) {
    const ctrl = event.ctrlKey || event.metaKey;

    if (!ctrl) return;

    // Ctrl+T: Go to connections to add new
    if (event.key === "t" && !event.shiftKey) {
      event.preventDefault();
      router.push("/");
      return;
    }

    // Ctrl+W: Close current tab
    if (event.key === "w") {
      event.preventDefault();
      if (sessionsStore.activeTabId) {
        sessionsStore.closeTab(sessionsStore.activeTabId);
      }
      return;
    }

    // Ctrl+,: Settings
    if (event.key === ",") {
      event.preventDefault();
      router.push("/settings");
      return;
    }

    // Ctrl+Tab / Ctrl+Shift+Tab: Cycle tabs
    if (event.key === "Tab") {
      event.preventDefault();
      const tabs = sessionsStore.tabs;
      if (tabs.length < 2) return;
      const currentIdx = tabs.findIndex((t) => t.id === sessionsStore.activeTabId);
      let nextIdx: number;
      if (event.shiftKey) {
        nextIdx = currentIdx <= 0 ? tabs.length - 1 : currentIdx - 1;
      } else {
        nextIdx = currentIdx >= tabs.length - 1 ? 0 : currentIdx + 1;
      }
      sessionsStore.setActiveTab(tabs[nextIdx].id);
      router.push("/terminal");
      return;
    }

    // Ctrl+1-9: Switch to tab by index
    const num = parseInt(event.key);
    if (num >= 1 && num <= 9) {
      event.preventDefault();
      const tabs = sessionsStore.tabs;
      const idx = num - 1;
      if (idx < tabs.length) {
        sessionsStore.setActiveTab(tabs[idx].id);
        router.push("/terminal");
      }
      return;
    }
  }

  onMounted(() => {
    window.addEventListener("keydown", handleKeydown);
  });

  onBeforeUnmount(() => {
    window.removeEventListener("keydown", handleKeydown);
  });
}
