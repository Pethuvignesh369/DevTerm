<script setup lang="ts">
import { computed, onMounted, onActivated, ref } from "vue";
import { useRouter } from "vue-router";
import { useHostsStore } from "@/stores/hosts";
import { useSessionsStore } from "@/stores/sessions";
import { useRecentsStore } from "@/stores/recents";
import { rpcClient } from "@/lib/rpc-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import HostForm from "./HostForm.vue";
import HostEditDialog from "./HostEditDialog.vue";

const hostsStore = useHostsStore();
const sessionsStore = useSessionsStore();
const recentsStore = useRecentsStore();
const router = useRouter();
const showForm = ref(false);
const showFavorites = ref(false);
const showEdit = ref(false);
const editingHost = ref<typeof hostsStore.hosts[number] | null>(null);
const connectingHostId = ref<string | null>(null);

function openEdit(host: typeof hostsStore.hosts[number]) {
  editingHost.value = host;
  showEdit.value = true;
}

onMounted(() => {
  hostsStore.fetchHosts();
  hostsStore.fetchIdentities();
  hostsStore.fetchGroups();
});

onActivated(() => {
  // Refresh hosts when coming back to this view
  hostsStore.fetchHosts();
});

async function connect(hostId: string) {
  const host = hostsStore.hosts.find((h) => h.id === hostId);
  if (!host) return;
  connectingHostId.value = hostId;
  try {
    await sessionsStore.connect(hostId, host.name);
    router.push("/terminal");
  } finally {
    connectingHostId.value = null;
  }
}

async function handleDelete(id: string, name: string) {
  if (confirm(`Delete host "${name}"? This cannot be undone.`)) {
    await hostsStore.deleteHost(id);
  }
}

async function duplicateHost(hostId: string) {
  const host = hostsStore.hosts.find((h) => h.id === hostId);
  if (!host) return;
  try {
    await hostsStore.createHost({
      name: host.name + " (copy)",
      hostname: host.hostname,
      port: host.port,
      username: host.username,
      identityId: host.identityId || undefined,
      favorite: false,
      tags: host.tags,
    });
  } catch (e) {
    hostsStore.error = e instanceof Error ? e.message : String(e);
  }
}

const displayedHosts = computed(() => {
  if (showFavorites.value) return hostsStore.favoriteHosts;
  return hostsStore.filteredHosts;
});

// Quick connect (without saving host)
const quickHost = ref("");
async function quickConnect() {
  if (!quickHost.value.trim()) return;
  const input = quickHost.value.trim();
  let username = "root";
  let hostname = input;

  if (input.includes("@")) {
    const parts = input.split("@");
    username = parts[0];
    hostname = parts[1];
  }
  if (hostname.includes(":")) {
    const parts = hostname.split(":");
    hostname = parts[0];
  }

  // Connect directly without saving — use ssh.connect with inline params
  try {
    await sessionsStore.connect("__quick__" + Date.now(), `${username}@${hostname}`);
    router.push("/terminal");
    quickHost.value = "";
  } catch (e) {
    hostsStore.error = e instanceof Error ? e.message : String(e);
  }
}

