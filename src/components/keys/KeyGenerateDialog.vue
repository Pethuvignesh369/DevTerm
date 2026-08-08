<script setup lang="ts">
import { ref, watch } from "vue";
import { useKeysStore } from "@/stores/keys";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ "update:open": [value: boolean] }>();

const keysStore = useKeysStore();
const name = ref("");
const keyType = ref<"ed25519" | "rsa">("ed25519");
const passphrase = ref("");
const bits = ref(4096);
const submitting = ref(false);
const error = ref("");

watch(
  () => props.open,
  (val) => {
    if (val) {
      name.value = "";
      keyType.value = "ed25519";
      passphrase.value = "";
      bits.value = 4096;
      error.value = "";
    }
  }
);

async function submit() {
  if (!name.value.trim()) {
    error.value = "Name is required";
    return;
  }
  submitting.value = true;
  error.value = "";
  try {
    await keysStore.generateKey({
      name: name.value.trim(),
      keyType: keyType.value,
      passphrase: passphrase.value || undefined,
      bits: keyType.value === "rsa" ? bits.value : undefined,
    });
    emit("update:open", false);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="fixed inset-0 bg-black/50" @click="emit('update:open', false)" />
      <div class="relative z-10 w-full max-w-md rounded-lg border border-border bg-card p-6 shadow-lg">
        <h3 class="text-lg font-semibold">Generate SSH Key</h3>
        <p class="mt-1 text-sm text-muted-foreground">Create a new SSH key pair.</p>

        <div v-if="error" class="mt-4 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {{ error }}
        </div>

        <form class="mt-4 space-y-4" @submit.prevent="submit">
          <div>
            <label class="mb-1 block text-sm font-medium">Name</label>
            <Input v-model="name" placeholder="e.g., production-key" />
          </div>

          <div>
            <label class="mb-1 block text-sm font-medium">Key Type</label>
            <div class="flex gap-4">
              <label class="flex items-center gap-2 text-sm">
                <input v-model="keyType" type="radio" value="ed25519" class="accent-primary" />
                ED25519 (recommended)
              </label>
              <label class="flex items-center gap-2 text-sm">
                <input v-model="keyType" type="radio" value="rsa" class="accent-primary" />
                RSA
              </label>
            </div>
          </div>

          <div v-if="keyType === 'rsa'">
            <label class="mb-1 block text-sm font-medium">Key Size (bits)</label>
            <select v-model="bits" class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm">
              <option :value="2048">2048</option>
              <option :value="4096">4096</option>
            </select>
          </div>

          <div>
            <label class="mb-1 block text-sm font-medium">Passphrase (optional)</label>
            <Input v-model="passphrase" type="password" placeholder="Leave empty for no passphrase" />
          </div>

          <div class="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" @click="emit('update:open', false)">Cancel</Button>
            <Button type="submit" :disabled="submitting">
              {{ submitting ? "Generating..." : "Generate" }}
            </Button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>
