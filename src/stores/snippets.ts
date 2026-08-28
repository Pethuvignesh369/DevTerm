import { defineStore } from "pinia";
import { ref } from "vue";
import { rpcClient } from "@/lib/rpc-client";

export interface Snippet {
  id: string;
  title: string;
  command: string;
  tags: string[];
  createdAt: string;
  updatedAt: string;
}

export const useSnippetsStore = defineStore("snippets", () => {
  const snippets = ref<Snippet[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchSnippets(query?: string) {
    loading.value = true;
    error.value = null;
    try {
      const result = await rpcClient.call<object, Snippet[]>("snippets.list", { query });
      snippets.value = result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function createSnippet(title: string, command: string, tags: string[] = []) {
    error.value = null;
    try {
      await rpcClient.call("snippets.create", { title, command, tags });
      await fetchSnippets();
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    }
  }

  async function updateSnippet(id: string, title: string, command: string, tags?: string[]) {
    error.value = null;
    try {
      await rpcClient.call("snippets.update", { id, title, command, ...(tags ? { tags } : {}) });
      await fetchSnippets();
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    }
  }

  async function deleteSnippet(id: string) {
    error.value = null;
    try {
      await rpcClient.call("snippets.delete", { id });
      snippets.value = snippets.value.filter((s) => s.id !== id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    }
  }

  return { snippets, loading, error, fetchSnippets, createSnippet, updateSnippet, deleteSnippet };
});
