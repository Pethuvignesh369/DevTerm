<script setup lang="ts">
import { ref, toRef } from "vue";
import { useTerminal } from "@/composables/useTerminal";

const props = defineProps<{
  sessionId: string;
}>();

const containerRef = ref<HTMLElement | null>(null);
const sessionIdRef = toRef(props, "sessionId");

const { searchText, searchPrevious } = useTerminal(containerRef, sessionIdRef);

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

// Ctrl+F to open search
function handleKeydown(event: KeyboardEvent) {
  if ((event.ctrlKey || event.metaKey) && event.key === "f") {
    event.preventDefault();
    toggleSearch();
  }
}
</script>

<template>
  <div class="relative flex h-full min-h-0 flex-col" @keydown="handleKeydown">
    <!-- Search bar -->
    <div v-if="showSearch" class="absolute right-4 top-2 z-10 flex items-center gap-2 rounded-md border border-border bg-card px-3 py-1.5 shadow-md">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search..."
        class="w-48 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
        autofocus
        @keydown="handleSearchKeydown"
      />
      <button class="text-xs text-muted-foreground hover:text-foreground" @click="searchText(searchQuery)">
        Next
      </button>
      <button class="text-xs text-muted-foreground hover:text-foreground" @click="searchPrevious(searchQuery)">
        Prev
      </button>
      <button class="text-muted-foreground hover:text-foreground" @click="toggleSearch">
        &times;
      </button>
    </div>

    <!-- Terminal container -->
    <div ref="containerRef" class="flex-1 min-h-0 overflow-hidden" style="height: 100%; width: 100%;" />
  </div>
</template>
