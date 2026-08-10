import { defineStore } from "pinia";
import { ref } from "vue";

export interface RecentConnection {
  hostId: string;
  hostName: string;
  hostname: string;
  username: string;
  port: number;
  connectedAt: string;
}

const STORAGE_KEY = "devterm_recent_connections";
const MAX_RECENTS = 8;

export const useRecentsStore = defineStore("recents", () => {
  const recents = ref<RecentConnection[]>(loadFromStorage());

  function loadFromStorage(): RecentConnection[] {
    try {
      const data = localStorage.getItem(STORAGE_KEY);
      return data ? JSON.parse(data) : [];
    } catch {
      return [];
    }
  }

  function save() {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(recents.value));
  }

  function addRecent(connection: Omit<RecentConnection, "connectedAt">) {
    // Remove existing entry for same host
    recents.value = recents.value.filter((r) => r.hostId !== connection.hostId);
    // Add to front
    recents.value.unshift({
      ...connection,
      connectedAt: new Date().toISOString(),
    });
    // Trim to max
    if (recents.value.length > MAX_RECENTS) {
      recents.value = recents.value.slice(0, MAX_RECENTS);
    }
    save();
  }

  function clearRecents() {
    recents.value = [];
    save();
  }

  return { recents, addRecent, clearRecents };
});
