<script setup lang="ts">
import { useSessionsStore } from "@/stores/sessions";
import { rpcClient } from "@/lib/rpc-client";
import HistoryPanel from "@/components/history/HistoryPanel.vue";

const sessionsStore = useSessionsStore();

function insertCommand(command: string) {
  const session = sessionsStore.activeSession;
  if (!session || session.status !== "connected") {
    alert("No active terminal session. Connect to a host first.");
    return;
  }
  // Insert the command (without newline so user can edit before pressing enter)
  rpcClient.call("ssh.write", {
    sessionId: session.id,
    data: command,
  });
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="border-b border-border px-6 py-4">
      <h2 class="text-2xl font-bold">Command History</h2>
      <p class="text-sm text-muted-foreground">Click a command to insert it into the active terminal.</p>
    </div>
    <div class="flex-1 overflow-hidden">
      <HistoryPanel @select="insertCommand" />
    </div>
  </div>
</template>
