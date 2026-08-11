<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useSessionsStore } from "@/stores/sessions";
import { rpcClient } from "@/lib/rpc-client";
import { Button } from "@/components/ui/button";
import ConfirmDialog from "@/components/ui/ConfirmDialog.vue";

interface FileEntry { name: string; path?: string; size: number; modTime: string; mode?: string; isDir: boolean }
interface LocalResult { path: string; entries: FileEntry[] }
interface TransferProgress {
  transferId: string; fileName?: string; direction?: "upload" | "download";
  percent: number; bytesTransferred?: number; totalBytes?: number; bytesPerSec?: number;
  etaSeconds?: number; status: "in_progress" | "complete" | "failed" | "cancelled";
}
interface ContextMenu { side: "local" | "remote"; entry: FileEntry; x: number; y: number }

const sessionsStore = useSessionsStore();
const router = useRouter();
const localPath = ref("");
const remotePath = ref("/");
const localEntries = ref<FileEntry[]>([]);
const remoteEntries = ref<FileEntry[]>([]);
const localLoading = ref(false);
const remoteLoading = ref(false);
const error = ref<string | null>(null);
const transfers = ref<TransferProgress[]>([]);
const dragOver = ref<"local" | "remote" | null>(null);
const remotePane = ref<HTMLElement | null>(null);
const pointerTransfer = ref<{ entry: FileEntry; startX: number; startY: number } | null>(null);
const contextMenu = ref<ContextMenu | null>(null);
const copiedLocalFile = ref<FileEntry | null>(null);
const pendingDelete = ref<FileEntry | null>(null);
let unsubs: (() => void)[] = [];

// Prefer the focused terminal, but keep Files usable when another terminal tab
// is connected (for example after navigating here from Connections).
const session = computed(() => {
  if (sessionsStore.activeSession?.status === "connected") return sessionsStore.activeSession;
  return Object.values(sessionsStore.sessions).find((item) => item.status === "connected") ?? null;
});
const remoteLabel = computed(() => session.value ? `${session.value.hostName} · remote` : "Remote host");

function joinRemote(directory: string, name: string) { return directory === "/" ? `/${name}` : `${directory}/${name}`; }
function parent(path: string, separator: string) {
  const trimmed = path.replace(/[\\/]+$/, "");
  const index = trimmed.lastIndexOf(separator);
  return index <= 0 ? (separator === "/" ? "/" : trimmed.slice(0, 3)) : trimmed.slice(0, index);
}
function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
  return `${(bytes / 1024 ** 3).toFixed(1)} GB`;
}
function formatSpeed(bytesPerSec?: number) { return `${formatSize(bytesPerSec ?? 0)}/s`; }
function formatEta(seconds?: number) {
  if (!seconds || seconds < 1) return "finishing";
  const minutes = Math.floor(seconds / 60);
  return minutes ? `${minutes}m ${seconds % 60}s left` : `${seconds}s left`;
}

