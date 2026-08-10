<script setup lang="ts">
import { ref, watch } from "vue";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const props = defineProps<{
  open: boolean;
  hostName: string;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  submit: [passphrase: string];
  cancel: [];
}>();

const passphrase = ref("");

watch(
  () => props.open,
  (val) => {
    if (val) passphrase.value = "";
  }
);

function handleSubmit() {
  emit("submit", passphrase.value);
  emit("update:open", false);
}

function handleCancel() {
  emit("cancel");
  emit("update:open", false);
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="fixed inset-0 bg-black/60 backdrop-blur-sm" @click="handleCancel" />
      <div class="relative z-10 w-full max-w-sm rounded-xl border border-border bg-card p-6 shadow-2xl animate-scale-in">
        <div class="flex items-center gap-3 mb-4">
          <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <svg class="h-5 w-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
          </div>
          <div>
            <h3 class="font-semibold">Passphrase Required</h3>
            <p class="text-xs text-muted-foreground">{{ hostName }}</p>
          </div>
        </div>

        <form @submit.prevent="handleSubmit">
          <Input
            v-model="passphrase"
            type="password"
            placeholder="Enter key passphrase..."
            class="mb-4"
            autofocus
          />
          <div class="flex justify-end gap-2">
            <Button type="button" variant="outline" size="sm" @click="handleCancel">Cancel</Button>
            <Button type="submit" size="sm">Unlock & Connect</Button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>
