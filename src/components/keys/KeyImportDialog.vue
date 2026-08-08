<script setup lang="ts">
import { ref, watch } from "vue";
import { useKeysStore } from "@/stores/keys";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ "update:open": [value: boolean] }>();

const keysStore = useKeysStore();
const name = ref("");
const privateKey = ref("");
const passphrase = ref("");
const submitting = ref(false);
const error = ref("");

watch(
  () => props.open,
  (val) => {
    if (val) {
      name.value = "";
      privateKey.value = "";
      passphrase.value = "";
      error.value = "";
    }
  }
);

async function submit() {
  if (!name.value.trim()) {
    error.value = "Name is required";
    return;
  }
  if (!privateKey.value.trim()) {
    error.value = "Private key content is required";
    return;
  }
  submitting.value = true;
  error.value = "";
  try {
    await keysStore.importKey({
      name: name.value.trim(),
      privateKey: privateKey.value,
      passphrase: passphrase.value || undefined,
    });
    emit("update:open", false);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    submitting.value = false;
  }
}

function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  const reader = new FileReader();
  reader.onload = () => {
    privateKey.value = reader.result as string;
  };
  reader.readAsText(file);
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="fixed inset-0 bg-black/50" @click="emit('update:open', false)" />
      <div class="relative z-10 w-full max-w-md rounded-lg border border-border bg-card p-6 shadow-lg">
        <h3 class="text-lg font-semibold">Import SSH Key</h3>
        <p class="mt-1 text-sm text-muted-foreground">Import an existing private key file.</p>

        <div v-if="error" class="mt-4 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {{ error }}
        </div>

        <form class="mt-4 space-y-4" @submit.prevent="submit">
          <div>
            <label class="mb-1 block text-sm font-medium">Name</label>
            <Input v-model="name" placeholder="e.g., my-server-key" />
          </div>

          <div>
            <label class="mb-1 block text-sm font-medium">Private Key File</label>
            <input
              type="file"
              class="w-full text-sm text-muted-foreground file:mr-4 file:rounded-md file:border-0 file:bg-primary file:px-4 file:py-2 file:text-sm file:font-medium file:text-primary-foreground hover:file:bg-primary/90"
              @change="handleFileSelect"
            />
          </div>

          <div>
            <label class="mb-1 block text-sm font-medium">Passphrase (if encrypted)</label>
            <Input v-model="passphrase" type="password" placeholder="Leave empty if key is not encrypted" />
          </div>

          <div v-if="privateKey" class="rounded-md bg-muted p-3">
            <p class="text-xs text-muted-foreground">Key loaded ({{ privateKey.length }} characters)</p>
          </div>

          <div class="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" @click="emit('update:open', false)">Cancel</Button>
            <Button type="submit" :disabled="submitting">
              {{ submitting ? "Importing..." : "Import" }}
            </Button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>
