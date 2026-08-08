<script setup lang="ts">
import { useSessionsStore } from "@/stores/sessions";
import { rpcClient } from "@/lib/rpc-client";
import SnippetPanel from "@/components/snippets/SnippetPanel.vue";

const sessionsStore = useSessionsStore();

function runSnippet(command: string) {
  const session = sessionsStore.activeSession;
  if (!session || session.status !== "connected") {
    alert("No active terminal session. Connect to a host first.");
    return;
  }
  // Send the command + newline to execute it
  rpcClient.call("ssh.write", {
    sessionId: session.id,
    data: command + "\n",
  });
}
</script>

<template>
  <SnippetPanel @run="runSnippet" />
</template>
