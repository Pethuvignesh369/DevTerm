<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from "vue";
import { useSessionsStore } from "@/stores/sessions";
import { useSettingsStore } from "@/stores/settings";
import { rpcClient } from "@/lib/rpc-client";
import { Button } from "@/components/ui/button";

interface MetricsTick {
  sessionId: string;
  metrics: {
    cpu?: number;
    memTotal?: number;
    memUsed?: number;
    diskTotal?: number;
    diskUsed?: number;
    diskPercent?: number;
    netRx?: number;
    netTx?: number;
    uptime?: string;
    unavailable?: Record<string, boolean>;
  };
}

const sessionsStore = useSessionsStore();
const settingsStore = useSettingsStore();
const monitoring = ref(false);
const metrics = ref<MetricsTick["metrics"]>({});
const cpuHistory = ref<number[]>([]);
const memHistory = ref<number[]>([]);
let unsubscribe: (() => void) | null = null;

onMounted(() => {
  unsubscribe = rpcClient.subscribe("monitor.tick", (params: unknown) => {
    const tick = params as MetricsTick;
    const session = sessionsStore.activeSession;
    if (!session || tick.sessionId !== session.id) return;
    metrics.value = tick.metrics;

    // Track history
    if (tick.metrics.cpu !== undefined) {
      cpuHistory.value.push(tick.metrics.cpu);
      if (cpuHistory.value.length > 60) cpuHistory.value.shift();
    }
    if (tick.metrics.memTotal && tick.metrics.memUsed) {
      const pct = Math.round((tick.metrics.memUsed / tick.metrics.memTotal) * 100);
      memHistory.value.push(pct);
      if (memHistory.value.length > 60) memHistory.value.shift();
    }
  });
});

onBeforeUnmount(() => {
  unsubscribe?.();
  stopMonitoring();
});

async function startMonitoring() {
  const session = sessionsStore.activeSession;
  if (!session) return;
  try {
    await rpcClient.call("monitor.start", {
      sessionId: session.id,
      interval: settingsStore.settings.monitorPollInterval,
    });
    monitoring.value = true;
  } catch (e) {
    console.error("Failed to start monitoring:", e);
  }
}

async function stopMonitoring() {
  const session = sessionsStore.activeSession;
  if (!session || !monitoring.value) return;
  try {
    await rpcClient.call("monitor.stop", { sessionId: session.id });
  } catch {
    // Ignore
  }
  monitoring.value = false;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
}
</script>

