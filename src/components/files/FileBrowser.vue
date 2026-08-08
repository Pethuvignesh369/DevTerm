<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useSessionsStore } from "@/stores/sessions";
import { rpcClient } from "@/lib/rpc-client";
import { Button } from "@/components/ui/button";

interface FileEntry {
  name: string;
  size: number;
  mode: string;
  modTime: string;
  isDir: boolean;
}

interface TransferProgress {
  transferId: string;
  percent: number;
  bytesPerSec: number;
  status: "in_progress" | "complete" | "failed";
}

const sessionsStore = useSessionsStore();
const remotePath = ref("/");
const remoteEntries = ref<FileEntry[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const transfers = ref<TransferProgress[]>([]);

onMounted(() => {
  // Subscribe to transfer progress notifications
  rpcClient.subscribe("sftp.progress", (params: unknown) => {
    const p = params as TransferProgress;
    const existing = transfers.value.find((t) => t.transferId === p.transferId);
    if (existing) {
      Object.assign(existing, p);
    } else {
      transfers.value.push(p);
    }
  });

  rpcClient.subscribe("sftp.complete", (params: unknown) => {
    const p = params as { transferId: string };
    transfers.value = transfers.value.filter((t) => t.transferId !== p.transferId);
  });

  // Auto-load if there's an active session
  if (sessionsStore.activeSession?.status === "connected") {
    loadRemoteDir("/");
  }
});

async function loadRemoteDir(path: string) {
  const session = sessionsStore.activeSession;
  if (!session) {
    error.value = "No active session. Connect to a host first.";
    return;
  }

  loading.value = true;
  error.value = null;
  try {
    const result = await rpcClient.call<object, FileEntry[]>("sftp.list", {
      sessionId: session.id,
      path,
    });
    remoteEntries.value = result;
    remotePath.value = path;
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

function navigateUp() {
  const parts = remotePath.value.split("/").filter(Boolean);
  parts.pop();
  loadRemoteDir("/" + parts.join("/") || "/");
}

function navigateInto(entry: FileEntry) {
  if (!entry.isDir) return;
  const newPath = remotePath.value === "/" ? `/${entry.name}` : `${remotePath.value}/${entry.name}`;
  loadRemoteDir(newPath);
}

async function deleteEntry(entry: FileEntry) {
  const session = sessionsStore.activeSession;
  if (!session) return;
  if (!confirm(`Delete "${entry.name}"? This cannot be undone.`)) return;

  const fullPath = remotePath.value === "/" ? `/${entry.name}` : `${remotePath.value}/${entry.name}`;
  try {
    await rpcClient.call("sftp.delete", { sessionId: session.id, path: fullPath });
    await loadRemoteDir(remotePath.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

async function renameEntry(entry: FileEntry) {
  const session = sessionsStore.activeSession;
  if (!session) return;

  const newName = prompt(`Rename "${entry.name}" to:`, entry.name);
  if (!newName || newName === entry.name) return;

  const oldPath = remotePath.value === "/" ? `/${entry.name}` : `${remotePath.value}/${entry.name}`;
  const newPath = remotePath.value === "/" ? `/${newName}` : `${remotePath.value}/${newName}`;
  try {
    await rpcClient.call("sftp.rename", { sessionId: session.id, oldPath, newPath });
    await loadRemoteDir(remotePath.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
}
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between border-b border-border px-6 py-4">
      <div>
        <h2 class="text-2xl font-bold">File Browser</h2>
        <p class="text-sm text-muted-foreground">Transfer files via SFTP.</p>
      </div>
      <Button v-if="sessionsStore.activeSession" @click="loadRemoteDir(remotePath)">Refresh</Button>
    </div>

    <!-- No session -->
    <div v-if="!sessionsStore.activeSession" class="flex flex-1 items-center justify-center">
      <p class="text-muted-foreground">Connect to a host to browse remote files.</p>
    </div>

    <template v-else>
      <!-- Path bar -->
      <div class="flex items-center gap-2 border-b border-border px-6 py-2">
        <Button variant="ghost" size="sm" @click="navigateUp">⬆ Up</Button>
        <span class="font-mono text-sm text-muted-foreground">{{ remotePath }}</span>
      </div>

      <!-- Transfers in progress -->
      <div v-if="transfers.length > 0" class="border-b border-border px-6 py-2">
        <div v-for="t in transfers" :key="t.transferId" class="flex items-center gap-3 text-sm">
          <div class="h-2 flex-1 rounded bg-muted">
            <div class="h-full rounded bg-primary transition-all" :style="{ width: `${t.percent}%` }" />
          </div>
          <span class="text-xs text-muted-foreground">{{ t.percent }}%</span>
        </div>
      </div>

      <!-- Error -->
      <div v-if="error" class="mx-6 mt-2 rounded-md bg-destructive/10 px-4 py-2 text-sm text-destructive">
        {{ error }}
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-1 items-center justify-center">
        <p class="text-muted-foreground">Loading...</p>
      </div>

      <!-- File list -->
      <div v-else-if="remoteEntries.length > 0" class="flex-1 overflow-auto">
        <table class="w-full">
          <thead class="sticky top-0 bg-card text-left text-xs text-muted-foreground">
            <tr class="border-b border-border">
              <th class="px-6 py-2">Name</th>
              <th class="px-4 py-2">Size</th>
              <th class="px-4 py-2">Permissions</th>
              <th class="px-4 py-2">Modified</th>
              <th class="px-4 py-2">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="entry in remoteEntries"
              :key="entry.name"
              class="border-b border-border transition-colors hover:bg-accent/50"
              :class="entry.isDir ? 'cursor-pointer' : ''"
              @dblclick="navigateInto(entry)"
            >
              <td class="px-6 py-2 text-sm">
                <span v-if="entry.isDir" class="mr-1">📁</span>
                <span v-else class="mr-1">📄</span>
                {{ entry.name }}
              </td>
              <td class="px-4 py-2 text-sm text-muted-foreground">
                {{ entry.isDir ? "-" : formatSize(entry.size) }}
              </td>
              <td class="px-4 py-2 font-mono text-xs text-muted-foreground">{{ entry.mode }}</td>
              <td class="px-4 py-2 text-xs text-muted-foreground">{{ entry.modTime }}</td>
              <td class="px-4 py-2">
                <div class="flex gap-1">
                  <Button variant="ghost" size="sm" @click="renameEntry(entry)">Rename</Button>
                  <Button variant="ghost" size="sm" class="text-destructive" @click="deleteEntry(entry)">Delete</Button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-else class="flex flex-1 items-center justify-center">
        <div class="text-center">
          <p class="text-muted-foreground">Click Refresh to load the remote directory.</p>
        </div>
      </div>
    </template>
  </div>
</template>
