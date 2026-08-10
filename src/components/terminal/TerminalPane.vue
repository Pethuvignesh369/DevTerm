<script setup lang="ts">
import { ref, toRef } from "vue";
import { useTerminal } from "@/composables/useTerminal";

const props = defineProps<{
  sessionId: string;
}>();

const containerRef = ref<HTMLElement | null>(null);
const sessionIdRef = toRef(props, "sessionId");

const { searchText, searchPrevious, copySelection, paste, clear } = useTerminal(containerRef, sessionIdRef);

const showSearch = ref(false);
const searchQuery = ref("");

function toggleSearch() {
  showSearch.value = !showSearch.value;
  if (!showSearch.value) {
    searchQuery.value = "";
  }
}

function handleSearchKeydown(event: KeyboardEvent) {
  if (event.key === "Enter") {
    if (event.shiftKey) {
      searchPrevious(searchQuery.value);
    } else {
      searchText(searchQuery.value);
    }
  }
  if (event.key === "Escape") {
    toggleSearch();
  }
}

function handleKeydown(event: KeyboardEvent) {
  // Ctrl+F to open search
  if ((event.ctrlKey || event.metaKey) && event.key === "f") {
    event.preventDefault();
    toggleSearch();
  }
  // Ctrl+Shift+C to copy
  if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === "C") {
    event.preventDefault();
    copySelection();
  }
  // Ctrl+Shift+V to paste
  if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === "V") {
    event.preventDefault();
    paste();
  }
  // Ctrl+L to clear
  if ((event.ctrlKey || event.metaKey) && event.key === "l") {
    event.preventDefault();
    clear();
  }
}
</script>

<template>
  <div class="relative flex h-full min-h-0 flex-col" @keydown="handleKeydown">
    <!-- Search bar -->
    <div
      v-if="showSearch"
      class="absolute right-4 top-2 z-10 flex items-center gap-2 rounded-lg border border-border/50 bg-card/90 backdrop-blur-md px-3 py-1.5 shadow-lg animate-scale-in"
    >
      <svg class="h-3.5 w-3.5 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Find in terminal..."
        class="w-48 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
        autofocus
        @keydown="handleSearchKeydown"
      />
      <div class="flex gap-0.5">
        <button class="rounded p-1 text-muted-foreground transition-smooth hover:bg-accent hover:text-foreground" title="Previous (Shift+Enter)" @click="searchPrevious(searchQuery)">
          <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7" />
          </svg>
        </button>
        <button class="rounded p-1 text-muted-foreground transition-smooth hover:bg-accent hover:text-foreground" title="Next (Enter)" @click="searchText(searchQuery)">
          <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
          </svg>
        </button>
      </div>
      <button class="rounded p-1 text-muted-foreground transition-smooth hover:bg-accent hover:text-foreground" @click="toggleSearch">
        <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>

    <!-- Terminal container -->
    <div ref="containerRef" class="flex-1 min-h-0 overflow-hidden" style="height: 100%; width: 100%;" />
  </div>
</template>