async function importSSHConfig() {
  try {
    const result = await rpcClient.call<object, { imported: number; total: number }>("hosts.importSSHConfig", {});
    if (result.imported > 0) {
      await hostsStore.fetchHosts();
      alert(`Imported ${result.imported} of ${result.total} hosts from ~/.ssh/config`);
    } else {
      alert(`No new hosts to import (${result.total} entries found, all already exist)`);
    }
  } catch (e) {
    hostsStore.error = e instanceof Error ? e.message : String(e);
  }
}
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between px-6 py-5">
      <div>
        <h2 class="text-2xl font-bold tracking-tight">Connections</h2>
        <p class="mt-0.5 text-sm text-muted-foreground">Manage and connect to your SSH hosts.</p>
      </div>
      <div class="flex gap-2">
        <Button variant="outline" class="gap-2 text-xs" @click="importSSHConfig">
          <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
          </svg>
          Import SSH Config
        </Button>
        <Button class="gap-2" @click="showForm = true">
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
          </svg>
          Add Host
        </Button>
      </div>
    </div>

    <!-- Quick Connect -->
    <div class="flex items-center gap-2 px-6 pb-3">
      <div class="relative flex-1 max-w-md">
        <svg class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
        <input
          v-model="quickHost"
          class="w-full rounded-lg border border-input bg-background pl-9 pr-3 py-2 text-sm placeholder:text-muted-foreground focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-smooth"
          placeholder="Quick connect: user@hostname:port"
          @keydown.enter="quickConnect"
        />
      </div>
      <Button v-if="quickHost" size="sm" @click="quickConnect">Connect</Button>
    </div>

    <!-- Recent connections -->
    <div v-if="recentsStore.recents.length > 0" class="px-6 pb-3">
      <p class="text-[10px] font-medium text-muted-foreground uppercase tracking-wider mb-2">Recent</p>
      <div class="flex gap-2 overflow-x-auto scrollbar-thin pb-1">
        <button
          v-for="recent in recentsStore.recents.slice(0, 5)"
          :key="recent.hostId"
          class="shrink-0 flex items-center gap-2 rounded-lg border border-border/50 bg-card/50 px-3 py-1.5 text-xs transition-smooth hover:border-primary/30 hover:bg-accent press-effect"
          :disabled="!hostsStore.hosts.some((host) => host.id === recent.hostId)"
          :title="hostsStore.hosts.some((host) => host.id === recent.hostId) ? `Connect to ${recent.hostName}` : 'This host has been removed'"
          @click="connect(recent.hostId)"
        >
          <span class="h-1.5 w-1.5 rounded-full bg-muted-foreground/40" />
          <span class="font-medium">{{ recent.hostName }}</span>
        </button>
      </div>
    </div>

    <!-- Search and filters -->
    <div class="flex items-center gap-3 px-6 pb-4">
      <div class="relative flex-1 max-w-sm">
        <svg class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <Input
          v-model="hostsStore.searchQuery"
          placeholder="Search hosts..."
          class="pl-9"
        />
      </div>
      <button
        class="flex items-center gap-1.5 rounded-lg border px-3 py-2 text-sm transition-smooth"
        :class="showFavorites ? 'border-primary/50 bg-primary/10 text-primary' : 'border-border text-muted-foreground hover:bg-accent'"
        @click="showFavorites = !showFavorites"
      >
        <svg class="h-4 w-4" :fill="showFavorites ? 'currentColor' : 'none'" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
        </svg>
        Favorites
      </button>
    </div>

    <!-- Error -->
    <div v-if="hostsStore.error" class="mx-6 mb-4 animate-fade-in rounded-lg bg-destructive/10 px-4 py-3 text-sm text-destructive">
      {{ hostsStore.error }}
    </div>

    <!-- Loading -->
    <div v-if="hostsStore.loading" class="flex flex-1 items-center justify-center">
      <div class="flex items-center gap-3 text-muted-foreground">
        <svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        Loading hosts...
      </div>
    </div>

    <!-- Empty state -->
    <div v-else-if="displayedHosts.length === 0" class="flex flex-1 flex-col items-center justify-center gap-5">
      <div class="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
        <svg class="h-8 w-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14M12 5l7 7-7 7" />
        </svg>
      </div>
      <div class="text-center">
        <p class="text-lg font-medium">{{ showFavorites ? "No favorites" : "No hosts yet" }}</p>
        <p class="mt-1 text-sm text-muted-foreground">
          {{ showFavorites ? "Star a host to add it to favorites." : "Add your first SSH host to get started." }}
        </p>
      </div>
      <Button v-if="!showFavorites" @click="showForm = true">Add Your First Host</Button>
    </div>

    <!-- Host grid -->
    <div v-else class="flex-1 overflow-auto px-6 pb-6 scrollbar-thin">
      <div class="grid gap-3 sm:grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 stagger-fade-in">
        <div
          v-for="host in displayedHosts"
          :key="host.id"
          class="group rounded-xl border bg-card p-4 hover-lift transition-all duration-200"
          :class="connectingHostId === host.id ? 'border-primary/50 ring-2 ring-primary/20 animate-pulse' : 'border-border'"
          @dblclick="connect(host.id)"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <button
                  class="shrink-0 text-base transition-smooth hover:scale-125"
                  :class="host.favorite ? 'text-yellow-500' : 'text-muted-foreground/40 hover:text-yellow-500'"
                  @click="hostsStore.toggleFavorite(host.id)"
                >
                  <svg class="h-4 w-4" :fill="host.favorite ? 'currentColor' : 'none'" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                  </svg>
                </button>
                <h3 class="truncate font-semibold text-sm">{{ host.name }}</h3>
              </div>
              <p class="mt-1.5 truncate font-mono text-xs text-muted-foreground">
                {{ host.username }}@{{ host.hostname }}:{{ host.port }}
              </p>
              <div v-if="host.tags.length" class="mt-2 flex flex-wrap gap-1">
                <span
                  v-for="tag in host.tags"
                  :key="tag"
                  class="rounded-md bg-secondary px-1.5 py-0.5 text-[10px] font-medium text-secondary-foreground"
                >
                  {{ tag }}
                </span>
              </div>
            </div>
          </div>

          <!-- Actions -->
          <div class="mt-3 flex items-center gap-2 border-t border-border/50 pt-3">
            <Button size="sm" class="flex-1 gap-1.5 text-xs" :disabled="connectingHostId === host.id" @click="connect(host.id)">
              <svg v-if="connectingHostId === host.id" class="h-3.5 w-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              <svg v-else class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              {{ connectingHostId === host.id ? "Connecting..." : "Connect" }}
            </Button>
            <button
              class="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground opacity-0 transition-smooth hover:bg-accent hover:text-foreground group-hover:opacity-100"
              title="Edit"
              @click="openEdit(host)"
            >
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </button>
            <button
              class="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground opacity-0 transition-smooth hover:bg-accent hover:text-foreground group-hover:opacity-100"
              title="Duplicate"
              @click="duplicateHost(host.id)"
            >
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
              </svg>
            </button>
            <button
              class="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground opacity-0 transition-smooth hover:bg-destructive/10 hover:text-destructive group-hover:opacity-100"
              title="Delete"
              @click="handleDelete(host.id, host.name)"
            >
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Host Dialog -->
    <HostForm v-model:open="showForm" />
    <HostEditDialog v-model:open="showEdit" :host="editingHost" />
  </div>
</template>
