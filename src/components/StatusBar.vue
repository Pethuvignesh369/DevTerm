<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from "vue";
import { useSessionsStore } from "@/stores/sessions";

const sessionsStore = useSessionsStore();
const elapsed = ref("");
let timer: ReturnType<typeof setInterval> | null = null;
const startTime = ref<number | null>(null);

const activeHost = computed(() => {
  const session = sessionsStore.activeSession;
  if (!session || session.status !== "connected") return null;
  return session.hostName;
});

const sessionCount = computed(() => {
  return Object.values(sessionsStore.sessions).filter((s) => s.status === "connected").length;
});

function updateElapsed() {
  if (!startTime.value) {
    elapsed.value = "";
    return;
  }
  const diff = Math.floor((Date.now() - startTime.value) / 1000);
  const h = Math.floor(diff / 3600);
  const m = Math.floor((diff % 3600) / 60);
  const s = diff % 60;
  if (h > 0) {
    elapsed.value = `${h}h ${m}m`;
  } else if (m > 0) {
    elapsed.value = `${m}m ${s}s`;
  } else {
    elapsed.value = `${s}s`;
  }
}

onMounted(() => {
  startTime.value = Date.now();
  timer = setInterval(updateElapsed, 1000);
  updateElapsed();
});

onBeforeUnmount(() => {
  if (timer) clearInterval(timer);
});
</script>

<template>
  <footer class="flex h-6 items-center justify-between border-t border-border/50 bg-card/50 px-4 text-[10px] text-muted-foreground">
    <div class="flex items-center gap-3">
      <div v-if="activeHost" class="flex items-center gap-1.5">
        <span class="status-dot status-dot-connected" style="height: 6px; width: 6px;" />
        <span>{{ activeHost }}</span>
      </div>
      <span v-if="sessionCount > 0">{{ sessionCount }} session{{ sessionCount > 1 ? 's' : '' }}</span>
    </div>
    <div class="flex items-center gap-3">
      <span v-if="elapsed">Session: {{ elapsed }}</span>
      <span class="font-mono">Ctrl+K: Commands</span>
    </div>
  </footer>
</template>
