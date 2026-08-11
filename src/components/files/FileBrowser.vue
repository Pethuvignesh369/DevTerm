<script setup lang="ts">
import { ref, onMounted, onActivated, watch } from "vue";
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

// Re-load when reactivated (KeepAlive) or session changes
onActivated(() => {
  if (sessionsStore.activeSession?.status === "connected" && remoteEntries.value.length === 0) {
    loadRemoteDir("/");
  }
});

let lastSessionId = "";
watch(
  () => sessionsStore.activeSession?.id,
  (newId) => {
    if (newId && newId !== lastSessionId) {
      lastSessionId = newId;
      remotePath.value = "/";
      remoteEntries.value = [];
      if (sessionsStore.activeSession?.status === "connected") {
        loadRemoteDir("/");
      }
    }
  }
);

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

function navigateToBreadcrumb(index: number) {
  const parts = remotePath.value.split("/").filter(Boolean);
  const newPath = "/" + parts.slice(0, index + 1).join("/");
  loadRemoteDir(newPath);
}

function navigateInto(entry: FileEntry) {
  if (!entry.isDir) return;
  const newPath = remotePath.value === "/" ? `/${entry.name}` : `${remotePath.value}/${entry.name}`;
  loadRemoteDir(newPath);
}

// Sorting
const sortColumn = ref<"name" | "size" | "modTime">("name");
const sortAsc = ref(true);

function toggleSort(col: "name" | "size" | "modTime") {
  if (sortColumn.value === col) {
    sortAsc.value = !sortAsc.value;
  } else {
    sortColumn.value = col;
    sortAsc.value = true;
  }
}

function sortedEntries() {
  const entries = [...remoteEntries.value];
  // Dirs first, then sort by column
  entries.sort((a, b) => {
    if (a.isDir && !b.isDir) return -1;
    if (!a.isDir && b.isDir) return 1;

    let cmp = 0;
    switch (sortColumn.value) {
      case "name":
        cmp = a.name.localeCompare(b.name);
        break;
      case "size":
        cmp = a.size - b.size;
        break;
      case "modTime":
        cmp = a.modTime.localeCompare(b.modTime);
        break;
    }
    return sortAsc.value ? cmp : -cmp;
  });
  return entries;
}

// Double-click to download
function handleDoubleClick(entry: FileEntry) {
  if (entry.isDir) {
    navigateInto(entry);
  } else {
    const session = sessionsStore.activeSession;
    if (!session) return;
    const fullPath = remotePath.value === "/" ? `/${entry.name}` : `${remotePath.value}/${entry.name}`;
    downloadEntry(fullPath, entry.name);
  }
}

