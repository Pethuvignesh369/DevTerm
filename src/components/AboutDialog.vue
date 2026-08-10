<script setup lang="ts">
defineProps<{ open: boolean }>();
const emit = defineEmits<{ "update:open": [value: boolean] }>();
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="fixed inset-0 bg-black/60 backdrop-blur-sm" @click="emit('update:open', false)" />
      <div class="relative z-10 w-full max-w-xs rounded-xl border border-border bg-card p-6 shadow-2xl animate-scale-in text-center">
        <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary text-primary-foreground">
          <svg class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 17l6-6-6-6M12 19h8" />
          </svg>
        </div>
        <h3 class="text-lg font-bold">DevTerm</h3>
        <p class="mt-1 text-xs text-muted-foreground">Modern SSH Client for DevOps</p>
        <div class="mt-4 space-y-1 text-xs text-muted-foreground">
          <p>Version <span class="font-mono text-foreground">0.1.0</span></p>
          <p>Tauri v2 + Vue 3 + Go</p>
          <p class="mt-2">MIT License</p>
        </div>
        <button
          class="mt-5 w-full rounded-lg bg-secondary py-2 text-xs font-medium text-secondary-foreground transition-smooth hover:bg-secondary/80"
          @click="emit('update:open', false)"
        >
          Close
        </button>
      </div>
    </div>
  </Teleport>
</template>
