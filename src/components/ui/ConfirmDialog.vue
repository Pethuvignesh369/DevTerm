<script setup lang="ts">
import { Button } from "@/components/ui/button";

defineProps<{
  open: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: "default" | "destructive";
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  confirm: [];
  cancel: [];
}>();

function handleConfirm() {
  emit("confirm");
  emit("update:open", false);
}

function handleCancel() {
  emit("cancel");
  emit("update:open", false);
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="fixed inset-0 bg-black/60 backdrop-blur-sm" @click="handleCancel" />
        <div class="relative z-10 w-full max-w-sm rounded-xl border border-border bg-card p-6 shadow-2xl animate-scale-in">
          <h3 class="text-lg font-semibold">{{ title }}</h3>
          <p class="mt-2 text-sm text-muted-foreground">{{ message }}</p>
          <div class="mt-5 flex justify-end gap-2">
            <Button variant="outline" size="sm" @click="handleCancel">
              {{ cancelText || "Cancel" }}
            </Button>
            <Button
              :variant="variant === 'destructive' ? 'destructive' : 'default'"
              size="sm"
              @click="handleConfirm"
            >
              {{ confirmText || "Confirm" }}
            </Button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active, .modal-leave-active {
  transition: all 0.2s ease;
}
.modal-enter-from, .modal-leave-to {
  opacity: 0;
}
</style>
