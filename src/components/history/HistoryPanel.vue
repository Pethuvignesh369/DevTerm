<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useHistoryStore } from "@/stores/history";
import { Input } from "@/components/ui/input";

const historyStore = useHistoryStore();
const searchQuery = ref("");
let searchTimeout: ReturnType<typeof setTimeout> | null = null;

onMounted(() => {
  historyStore.search("");
});

function handleSearch() {
  if (searchTimeout) clearTimeout(searchTimeout);
  searchTimeout = setTimeout(() => {
    historyStore.search(searchQuery.value);
  }, 200);
}

const emit = defineEmits<{
  select: [command: string];
}>();

function selectEntry(command: string) {
  emit("select", command);
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="border-b border-border px-4 py-3">
      <Input
        v-model="searchQuery"
        placeholder="Search command history..."
        @input="handleSearch"
      />
    </div>

    <div v-if="historyStore.loading" class="flex flex-1 items-center justify-center">
      <p class="text-sm text-muted-foreground">Loading...</p>
    </div>

    <div v-else-if="historyStore.entries.length === 0" class="flex flex-1 items-center justify-center">
      <p class="text-sm text-muted-foreground">No history entries found.</p>
    </div>

    <div v-else class="flex-1 overflow-auto">
      <button
        v-for="entry in historyStore.entries"
        :key="entry.id"
        class="w-full border-b border-border px-4 py-2 text-left transition-colors hover:bg-accent"
        @click="selectEntry(entry.command)"
      >
        <p class="truncate font-mono text-sm">{{ entry.command }}</p>
        <p class="text-xs text-muted-foreground">{{ entry.executedAt }}</p>
      </button>
    </div>
  </div>
</template>
