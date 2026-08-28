<script setup lang="ts">
import { useSessionsStore } from "@/stores/sessions";
import { useNotificationsStore } from "@/stores/notifications";
import { rpcClient } from "@/lib/rpc-client";
import SnippetPanel from "@/components/snippets/SnippetPanel.vue";

const sessionsStore = useSessionsStore();
const notificationsStore = useNotificationsStore();

function runSnippet(command: string) {
  const session = sessionsStore.activeSession;
  if (!session || session.status !== "connected") {
    notificationsStore.add({
      type: "warning",
      title: "No active terminal",
      message: "Connect to a host before running a snippet.",
    });
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
