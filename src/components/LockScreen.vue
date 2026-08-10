<script setup lang="ts">
import { useAppLockStore } from "@/stores/applock";

const lockStore = useAppLockStore();

function handleUnlock() {
  // For now, just unlock on click/keypress
  // In future: could require master password or biometric
  lockStore.unlock();
}
</script>

<template>
  <div
    v-if="lockStore.locked"
    class="fixed inset-0 z-[100] flex items-center justify-center bg-background/95 backdrop-blur-md"
    @click="handleUnlock"
    @keydown="handleUnlock"
    tabindex="0"
  >
    <div class="text-center animate-fade-in">
      <div class="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-2xl bg-primary/10">
        <svg class="h-10 w-10 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
        </svg>
      </div>
      <h2 class="text-xl font-bold">DevTerm Locked</h2>
      <p class="mt-2 text-sm text-muted-foreground">Click or press any key to unlock</p>
      <p class="mt-4 text-xs text-muted-foreground/60">Sessions are still active in the background</p>
    </div>
  </div>
</template>