async function downloadEntry(remoteFilePath: string, filename: string) {
  const session = sessionsStore.activeSession;
  if (!session) return;
  const localPath = prompt(`Save ${filename} to this full local path:`);
  if (!localPath?.trim()) return;
  try {
    await rpcClient.call("sftp.download", { sessionId: session.id, remotePath: remoteFilePath, localPath: localPath.trim() });
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

async function createFolder() {
  const session = sessionsStore.activeSession;
  if (!session) return;
  const name = prompt("New folder name:");
  if (!name?.trim() || name.includes("/")) return;
  const path = remotePath.value === "/" ? `/${name.trim()}` : `${remotePath.value}/${name.trim()}`;
  try {
    await rpcClient.call("sftp.mkdir", { sessionId: session.id, path });
    await loadRemoteDir(remotePath.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

const breadcrumbs = () => {
  return remotePath.value.split("/").filter(Boolean);
};

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
      <div v-if="sessionsStore.activeSession" class="flex gap-2">
        <Button variant="outline" size="sm" @click="createFolder">New folder</Button>
        <Button size="sm" @click="loadRemoteDir(remotePath)">Refresh</Button>
      </div>
    </div>

    <!-- No session -->
    <div v-if="!sessionsStore.activeSession" class="flex flex-1 items-center justify-center">
      <p class="text-muted-foreground">Connect to a host to browse remote files.</p>
    </div>

    <template v-else>
      <!-- Path bar with breadcrumbs -->
      <div class="flex items-center gap-1 border-b border-border/50 px-6 py-2 overflow-x-auto scrollbar-thin">
        <button class="shrink-0 rounded p-1 text-muted-foreground transition-smooth hover:bg-accent hover:text-foreground" @click="loadRemoteDir('/')">
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
          </svg>
        </button>
        <span class="text-muted-foreground/50">/</span>
        <template v-for="(crumb, idx) in breadcrumbs()" :key="idx">
          <button
            class="shrink-0 rounded px-1.5 py-0.5 text-xs font-medium transition-smooth hover:bg-accent"
            :class="idx === breadcrumbs().length - 1 ? 'text-foreground' : 'text-muted-foreground'"
            @click="navigateToBreadcrumb(idx)"
          >
            {{ crumb }}
          </button>
          <span v-if="idx < breadcrumbs().length - 1" class="text-muted-foreground/50">/</span>
        </template>
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
      <div v-else-if="remoteEntries.length > 0" class="flex-1 overflow-auto scrollbar-thin">
        <table class="w-full">
          <thead class="sticky top-0 bg-card text-left text-xs text-muted-foreground">
            <tr class="border-b border-border/50">
              <th class="px-6 py-2 cursor-pointer hover:text-foreground transition-smooth" @click="toggleSort('name')">
                Name {{ sortColumn === 'name' ? (sortAsc ? '↑' : '↓') : '' }}
              </th>
              <th class="px-4 py-2 cursor-pointer hover:text-foreground transition-smooth" @click="toggleSort('size')">
                Size {{ sortColumn === 'size' ? (sortAsc ? '↑' : '↓') : '' }}
              </th>
              <th class="px-4 py-2">Permissions</th>
              <th class="px-4 py-2 cursor-pointer hover:text-foreground transition-smooth" @click="toggleSort('modTime')">
                Modified {{ sortColumn === 'modTime' ? (sortAsc ? '↑' : '↓') : '' }}
              </th>
              <th class="px-4 py-2">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="entry in sortedEntries()"
              :key="entry.name"
              class="group border-b border-border/30 transition-smooth hover:bg-accent/30"
              :class="entry.isDir ? 'cursor-pointer' : ''"
              @dblclick="handleDoubleClick(entry)"
            >
              <td class="px-6 py-2 text-sm">
                <div class="flex items-center gap-2">
                  <svg v-if="entry.isDir" class="h-4 w-4 text-blue-400" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
                  </svg>
                  <svg v-else class="h-4 w-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                  <span :class="entry.isDir ? 'font-medium' : ''">{{ entry.name }}</span>
                </div>
              </td>
              <td class="px-4 py-2 text-xs text-muted-foreground font-mono">
                {{ entry.isDir ? "-" : formatSize(entry.size) }}
              </td>
              <td class="px-4 py-2 font-mono text-xs text-muted-foreground">{{ entry.mode }}</td>
              <td class="px-4 py-2 text-xs text-muted-foreground">{{ entry.modTime.split('T')[0] }}</td>
              <td class="px-4 py-2">
                <div class="flex gap-1 opacity-0 group-hover:opacity-100 transition-smooth">
                  <button v-if="!entry.isDir" class="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground" title="Download" @click.stop="downloadEntry(remotePath === '/' ? `/${entry.name}` : `${remotePath}/${entry.name}`, entry.name)">
                    <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 3v12m0 0l-4-4m4 4l4-4m5 4v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4" /></svg>
                  </button>
                  <button class="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground" title="Rename" @click.stop="renameEntry(entry)">
                    <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                    </svg>
                  </button>
                  <button class="rounded p-1 text-muted-foreground hover:bg-destructive/20 hover:text-destructive" title="Delete" @click.stop="deleteEntry(entry)">
                    <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
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