async function loadLocal(path = localPath.value) {
  localLoading.value = true; error.value = null;
  try {
    const result = await rpcClient.call<{ path?: string }, LocalResult>("local.list", path ? { path } : {});
    localPath.value = result.path; localEntries.value = result.entries;
  } catch (e) { error.value = e instanceof Error ? e.message : String(e); }
  finally { localLoading.value = false; }
}
async function loadRemote(path = remotePath.value) {
  if (!session.value) { remoteEntries.value = []; return; }
  remoteLoading.value = true; error.value = null;
  try {
    remoteEntries.value = await rpcClient.call("sftp.list", { sessionId: session.value.id, path });
    remotePath.value = path;
  } catch (e) { error.value = e instanceof Error ? e.message : String(e); }
  finally { remoteLoading.value = false; }
}
function openLocal(entry: FileEntry) { if (entry.isDir && entry.path) loadLocal(entry.path); }
function openRemote(entry: FileEntry) { if (entry.isDir) loadRemote(joinRemote(remotePath.value, entry.name)); }
function beginPointerTransfer(event: PointerEvent, entry: FileEntry) {
  if (entry.isDir || !session.value) return;
  pointerTransfer.value = { entry, startX: event.clientX, startY: event.clientY };
}
function isOverRemotePane(event: PointerEvent) {
  const rect = remotePane.value?.getBoundingClientRect();
  return !!rect && event.clientX >= rect.left && event.clientX <= rect.right && event.clientY >= rect.top && event.clientY <= rect.bottom;
}
function handlePointerMove(event: PointerEvent) {
  const transfer = pointerTransfer.value;
  if (!transfer) return;
  const moved = Math.abs(event.clientX - transfer.startX) > 6 || Math.abs(event.clientY - transfer.startY) > 6;
  if (moved) dragOver.value = isOverRemotePane(event) ? "remote" : null;
}
function handlePointerUp(event: PointerEvent) {
  const transfer = pointerTransfer.value;
  pointerTransfer.value = null;
  const shouldUpload = !!transfer && dragOver.value === "remote" && isOverRemotePane(event);
  dragOver.value = null;
  if (shouldUpload && transfer) void upload(transfer.entry);
}
async function download(entry: FileEntry) {
  if (!session.value || entry.isDir) return;
  try { await rpcClient.call("sftp.download", { sessionId: session.value.id, remotePath: joinRemote(remotePath.value, entry.name), localPath: localPath.value }); }
  catch (e) { error.value = e instanceof Error ? e.message : String(e); }
}
async function upload(entry: FileEntry) {
  if (!session.value || entry.isDir || !entry.path) return;
  try { await rpcClient.call("sftp.upload", { sessionId: session.value.id, localPath: entry.path, remotePath: joinRemote(remotePath.value, entry.name) }); }
  catch (e) { error.value = e instanceof Error ? e.message : String(e); }
}
async function cancelTransfer(transferId: string) {
  try { await rpcClient.call("sftp.cancel", { transferId }); }
  catch (e) { error.value = e instanceof Error ? e.message : String(e); }
}
function openContextMenu(event: MouseEvent, side: "local" | "remote", entry: FileEntry) { contextMenu.value = { side, entry, x: event.clientX, y: event.clientY }; }
async function copyText(value: string) { try { await navigator.clipboard.writeText(value); } catch { error.value = "Could not copy to the clipboard."; } finally { contextMenu.value = null; } }
function copyLocalFile() { if (contextMenu.value?.side === "local") copiedLocalFile.value = contextMenu.value.entry; contextMenu.value = null; }
async function pasteLocalFile() { if (copiedLocalFile.value) await upload(copiedLocalFile.value); }
async function compressRemote(entry: FileEntry) {
  if (!session.value) return;
  contextMenu.value = null;
  try { const source = joinRemote(remotePath.value, entry.name); await rpcClient.call("sftp.compress", { sessionId: session.value.id, path: source, archivePath: `${source}.zip` }); await loadRemote(); }
  catch (e) { error.value = e instanceof Error ? e.message : String(e); }
}
async function deleteRemote(entry: FileEntry) {
  if (!session.value) return;
  contextMenu.value = null;
  try {
    await rpcClient.call("sftp.delete", { sessionId: session.value.id, path: joinRemote(remotePath.value, entry.name) });
    await loadRemote();
  } catch (e) { error.value = e instanceof Error ? e.message : String(e); }
}
function askDelete(entry: FileEntry) { pendingDelete.value = entry; contextMenu.value = null; }

