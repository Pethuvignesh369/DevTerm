<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useHostsStore } from "@/stores/hosts";
import { useSessionsStore } from "@/stores/sessions";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import HostForm from "./HostForm.vue";

const hostsStore = useHostsStore();
const sessionsStore = useSessionsStore();
const router = useRouter();
const showForm = ref(false);
const showFavorites = ref(false);

onMounted(() => {
  hostsStore.fetchHosts();
  hostsStore.fetchIdentities();
  hostsStore.fetchGroups();
});

async function connect(hostId: string) {
  const host = hostsStore.hosts.find((h) => h.id === hostId);
  if (!host) return;
  await sessionsStore.connect(hostId, host.name);
  router.push("/terminal");
}

async function handleDelete(id: string, name: string) {
  if (confirm(`Delete host "${name}"? This cannot be undone.`)) {
    await hostsStore.deleteHost(id);
  }
}

const displayedHosts = () => {
  if (showFavorites.value) return hostsStore.favoriteHosts;
  return hostsStore.filteredHosts;
};
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between px-6 py-5">
      <div>
        <h2 class="text-2xl font-bold tracking-tight">Connections</h2>
        <p class="mt-0.5 text-sm text-muted-foreground">Manage and connect to your SSH hosts.</p>
      </div>
      <Button class="gap-2" @click="showForm = true">
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        Add Host
      </Button>
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
    <div v-else-if="displayedHosts().length === 0" class="flex flex-1 flex-col items-center justify-center gap-5">
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
      <div class="grid gap-3 sm:grid-cols-1 lg:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="host in displayedHosts()"
          :key="host.id"
          class="group animate-fade-in rounded-xl border border-border bg-card p-4 card-hover"
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
            <Button size="sm" class="flex-1 gap-1.5 text-xs" @click="connect(host.id)">
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              Connect
            </Button>
            <button
              class="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground opacity-0 transition-smooth hover:bg-destructive/10 hover:text-destructive group-hover:opacity-100"
              @click="handleDelete(host.id, host.name)"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Host Dialog -->
    <HostForm v-model:open="showForm" />
  </div>
</template>
