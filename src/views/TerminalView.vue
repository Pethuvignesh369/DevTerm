<script setup lang="ts">
import { computed } from "vue";
import { useSessionsStore } from "@/stores/sessions";
import TerminalTabs from "@/components/terminal/TerminalTabs.vue";
import TerminalPane from "@/components/terminal/TerminalPane.vue";
import SplitContainer from "@/components/terminal/SplitContainer.vue";

const sessionsStore = useSessionsStore();

const activeSplit = computed(() => {
  if (!sessionsStore.activeTabId) return null;
  return sessionsStore.splits[sessionsStore.activeTabId] ?? null;
});
</script>

<template>
  <TerminalTabs>
    <template v-if="sessionsStore.activeSession">
      <!-- Connected: show terminal (possibly split) -->
      <template v-if="sessionsStore.activeSession.status === 'connected'">
        <!-- Split view -->
        <SplitContainer v-if="activeSplit" :direction="activeSplit.direction">
          <template #first>
            <TerminalPane
              :session-id="sessionsStore.activeSession.id"
              :key="'split-1-' + sessionsStore.activeSession.id"
            />
          </template>
          <template #second>
            <TerminalPane
              :session-id="activeSplit.sessionId"
              :key="'split-2-' + activeSplit.sessionId"
            />
          </template>
        </SplitContainer>

        <!-- Single pane -->
        <TerminalPane
          v-else
          :session-id="sessionsStore.activeSession.id"
          :key="sessionsStore.activeSession.id"
        />
      </template>

      <!-- Connecting -->
      <div v-else-if="sessionsStore.activeSession.status === 'connecting'" class="flex h-full items-center justify-center bg-[#0d1117]">
        <div class="text-center animate-fade-in">
          <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10">
            <svg class="h-6 w-6 text-primary animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          </div>
          <p class="text-sm font-medium text-white">Establishing connection...</p>
          <p class="mt-1 text-xs text-zinc-400">{{ sessionsStore.activeSession.hostName }}</p>
        </div>
      </div>

      <!-- Error -->
      <div v-else-if="sessionsStore.activeSession.status === 'error'" class="flex h-full items-center justify-center bg-[#0d1117]">
        <div class="max-w-sm text-center animate-fade-in">
          <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-red-500/10">
            <svg class="h-6 w-6 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <p class="text-sm font-medium text-white">Connection Failed</p>
          <p class="mt-2 rounded-lg bg-red-500/10 px-4 py-2.5 font-mono text-xs text-red-300">
            {{ sessionsStore.activeSession.error }}
          </p>
          <p class="mt-3 text-xs text-zinc-500">Check host settings and network, then try again.</p>
        </div>
      </div>

      <!-- Disconnected -->
      <div v-else class="flex h-full items-center justify-center bg-[#0d1117]">
        <div class="text-center animate-fade-in">
          <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-zinc-500/10">
            <svg class="h-6 w-6 text-zinc-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
            </svg>
          </div>
          <p class="text-sm text-zinc-400">Session disconnected</p>
        </div>
      </div>
    </template>

    <template v-else>
      <div class="flex h-full flex-col items-center justify-center gap-5 bg-[#0d1117]">
        <div class="flex h-16 w-16 items-center justify-center rounded-2xl bg-zinc-800">
          <svg class="h-8 w-8 text-zinc-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 17l6-6-6-6M12 19h8" />
          </svg>
        </div>
        <div class="text-center">
          <p class="text-sm font-medium text-zinc-300">No active sessions</p>
          <p class="mt-1 text-xs text-zinc-500">
            Connect to a host from Connections to open a terminal.
          </p>
        </div>
      </div>
    </template>
  </TerminalTabs>
</template>