onMounted(() => {
  loadLocal();
  if (session.value) loadRemote();
  unsubs = [
    rpcClient.subscribe("sftp.progress", (value) => {
      const progress = value as TransferProgress;
      const existing = transfers.value.find((item) => item.transferId === progress.transferId);
      if (existing) Object.assign(existing, progress); else transfers.value.push(progress);
    }),
    rpcClient.subscribe("sftp.complete", (value) => {
      const { transferId } = value as { transferId: string };
      loadLocal(); loadRemote();
      setTimeout(() => { transfers.value = transfers.value.filter((item) => item.transferId !== transferId); }, 1200);
    }),
  ];
  window.addEventListener("pointermove", handlePointerMove);
  window.addEventListener("pointerup", handlePointerUp);
  window.addEventListener("click", () => { contextMenu.value = null; });
});
onActivated(() => { loadLocal(); if (session.value) loadRemote(); });
onBeforeUnmount(() => {
  unsubs.forEach((unsubscribe) => unsubscribe());
  window.removeEventListener("pointermove", handlePointerMove);
  window.removeEventListener("pointerup", handlePointerUp);
});
watch(() => session.value?.id, () => { remotePath.value = "/"; loadRemote("/"); });
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <header class="flex flex-wrap items-center justify-between gap-3 border-b border-border px-5 py-3">
      <div>
        <h2 class="text-lg font-semibold">Files</h2>
        <p class="text-xs text-muted-foreground">Drag files between panes to transfer them.</p>
      </div>
      <div class="flex gap-2">
        <Button size="sm" variant="outline" @click="loadLocal()">Refresh local</Button>
        <Button size="sm" :disabled="!session" @click="loadRemote()">Refresh remote</Button>
      </div>
    </header>

    <div v-if="error" class="mx-5 mt-3 rounded-lg border border-destructive/20 bg-destructive/10 px-3 py-2 text-sm text-destructive">{{ error }}</div>
    <div v-if="transfers.length" class="mx-5 mt-3 space-y-3 rounded-lg border border-primary/15 bg-primary/5 p-3">
      <div v-for="transfer in transfers" :key="transfer.transferId" class="space-y-1.5 text-xs">
        <div class="flex items-center justify-between gap-3">
          <span class="min-w-0 truncate font-medium">{{ transfer.direction === 'upload' ? 'Uploading' : 'Downloading' }} {{ transfer.fileName || 'file' }}</span>
          <div class="flex shrink-0 items-center gap-2">
            <button v-if="transfer.status === 'in_progress'" class="rounded border border-border px-2 py-0.5 text-[10px] font-medium text-muted-foreground hover:bg-accent hover:text-foreground" @click="cancelTransfer(transfer.transferId)">Cancel</button>
            <span class="font-medium" :class="transfer.status === 'failed' ? 'text-destructive' : transfer.status === 'cancelled' ? 'text-muted-foreground' : 'text-primary'">{{ transfer.status === 'complete' ? 'Complete' : transfer.status === 'failed' ? 'Failed' : transfer.status === 'cancelled' ? 'Cancelled' : `${transfer.percent}%` }}</span>
          </div>
        </div>
        <div class="h-1.5 overflow-hidden rounded-full bg-muted"><div class="h-full rounded-full bg-primary transition-all duration-150" :class="transfer.status === 'failed' ? 'bg-destructive' : ''" :style="{ width: `${transfer.percent}%` }" /></div>
        <div class="flex flex-wrap items-center gap-x-3 text-muted-foreground">
          <span>{{ formatSize(transfer.bytesTransferred ?? 0) }} / {{ formatSize(transfer.totalBytes ?? 0) }}</span>
          <span>{{ formatSpeed(transfer.bytesPerSec) }}</span>
          <span v-if="transfer.status === 'in_progress'">{{ formatEta(transfer.etaSeconds) }}</span>
        </div>
      </div>
    </div>

    <div class="grid min-h-0 flex-1 grid-cols-1 divide-y divide-border lg:grid-cols-2 lg:divide-x lg:divide-y-0">
      <section class="flex min-h-0 flex-col">
        <div class="flex items-center gap-2 border-b border-border/60 px-4 py-2.5">
          <span class="flex h-6 w-6 items-center justify-center rounded-md bg-blue-500/10 text-blue-500">⌂</span>
          <span class="text-xs font-semibold uppercase tracking-wide">Local</span>
          <input v-model="localPath" class="ml-1 min-w-0 flex-1 bg-transparent font-mono text-xs text-muted-foreground outline-none" @keydown.enter="loadLocal()" />
          <button class="rounded p-1 text-muted-foreground hover:bg-accent" title="Parent directory" @click="loadLocal(parent(localPath, '\\'))">↑</button>
        </div>
        <div class="min-h-0 flex-1 overflow-auto scrollbar-thin">
          <div v-if="localLoading" class="p-4 text-sm text-muted-foreground">Loading local files…</div>
          <div v-for="entry in localEntries" v-else :key="entry.path" role="button" tabindex="0" class="group flex w-full cursor-grab items-center gap-2 border-b border-border/40 px-4 py-2 text-left text-sm hover:bg-accent/60 active:cursor-grabbing" @dblclick="openLocal(entry)" @keydown.enter="openLocal(entry)" @pointerdown="beginPointerTransfer($event, entry)" @contextmenu.prevent="openContextMenu($event, 'local', entry)">
            <span class="text-base">{{ entry.isDir ? '📁' : '📄' }}</span><span class="min-w-0 flex-1 truncate" :class="entry.isDir ? 'font-medium' : ''">{{ entry.name }}</span><span class="text-xs text-muted-foreground">{{ entry.isDir ? '' : formatSize(entry.size) }}</span>
            <button v-if="!entry.isDir && session" class="opacity-0 text-xs text-primary group-hover:opacity-100 focus-visible:opacity-100" @pointerdown.stop @click.stop="upload(entry)">Upload →</button>
          </div>
        </div>
      </section>

      <section ref="remotePane" class="relative flex min-h-0 flex-col" :class="dragOver === 'remote' ? 'bg-primary/5 ring-inset ring-2 ring-primary/50' : ''">
        <div class="flex items-center gap-2 border-b border-border/60 px-4 py-2.5">
          <span class="flex h-6 w-6 items-center justify-center rounded-md bg-primary/10 text-primary">›_</span>
          <span class="text-xs font-semibold uppercase tracking-wide">{{ remoteLabel }}</span>
          <input v-model="remotePath" :disabled="!session" class="ml-1 min-w-0 flex-1 bg-transparent font-mono text-xs text-muted-foreground outline-none disabled:cursor-not-allowed" @keydown.enter="loadRemote()" />
          <button class="rounded p-1 text-muted-foreground hover:bg-accent disabled:opacity-40" title="Parent directory" :disabled="!session" @click="loadRemote(parent(remotePath, '/'))">↑</button>
          <button v-if="copiedLocalFile" class="rounded bg-primary/10 px-2 py-1 text-xs font-medium text-primary hover:bg-primary/20" @click="pasteLocalFile">Paste</button>
        </div>
        <div v-if="!session" class="flex flex-1 flex-col items-center justify-center gap-3 px-8 text-center">
          <div class="flex h-11 w-11 items-center justify-center rounded-xl bg-primary/10 font-mono text-primary">›_</div>
          <div>
            <p class="text-sm font-medium">No remote connection</p>
            <p class="mt-1 text-xs text-muted-foreground">Connect to a host to browse its files and transfer by drag and drop.</p>
          </div>
          <Button size="sm" @click="router.push('/')">Connect to a host</Button>
        </div>
        <div v-else class="min-h-0 flex-1 overflow-auto scrollbar-thin">
          <div v-if="remoteLoading" class="p-4 text-sm text-muted-foreground">Loading remote files…</div>
          <div v-for="entry in remoteEntries" v-else :key="entry.name" role="button" tabindex="0" class="group flex w-full items-center gap-2 border-b border-border/40 px-4 py-2 text-left text-sm hover:bg-accent/60" @dblclick="openRemote(entry)" @keydown.enter="openRemote(entry)" @contextmenu.prevent="openContextMenu($event, 'remote', entry)">
            <span class="text-base">{{ entry.isDir ? '📁' : '📄' }}</span><span class="min-w-0 flex-1 truncate" :class="entry.isDir ? 'font-medium' : ''">{{ entry.name }}</span><span class="text-xs text-muted-foreground">{{ entry.isDir ? '' : formatSize(entry.size) }}</span>
            <button v-if="!entry.isDir" class="opacity-0 text-xs text-primary group-hover:opacity-100 focus-visible:opacity-100" @click.stop="download(entry)">← Download</button>
          </div>
          <div v-if="!remoteLoading && !remoteEntries.length" class="p-4 text-sm text-muted-foreground">This folder is empty.</div>
        </div>
        <div v-if="dragOver === 'remote'" class="pointer-events-none absolute inset-2 z-10 flex items-center justify-center rounded-xl border-2 border-dashed border-primary bg-primary/10 backdrop-blur-[1px]">
          <div class="rounded-lg bg-card px-4 py-3 text-center shadow-lg">
            <p class="text-sm font-semibold text-primary">Drop to upload</p>
            <p class="mt-1 font-mono text-xs text-muted-foreground">{{ remotePath }}</p>
          </div>
        </div>
      </section>
    </div>
    <div v-if="contextMenu" class="fixed z-[70] min-w-44 overflow-hidden rounded-lg border border-border bg-popover py-1 shadow-xl" :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }" @click.stop>
      <p class="truncate border-b border-border px-3 py-1.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">{{ contextMenu.entry.name }}</p>
      <template v-if="contextMenu.side === 'local'">
        <button v-if="!contextMenu.entry.isDir && session" class="w-full px-3 py-2 text-left text-sm hover:bg-accent" @click="upload(contextMenu.entry); contextMenu = null">Upload</button>
        <button v-if="!contextMenu.entry.isDir" class="w-full px-3 py-2 text-left text-sm hover:bg-accent" @click="copyLocalFile">Copy</button>
        <button class="w-full px-3 py-2 text-left text-sm hover:bg-accent" @click="copyText(contextMenu.entry.path || '')">Copy path</button>
        <button class="w-full border-t border-border px-3 py-2 text-left text-sm hover:bg-accent" @click="loadLocal(); contextMenu = null">Refresh local</button>
      </template>
      <template v-else>
        <button v-if="!contextMenu.entry.isDir" class="w-full px-3 py-2 text-left text-sm hover:bg-accent" @click="download(contextMenu.entry); contextMenu = null">Download</button>
        <button class="w-full px-3 py-2 text-left text-sm hover:bg-accent" @click="compressRemote(contextMenu.entry)">Compress to ZIP</button>
        <button class="w-full px-3 py-2 text-left text-sm hover:bg-accent" @click="copyText(joinRemote(remotePath, contextMenu.entry.name))">Copy remote path</button>
        <button class="w-full px-3 py-2 text-left text-sm text-destructive hover:bg-destructive/10" @click="askDelete(contextMenu.entry)">Delete</button>
        <button class="w-full border-t border-border px-3 py-2 text-left text-sm hover:bg-accent" @click="loadRemote(); contextMenu = null">Refresh remote</button>
      </template>
    </div>
    <ConfirmDialog
      :open="!!pendingDelete"
      title="Delete remote item?"
      :message="pendingDelete ? `“${pendingDelete.name}” will be permanently removed from the remote host.` : ''"
      confirm-text="Delete"
      variant="destructive"
      @update:open="(open) => { if (!open) pendingDelete = null; }"
      @confirm="pendingDelete && deleteRemote(pendingDelete)"
    />
  </div>
</template>
