import { defineStore } from "pinia";
import { ref } from "vue";
import { rpcClient } from "@/lib/rpc-client";

export interface HistoryEntry {
  id: number;
  hostId: string;
  command: string;
  executedAt: string;
}

export const useHistoryStore = defineStore("history", () => {
  const entries = ref<HistoryEntry[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function search(query: string, hostId?: string) {
    loading.value = true;
    error.value = null;
    try {
      const result = await rpcClient.call<object, HistoryEntry[]>("history.search", {
        query,
        hostId,
        limit: 100,
      });
      entries.value = result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function record(hostId: string, command: string) {
    try {
      await rpcClient.call("history.record", { hostId, command });
    } catch {
      // Best effort
    }
  }

  return { entries, loading, error, search, record };
});
