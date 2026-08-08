<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useKeysStore } from "@/stores/keys";
import { Button } from "@/components/ui/button";
import KeyGenerateDialog from "./KeyGenerateDialog.vue";
import KeyImportDialog from "./KeyImportDialog.vue";

const keysStore = useKeysStore();
const showGenerate = ref(false);
const showImport = ref(false);

onMounted(() => {
  keysStore.fetchKeys();
});

async function handleDelete(id: string, name: string) {
  if (confirm(`Delete key "${name}"? This cannot be undone.`)) {
    await keysStore.deleteKey(id);
  }
}
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between border-b border-border px-6 py-4">
      <div>
        <h2 class="text-2xl font-bold">SSH Keys</h2>
        <p class="text-sm text-muted-foreground">Generate, import, and manage SSH key pairs.</p>
      </div>
      <div class="flex gap-2">
        <Button variant="outline" @click="showImport = true">Import Key</Button>
        <Button @click="showGenerate = true">Generate Key</Button>
      </div>
    </div>

    <!-- Error -->
    <div v-if="keysStore.error" class="mx-6 mt-4 rounded-md bg-destructive/10 px-4 py-3 text-sm text-destructive">
      {{ keysStore.error }}
    </div>

    <!-- Loading -->
    <div v-if="keysStore.loading" class="flex flex-1 items-center justify-center">
      <p class="text-muted-foreground">Loading keys...</p>
    </div>

    <!-- Empty state -->
    <div v-else-if="keysStore.keys.length === 0" class="flex flex-1 flex-col items-center justify-center gap-4">
      <div class="text-center">
        <p class="text-lg font-medium">No SSH keys yet</p>
        <p class="text-sm text-muted-foreground">Generate a new key pair or import an existing private key.</p>
      </div>
      <div class="flex gap-2">
        <Button variant="outline" @click="showImport = true">Import Key</Button>
        <Button @click="showGenerate = true">Generate Key</Button>
      </div>
    </div>

    <!-- Key list -->
    <div v-else class="flex-1 overflow-auto p-6">
      <div class="space-y-3">
        <div
          v-for="key in keysStore.keys"
          :key="key.id"
          class="flex items-center justify-between rounded-lg border border-border p-4"
        >
          <div class="flex-1">
            <div class="flex items-center gap-2">
              <span class="font-medium">{{ key.name }}</span>
              <span class="rounded bg-secondary px-2 py-0.5 text-xs font-mono uppercase">
                {{ key.keyType }}
              </span>
              <span v-if="key.passphraseProtected" class="rounded bg-primary/10 px-2 py-0.5 text-xs text-primary">
                passphrase
              </span>
            </div>
            <p class="mt-1 font-mono text-xs text-muted-foreground">{{ key.fingerprint }}</p>
            <p class="mt-1 text-xs text-muted-foreground">Created {{ key.createdAt }}</p>
          </div>
          <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="handleDelete(key.id, key.name)">
            Delete
          </Button>
        </div>
      </div>
    </div>

    <!-- Dialogs -->
    <KeyGenerateDialog v-model:open="showGenerate" />
    <KeyImportDialog v-model:open="showImport" />
  </div>
</template>
