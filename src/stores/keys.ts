import { defineStore } from "pinia";
import { ref } from "vue";
import { rpcClient } from "@/lib/rpc-client";

export interface SSHKey {
  id: string;
  name: string;
  keyType: "rsa" | "ed25519";
  publicKey: string;
  fingerprint: string;
  passphraseProtected: boolean;
  createdAt: string;
}

export interface GenerateKeyParams {
  name: string;
  keyType: "rsa" | "ed25519";
  passphrase?: string;
  bits?: number;
}

export interface ImportKeyParams {
  name: string;
  privateKey: string;
  passphrase?: string;
}

export const useKeysStore = defineStore("keys", () => {
  const keys = ref<SSHKey[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchKeys() {
    loading.value = true;
    error.value = null;
    try {
      const result = await rpcClient.call<object, SSHKey[]>("keys.list", {});
      keys.value = result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function generateKey(params: GenerateKeyParams) {
    loading.value = true;
    error.value = null;
    try {
      await rpcClient.call("keys.generate", params);
      await fetchKeys();
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function importKey(params: ImportKeyParams) {
    loading.value = true;
    error.value = null;
    try {
      await rpcClient.call("keys.import", params);
      await fetchKeys();
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function deleteKey(id: string) {
    loading.value = true;
    error.value = null;
    try {
      await rpcClient.call("keys.delete", { id });
      keys.value = keys.value.filter((k) => k.id !== id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { keys, loading, error, fetchKeys, generateKey, importKey, deleteKey };
});
