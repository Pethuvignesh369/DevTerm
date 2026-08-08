import { defineStore } from "pinia";
import { ref } from "vue";
import { rpcClient } from "@/lib/rpc-client";

export type ForwardType = "local" | "remote" | "dynamic";

export interface ForwardRule {
  id: string;
  hostId: string;
  type: ForwardType;
  localHost: string;
  localPort: number;
  remoteHost?: string;
  remotePort?: number;
  createdAt: string;
}

export interface StartForwardParams {
  sessionId: string;
  type: ForwardType;
  localHost?: string;
  localPort: number;
  remoteHost?: string;
  remotePort?: number;
}

export const useForwardsStore = defineStore("forwards", () => {
  const rules = ref<ForwardRule[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchRules() {
    loading.value = true;
    error.value = null;
    try {
      const result = await rpcClient.call<object, ForwardRule[]>("forward.list", {});
      rules.value = result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function startForward(params: StartForwardParams) {
    error.value = null;
    try {
      await rpcClient.call("forward.start", params);
      await fetchRules();
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    }
  }

  async function stopForward(id: string) {
    error.value = null;
    try {
      await rpcClient.call("forward.stop", { id });
      rules.value = rules.value.filter((r) => r.id !== id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    }
  }

  return { rules, loading, error, fetchRules, startForward, stopForward };
});
