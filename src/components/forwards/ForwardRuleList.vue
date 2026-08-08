<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useForwardsStore } from "@/stores/forwards";
import { useSessionsStore } from "@/stores/sessions";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { ForwardType } from "@/stores/forwards";

const forwardsStore = useForwardsStore();
const sessionsStore = useSessionsStore();
const showForm = ref(false);
const formType = ref<ForwardType>("local");
const formLocalHost = ref("127.0.0.1");
const formLocalPort = ref(8080);
const formRemoteHost = ref("127.0.0.1");
const formRemotePort = ref(80);
const formError = ref("");

onMounted(() => {
  forwardsStore.fetchRules();
});

async function handleCreate() {
  const session = sessionsStore.activeSession;
  if (!session) {
    formError.value = "No active session. Connect to a host first.";
    return;
  }
  formError.value = "";
  try {
    await forwardsStore.startForward({
      sessionId: session.id,
      type: formType.value,
      localHost: formLocalHost.value,
      localPort: formLocalPort.value,
      remoteHost: formType.value !== "dynamic" ? formRemoteHost.value : undefined,
      remotePort: formType.value !== "dynamic" ? formRemotePort.value : undefined,
    });
    showForm.value = false;
  } catch (e) {
    formError.value = e instanceof Error ? e.message : String(e);
  }
}

function formatRule(rule: typeof forwardsStore.rules[number]): string {
  if (rule.type === "dynamic") {
    return `SOCKS ${rule.localHost}:${rule.localPort}`;
  }
  if (rule.type === "local") {
    return `L ${rule.localHost}:${rule.localPort} → ${rule.remoteHost}:${rule.remotePort}`;
  }
  return `R ${rule.remoteHost}:${rule.remotePort} → ${rule.localHost}:${rule.localPort}`;
}
</script>

<template>
  <div class="flex h-full flex-col p-6">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-2xl font-bold">Port Forwarding</h2>
        <p class="mt-1 text-sm text-muted-foreground">Manage SSH tunnels.</p>
      </div>
      <Button @click="showForm = !showForm">
        {{ showForm ? "Cancel" : "New Forward" }}
      </Button>
    </div>

    <!-- Create form -->
    <div v-if="showForm" class="mt-4 rounded-lg border border-border p-4">
      <div v-if="formError" class="mb-3 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
        {{ formError }}
      </div>
      <div class="space-y-3">
        <div>
          <label class="mb-1 block text-sm font-medium">Type</label>
          <div class="flex gap-4">
            <label class="flex items-center gap-2 text-sm">
              <input v-model="formType" type="radio" value="local" class="accent-primary" />
              Local
            </label>
            <label class="flex items-center gap-2 text-sm">
              <input v-model="formType" type="radio" value="remote" class="accent-primary" />
              Remote
            </label>
            <label class="flex items-center gap-2 text-sm">
              <input v-model="formType" type="radio" value="dynamic" class="accent-primary" />
              Dynamic (SOCKS)
            </label>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="mb-1 block text-sm font-medium">Local Host</label>
            <Input v-model="formLocalHost" />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium">Local Port</label>
            <Input v-model.number="formLocalPort" type="number" />
          </div>
        </div>
        <div v-if="formType !== 'dynamic'" class="grid grid-cols-2 gap-3">
          <div>
            <label class="mb-1 block text-sm font-medium">Remote Host</label>
            <Input v-model="formRemoteHost" />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium">Remote Port</label>
            <Input v-model.number="formRemotePort" type="number" />
          </div>
        </div>
        <Button size="sm" @click="handleCreate">Start Forward</Button>
      </div>
    </div>

    <!-- Error -->
    <div v-if="forwardsStore.error" class="mt-4 rounded-md bg-destructive/10 px-4 py-3 text-sm text-destructive">
      {{ forwardsStore.error }}
    </div>

    <!-- Rules list -->
    <div v-if="forwardsStore.rules.length === 0" class="mt-8 text-center text-muted-foreground">
      No active forwarding rules.
    </div>
    <div v-else class="mt-4 space-y-2">
      <div
        v-for="rule in forwardsStore.rules"
        :key="rule.id"
        class="flex items-center justify-between rounded-lg border border-border p-3"
      >
        <div>
          <span class="rounded bg-secondary px-2 py-0.5 text-xs font-mono uppercase">{{ rule.type }}</span>
          <span class="ml-2 font-mono text-sm">{{ formatRule(rule) }}</span>
        </div>
        <Button variant="ghost" size="sm" class="text-destructive" @click="forwardsStore.stopForward(rule.id)">
          Stop
        </Button>
      </div>
    </div>
  </div>
</template>
