<script setup lang="ts">
import { useSessionsStore } from "@/stores/sessions";

const sessionsStore = useSessionsStore();
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- Tab bar -->
    <div v-if="sessionsStore.tabs.length > 0" class="flex items-center bg-card/50 backdrop-blur-sm border-b border-border/50">
      <div class="flex flex-1 items-center overflow-x-auto scrollbar-thin">
        <div
          v-for="tab in sessionsStore.tabs"
          :key="tab.id"
          class="group relative flex items-center gap-2 px-4 py-2.5 text-sm cursor-pointer transition-smooth select-none"
          :class="
            sessionsStore.activeTabId === tab.id
              ? 'text-foreground'
              : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'
          "
          @click="sessionsStore.setActiveTab(tab.id)"
        >
          <!-- Active tab indicator -->
          <div
            v-if="sessionsStore.activeTabId === tab.id"
            class="absolute bottom-0 left-2 right-2 h-0.5 rounded-full bg-primary"
          />

          <!-- Status dot -->
          <span
            class="status-dot shrink-0"
            :class="{
              'status-dot-connected': sessionsStore.sessions[tab.sessionId]?.status === 'connected',
              'status-dot-connecting': sessionsStore.sessions[tab.sessionId]?.status === 'connecting',
              'status-dot-error': sessionsStore.sessions[tab.sessionId]?.status === 'error',
              'status-dot-disconnected': sessionsStore.sessions[tab.sessionId]?.status === 'disconnected',
            }"
          />

          <span class="truncate max-w-[120px] text-xs font-medium">{{ tab.title }}</span>

          <!-- Close button -->
          <button
            class="ml-1 flex h-4 w-4 shrink-0 items-center justify-center rounded opacity-0 transition-smooth hover:bg-destructive/20 hover:text-destructive group-hover:opacity-100"
            @click.stop="sessionsStore.closeTab(tab.id)"
          >
            <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Terminal content -->
    <div class="flex-1 min-h-0 overflow-hidden bg-[#0d1117]">
      <slot />
    </div>
  </div>
</template>
