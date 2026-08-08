<script setup lang="ts">
import { useNotificationsStore } from "@/stores/notifications";

const notificationsStore = useNotificationsStore();
</script>

<template>
  <!-- Toast notifications -->
  <Teleport to="body">
    <div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
      <TransitionGroup name="toast">
        <div
          v-for="notification in notificationsStore.items"
          :key="notification.id"
          class="flex w-80 items-start gap-3 rounded-xl border px-4 py-3 shadow-lg glass animate-slide-in"
          :class="{
            'border-border/50': notification.type === 'info',
            'border-green-500/30': notification.type === 'success',
            'border-destructive/30': notification.type === 'error',
            'border-yellow-500/30': notification.type === 'warning',
          }"
        >
          <!-- Icon -->
          <div class="mt-0.5 shrink-0">
            <svg v-if="notification.type === 'success'" class="h-4 w-4 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
            </svg>
            <svg v-else-if="notification.type === 'error'" class="h-4 w-4 text-destructive" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <svg v-else-if="notification.type === 'warning'" class="h-4 w-4 text-yellow-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <svg v-else class="h-4 w-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>

          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium">{{ notification.title }}</p>
            <p v-if="notification.message" class="mt-0.5 text-xs text-muted-foreground line-clamp-2">
              {{ notification.message }}
            </p>
          </div>

          <button
            class="shrink-0 rounded p-0.5 text-muted-foreground transition-smooth hover:text-foreground"
            @click="notificationsStore.remove(notification.id)"
          >
            <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </TransitionGroup>
    </div>

    <!-- Backend unavailable banner -->
    <Transition name="banner">
      <div
        v-if="!notificationsStore.backendAvailable"
        class="fixed left-0 right-0 top-0 z-50 flex items-center justify-center gap-2 bg-destructive px-4 py-2 text-center text-sm text-destructive-foreground shadow-lg"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        Backend process unavailable. Please restart DevTerm.
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateX(20px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateX(20px) scale(0.95);
}
.banner-enter-active,
.banner-leave-active {
  transition: all 0.3s ease;
}
.banner-enter-from,
.banner-leave-to {
  opacity: 0;
  transform: translateY(-100%);
}
</style>