<template>
  <div class="flex h-full flex-col overflow-auto p-6">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-2xl font-bold">Dashboard</h2>
        <p class="mt-1 text-sm text-muted-foreground">Monitor remote host system metrics.</p>
      </div>
      <div v-if="sessionsStore.activeSession?.status === 'connected'">
        <Button v-if="!monitoring" @click="startMonitoring">Start Monitoring</Button>
        <Button v-else variant="outline" @click="stopMonitoring">Stop</Button>
      </div>
    </div>

    <!-- No session -->
    <div v-if="!sessionsStore.activeSession || sessionsStore.activeSession.status !== 'connected'" class="mt-12 text-center text-muted-foreground">
      Connect to a host to view system metrics.
    </div>

    <!-- Metrics cards -->
    <div v-else-if="monitoring" class="mt-6 grid grid-cols-2 gap-4 lg:grid-cols-3">
      <!-- CPU -->
      <div class="rounded-lg border border-border p-4">
        <h3 class="text-sm font-medium text-muted-foreground">CPU Usage</h3>
        <div v-if="metrics.unavailable?.cpu" class="mt-2 text-sm text-muted-foreground">Unavailable</div>
        <div v-else-if="metrics.cpu !== undefined" class="mt-1">
          <p class="text-3xl font-bold">{{ metrics.cpu }}%</p>
          <div class="mt-2 h-2 rounded bg-muted">
            <div class="h-full rounded bg-primary transition-all" :style="{ width: `${metrics.cpu}%` }" />
          </div>
        </div>
        <div v-else class="mt-2 text-sm text-muted-foreground">Collecting...</div>
      </div>

      <!-- Memory -->
      <div class="rounded-lg border border-border p-4">
        <h3 class="text-sm font-medium text-muted-foreground">Memory</h3>
        <div v-if="metrics.unavailable?.memory" class="mt-2 text-sm text-muted-foreground">Unavailable</div>
        <div v-else-if="metrics.memUsed !== undefined" class="mt-1">
          <p class="text-3xl font-bold">
            {{ metrics.memUsed }} <span class="text-sm font-normal text-muted-foreground">/ {{ metrics.memTotal }} MB</span>
          </p>
          <div class="mt-2 h-2 rounded bg-muted">
            <div
              class="h-full rounded bg-primary transition-all"
              :style="{ width: `${metrics.memTotal ? (metrics.memUsed / metrics.memTotal) * 100 : 0}%` }"
            />
          </div>
        </div>
        <div v-else class="mt-2 text-sm text-muted-foreground">Collecting...</div>
      </div>

      <!-- Disk -->
      <div class="rounded-lg border border-border p-4">
        <h3 class="text-sm font-medium text-muted-foreground">Disk (root)</h3>
        <div v-if="metrics.unavailable?.disk" class="mt-2 text-sm text-muted-foreground">Unavailable</div>
        <div v-else-if="metrics.diskUsed !== undefined" class="mt-1">
          <p class="text-3xl font-bold">
            {{ metrics.diskUsed }} <span class="text-sm font-normal text-muted-foreground">/ {{ metrics.diskTotal }} GB</span>
          </p>
          <div class="mt-2 h-2 rounded bg-muted">
            <div
              class="h-full rounded bg-primary transition-all"
              :style="{ width: `${metrics.diskPercent || 0}%` }"
            />
          </div>
        </div>
        <div v-else class="mt-2 text-sm text-muted-foreground">Collecting...</div>
      </div>

      <!-- Network -->
      <div class="rounded-lg border border-border p-4">
        <h3 class="text-sm font-medium text-muted-foreground">Network</h3>
        <div v-if="metrics.unavailable?.network" class="mt-2 text-sm text-muted-foreground">Unavailable</div>
        <div v-else-if="metrics.netRx !== undefined" class="mt-1">
          <p class="text-sm"><span class="font-medium">RX:</span> {{ formatBytes(metrics.netRx) }}</p>
          <p class="text-sm"><span class="font-medium">TX:</span> {{ formatBytes(metrics.netTx || 0) }}</p>
        </div>
        <div v-else class="mt-2 text-sm text-muted-foreground">Collecting...</div>
      </div>

      <!-- Uptime -->
      <div class="rounded-lg border border-border p-4">
        <h3 class="text-sm font-medium text-muted-foreground">Uptime</h3>
        <div v-if="metrics.unavailable?.uptime" class="mt-2 text-sm text-muted-foreground">Unavailable</div>
        <div v-else-if="metrics.uptime" class="mt-2 text-2xl font-bold">{{ metrics.uptime }}</div>
        <div v-else class="mt-2 text-sm text-muted-foreground">Collecting...</div>
      </div>

      <!-- CPU History -->
      <div class="rounded-lg border border-border p-4" v-if="cpuHistory.length > 1">
        <h3 class="text-sm font-medium text-muted-foreground">CPU History</h3>
        <div class="mt-3 flex h-16 items-end gap-px">
          <div
            v-for="(val, i) in cpuHistory"
            :key="i"
            class="flex-1 rounded-t bg-primary/60 transition-all"
            :style="{ height: `${val}%` }"
          />
        </div>
      </div>

      <!-- Memory History -->
      <div class="col-span-2 rounded-lg border border-border p-4" v-if="memHistory.length > 1">
        <h3 class="text-sm font-medium text-muted-foreground">Memory History</h3>
        <div class="mt-3 flex h-16 items-end gap-px">
          <div
            v-for="(val, i) in memHistory"
            :key="i"
            class="flex-1 rounded-t bg-blue-500/60 transition-all"
            :style="{ height: `${val}%` }"
          />
        </div>
      </div>
    </div>

    <div v-else class="mt-12 text-center text-muted-foreground">
      Click "Start Monitoring" to begin collecting metrics.
    </div>
  </div>
</template>
