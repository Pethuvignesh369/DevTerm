<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useSnippetsStore } from "@/stores/snippets";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const snippetsStore = useSnippetsStore();
const searchQuery = ref("");
const showCreate = ref(false);
const newTitle = ref("");
const newCommand = ref("");
const createError = ref("");
let searchTimeout: ReturnType<typeof setTimeout> | null = null;

onMounted(() => {
  snippetsStore.fetchSnippets();
});

function handleSearch() {
  if (searchTimeout) clearTimeout(searchTimeout);
  searchTimeout = setTimeout(() => {
    snippetsStore.fetchSnippets(searchQuery.value || undefined);
  }, 200);
}

async function handleCreate() {
  if (!newTitle.value.trim() || !newCommand.value.trim()) {
    createError.value = "Title and command are required";
    return;
  }
  createError.value = "";
  try {
    await snippetsStore.createSnippet(newTitle.value.trim(), newCommand.value.trim());
    showCreate.value = false;
    newTitle.value = "";
    newCommand.value = "";
  } catch (e) {
    createError.value = e instanceof Error ? e.message : String(e);
  }
}

async function handleDelete(id: string, title: string) {
  if (confirm(`Delete snippet "${title}"?`)) {
    await snippetsStore.deleteSnippet(id);
  }
}

const emit = defineEmits<{
  run: [command: string];
}>();
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between border-b border-border px-6 py-4">
      <div>
        <h2 class="text-2xl font-bold">Snippets</h2>
        <p class="text-sm text-muted-foreground">Save and manage frequently used commands.</p>
      </div>
      <Button @click="showCreate = !showCreate">
        {{ showCreate ? "Cancel" : "New Snippet" }}
      </Button>
    </div>

    <!-- Create form -->
    <div v-if="showCreate" class="border-b border-border px-6 py-4">
      <div v-if="createError" class="mb-3 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
        {{ createError }}
      </div>
      <div class="space-y-3">
        <Input v-model="newTitle" placeholder="Snippet title" />
        <textarea
          v-model="newCommand"
          placeholder="Command..."
          class="w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          rows="3"
        />
        <Button size="sm" @click="handleCreate">Save Snippet</Button>
      </div>
    </div>

    <!-- Search -->
    <div class="border-b border-border px-6 py-3">
      <Input v-model="searchQuery" placeholder="Filter snippets..." @input="handleSearch" />
    </div>

    <!-- Loading -->
    <div v-if="snippetsStore.loading" class="flex flex-1 items-center justify-center">
      <p class="text-sm text-muted-foreground">Loading...</p>
    </div>

    <!-- Empty -->
    <div v-else-if="snippetsStore.snippets.length === 0" class="flex flex-1 items-center justify-center">
      <p class="text-sm text-muted-foreground">No snippets found.</p>
    </div>

    <!-- List -->
    <div v-else class="flex-1 overflow-auto p-6">
      <div class="space-y-3">
        <div
          v-for="snippet in snippetsStore.snippets"
          :key="snippet.id"
          class="rounded-lg border border-border p-4"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1">
              <h4 class="font-medium">{{ snippet.title }}</h4>
              <pre class="mt-2 overflow-x-auto rounded bg-muted p-2 font-mono text-sm">{{ snippet.command }}</pre>
              <div v-if="snippet.tags.length" class="mt-2 flex gap-1">
                <span v-for="tag in snippet.tags" :key="tag" class="rounded bg-secondary px-2 py-0.5 text-xs">
                  {{ tag }}
                </span>
              </div>
            </div>
            <div class="ml-4 flex flex-col gap-1">
              <Button size="sm" variant="outline" @click="emit('run', snippet.command)">Run</Button>
              <Button size="sm" variant="ghost" class="text-destructive" @click="handleDelete(snippet.id, snippet.title)">
                Delete
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
