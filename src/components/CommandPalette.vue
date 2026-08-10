<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from "vue";
import { useRouter } from "vue-router";
import { useHostsStore } from "@/stores/hosts";
import { useSessionsStore } from "@/stores/sessions";
import { useSnippetsStore } from "@/stores/snippets";
import { rpcClient } from "@/lib/rpc-client";

const router = useRouter();
const hostsStore = useHostsStore();
const sessionsStore = useSessionsStore();
const snippetsStore = useSnippetsStore();

const open = ref(false);
const query = ref("");
const selectedIndex = ref(0);

interface CommandItem {
  id: string;
  label: string;
  description?: string;
  icon: string;
  action: () => void;
}

const allCommands = computed<CommandItem[]>(() => {
  const commands: CommandItem[] = [
    { id: "nav-connections", label: "Go to Connections", icon: "M5 12h14", action: () => router.push("/") },
    { id: "nav-terminal", label: "Go to Terminal", icon: "M4 17l6-6-6-6M12 19h8", action: () => router.push("/terminal") },
    { id: "nav-keys", label: "Go to SSH Keys", icon: "M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z", action: () => router.push("/keys") },
    { id: "nav-snippets", label: "Go to Snippets", icon: "M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4", action: () => router.push("/snippets") },
    { id: "nav-files", label: "Go to Files", icon: "M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z", action: () => router.push("/files") },
    { id: "nav-dashboard", label: "Go to Dashboard", icon: "M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6", action: () => router.push("/dashboard") },
    { id: "nav-settings", label: "Go to Settings", icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0", action: () => router.push("/settings") },
    { id: "nav-history", label: "Go to History", icon: "M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z", action: () => router.push("/history") },
    { id: "nav-forwards", label: "Go to Port Forwarding", icon: "M8 7h12m0 0l-4-4m4 4l-4 4", action: () => router.push("/forwards") },
  ];

  // Add hosts as quick-connect options
  for (const host of hostsStore.hosts) {
    commands.push({
      id: `connect-${host.id}`,
      label: `Connect: ${host.name}`,
      description: `${host.username}@${host.hostname}:${host.port}`,
      icon: "M13 10V3L4 14h7v7l9-11h-7z",
      action: async () => {
        await sessionsStore.connect(host.id, host.name);
        router.push("/terminal");
      },
    });
  }

  // Add snippets
  for (const snippet of snippetsStore.snippets.slice(0, 10)) {
    commands.push({
      id: `snippet-${snippet.id}`,
      label: `Run: ${snippet.title}`,
      description: snippet.command.slice(0, 60),
      icon: "M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4",
      action: () => {
        const session = sessionsStore.activeSession;
        if (session?.status === "connected") {
          rpcClient.call("ssh.write", { sessionId: session.id, data: snippet.command + "\n" });
          router.push("/terminal");
        }
      },
    });
  }

  return commands;
});

const filteredCommands = computed(() => {
  if (!query.value) return allCommands.value.slice(0, 12);
  const q = query.value.toLowerCase();
  return allCommands.value
    .filter((c) => c.label.toLowerCase().includes(q) || c.description?.toLowerCase().includes(q))
    .slice(0, 12);
});

function toggle() {
  open.value = !open.value;
  if (open.value) {
    query.value = "";
    selectedIndex.value = 0;
  }
}

function handleKeydown(event: KeyboardEvent) {
  if ((event.ctrlKey || event.metaKey) && event.key === "k") {
    event.preventDefault();
    toggle();
  }
  if (!open.value) return;

  if (event.key === "Escape") {
    open.value = false;
  } else if (event.key === "ArrowDown") {
    event.preventDefault();
    selectedIndex.value = Math.min(selectedIndex.value + 1, filteredCommands.value.length - 1);
  } else if (event.key === "ArrowUp") {
    event.preventDefault();
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0);
  } else if (event.key === "Enter") {
    event.preventDefault();
    const cmd = filteredCommands.value[selectedIndex.value];
    if (cmd) {
      cmd.action();
      open.value = false;
    }
  }
}

onMounted(() => {
  window.addEventListener("keydown", handleKeydown);
});

onBeforeUnmount(() => {
  window.removeEventListener("keydown", handleKeydown);
});
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="fixed inset-0 z-[60] flex items-start justify-center pt-[15vh]">
        <div class="fixed inset-0 bg-black/50 backdrop-blur-sm" @click="open = false" />
        <div class="relative z-10 w-full max-w-lg rounded-xl border border-border bg-card shadow-2xl animate-scale-in overflow-hidden">
          <!-- Search input -->
          <div class="flex items-center gap-3 border-b border-border px-4 py-3">
            <svg class="h-4 w-4 text-muted-foreground shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              v-model="query"
              type="text"
              placeholder="Type a command or search hosts..."
              class="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
              autofocus
            />
            <kbd class="rounded bg-muted px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground">ESC</kbd>
          </div>

          <!-- Results -->
          <div class="max-h-[300px] overflow-y-auto py-2 scrollbar-thin">
            <button
              v-for="(cmd, idx) in filteredCommands"
              :key="cmd.id"
              class="flex w-full items-center gap-3 px-4 py-2 text-left text-sm transition-smooth"
              :class="selectedIndex === idx ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent/50'"
              @click="cmd.action(); open = false"
              @mouseenter="selectedIndex = idx"
            >
              <svg class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
                <path stroke-linecap="round" stroke-linejoin="round" :d="cmd.icon" />
              </svg>
              <div class="flex-1 min-w-0">
                <p class="truncate font-medium text-foreground">{{ cmd.label }}</p>
                <p v-if="cmd.description" class="truncate text-xs text-muted-foreground">{{ cmd.description }}</p>
              </div>
            </button>
            <p v-if="filteredCommands.length === 0" class="px-4 py-6 text-center text-sm text-muted-foreground">No results found</p>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active, .modal-leave-active { transition: all 0.15s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
</style>
