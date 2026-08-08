import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { rpcClient } from "@/lib/rpc-client";

export interface Host {
  id: string;
  name: string;
  hostname: string;
  port: number;
  username: string;
  identityId: string | null;
  groupId: number | null;
  favorite: boolean;
  tags: string[];
  createdAt: string;
  updatedAt: string;
}

export interface Identity {
  id: string;
  name: string;
  authType: "password" | "key" | "agent";
  sshKeyId: string | null;
  vaultRef: string | null;
  createdAt: string;
}

export interface Group {
  id: number;
  name: string;
  createdAt: string;
}

export interface CreateHostParams {
  name: string;
  hostname: string;
  port: number;
  username: string;
  identityId?: string;
  groupId?: number;
  favorite?: boolean;
}

export const useHostsStore = defineStore("hosts", () => {
  const hosts = ref<Host[]>([]);
  const identities = ref<Identity[]>([]);
  const groups = ref<Group[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const searchQuery = ref("");

  const filteredHosts = computed(() => {
    if (!searchQuery.value) return hosts.value;
    const q = searchQuery.value.toLowerCase();
    return hosts.value.filter(
      (h) =>
        h.name.toLowerCase().includes(q) ||
        h.hostname.toLowerCase().includes(q) ||
        h.tags.some((t) => t.toLowerCase().includes(q))
    );
  });

  const favoriteHosts = computed(() => hosts.value.filter((h) => h.favorite));

  async function fetchHosts() {
    loading.value = true;
    error.value = null;
    try {
      const result = await rpcClient.call<object, Host[]>("hosts.list", {});
      hosts.value = result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function fetchIdentities() {
    try {
      const result = await rpcClient.call<object, Identity[]>("identities.list", {});
      identities.value = result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    }
  }

  async function fetchGroups() {
    try {
      const result = await rpcClient.call<object, Group[]>("groups.list", {});
      groups.value = result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    }
  }

  async function createHost(params: CreateHostParams) {
    error.value = null;
    try {
      await rpcClient.call("hosts.create", params);
      await fetchHosts();
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    }
  }

  async function updateHost(id: string, params: Partial<CreateHostParams>) {
    error.value = null;
    try {
      await rpcClient.call("hosts.update", { id, ...params });
      await fetchHosts();
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    }
  }

  async function deleteHost(id: string) {
    error.value = null;
    try {
      await rpcClient.call("hosts.delete", { id });
      hosts.value = hosts.value.filter((h) => h.id !== id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    }
  }

  async function toggleFavorite(id: string) {
    const host = hosts.value.find((h) => h.id === id);
    if (!host) return;
    await updateHost(id, { favorite: !host.favorite });
  }

  return {
    hosts,
    identities,
    groups,
    loading,
    error,
    searchQuery,
    filteredHosts,
    favoriteHosts,
    fetchHosts,
    fetchIdentities,
    fetchGroups,
    createHost,
    updateHost,
    deleteHost,
    toggleFavorite,
  };
});
