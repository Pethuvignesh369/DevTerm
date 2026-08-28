<script setup lang="ts">
import { ref, watch } from "vue";
import { useHostsStore } from "@/stores/hosts";
import { useKeysStore } from "@/stores/keys";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { rpcClient } from "@/lib/rpc-client";

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ "update:open": [value: boolean] }>();

const hostsStore = useHostsStore();
const keysStore = useKeysStore();

const name = ref("");
const hostname = ref("");
const port = ref(22);
const username = ref("root");
const identityId = ref("");
const favorite = ref(false);
const tags = ref("");
const submitting = ref(false);
const error = ref("");

// Auth method selection
const authMethod = ref<"none" | "key" | "password" | "existing">("none");
const selectedKeyId = ref("");
const passwordValue = ref("");
const pemFile = ref("");
const pemPassphrase = ref("");
const pemKeyName = ref("");

watch(
  () => props.open,
  (val) => {
    if (val) {
      name.value = "";
      hostname.value = "";
      port.value = 22;
      username.value = "root";
      identityId.value = "";
      favorite.value = false;
      tags.value = "";
      error.value = "";
      authMethod.value = "none";
      selectedKeyId.value = "";
      passwordValue.value = "";
      pemFile.value = "";
      pemPassphrase.value = "";
      pemKeyName.value = "";
      keysStore.fetchKeys();
      hostsStore.fetchIdentities();
    }
  }
);

function handlePemFileSelect(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  pemKeyName.value = file.name.replace(/\.[^.]+$/, "");
  const reader = new FileReader();
  reader.onload = () => {
    pemFile.value = reader.result as string;
  };
  reader.readAsText(file);
}

async function submit() {
  if (!name.value.trim()) {
    error.value = "Name is required";
    return;
  }
  if (!hostname.value.trim()) {
    error.value = "Hostname is required";
    return;
  }
  if (!username.value.trim()) {
    error.value = "Username is required";
    return;
  }

  submitting.value = true;
  error.value = "";

  try {
    let finalIdentityId = identityId.value || undefined;

    // Create identity based on auth method
    if (authMethod.value === "key" && pemFile.value) {
      // Import the PEM key first
      const keyResult = await rpcClient.call<object, { id: string }>("keys.import", {
        name: pemKeyName.value || name.value + "-key",
        privateKey: pemFile.value,
        passphrase: pemPassphrase.value || undefined,
      });

      // Create an identity that uses this key
      const identResult = await rpcClient.call<object, { id: string }>("identities.create", {
        name: name.value + " (key)",
        authType: "key",
        sshKeyId: keyResult.id,
      });
      finalIdentityId = identResult.id;

    } else if (authMethod.value === "key" && selectedKeyId.value) {
      // Use an already-imported key
      const identResult = await rpcClient.call<object, { id: string }>("identities.create", {
        name: name.value + " (key)",
        authType: "key",
        sshKeyId: selectedKeyId.value,
      });
      finalIdentityId = identResult.id;

    } else if (authMethod.value === "password" && passwordValue.value) {
      // Create identity with password — backend stores in vault
      const identResult = await rpcClient.call<object, { id: string }>("identities.create", {
        name: name.value + " (password)",
        authType: "password",
        password: passwordValue.value,
      });
      finalIdentityId = identResult.id;

    } else if (authMethod.value === "existing") {
      finalIdentityId = identityId.value || undefined;
    }

    await hostsStore.createHost({
      name: name.value.trim(),
      hostname: hostname.value.trim(),
      port: port.value,
      username: username.value.trim(),
      identityId: finalIdentityId,
      favorite: favorite.value,
      tags: tags.value.split(",").map((tag) => tag.trim()).filter(Boolean),
    });
    emit("update:open", false);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="fixed inset-0 bg-black/50" @click="emit('update:open', false)" />
      <div class="relative z-10 w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-lg">
        <h3 class="text-lg font-semibold">Add Host</h3>
        <p class="mt-1 text-sm text-muted-foreground">Configure a new SSH connection.</p>

        <div v-if="error" class="mt-4 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {{ error }}
        </div>

        <form class="mt-4 space-y-4" @submit.prevent="submit">
          <div>
            <label class="mb-1 block text-sm font-medium">Display Name</label>
            <Input v-model="name" placeholder="e.g., Production Server" />
          </div>

          <div class="grid grid-cols-3 gap-3">
            <div class="col-span-2">
              <label class="mb-1 block text-sm font-medium">Hostname / IP</label>
              <Input v-model="hostname" placeholder="192.168.1.100" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium">Port</label>
              <Input v-model.number="port" type="number" />
            </div>
          </div>

          <div>
            <label class="mb-1 block text-sm font-medium">Username</label>
            <Input v-model="username" placeholder="root" />
          </div>

          <!-- Authentication Method -->
          <div>
            <label class="mb-2 block text-sm font-medium">Authentication</label>
            <div class="flex flex-wrap gap-3">
              <label class="flex items-center gap-2 text-sm">
                <input v-model="authMethod" type="radio" value="none" class="accent-primary" />
                None / Agent
              </label>
              <label class="flex items-center gap-2 text-sm">
                <input v-model="authMethod" type="radio" value="key" class="accent-primary" />
                PEM Key
              </label>
              <label class="flex items-center gap-2 text-sm">
                <input v-model="authMethod" type="radio" value="password" class="accent-primary" />
                Password
              </label>
              <label v-if="hostsStore.identities.length > 0" class="flex items-center gap-2 text-sm">
                <input v-model="authMethod" type="radio" value="existing" class="accent-primary" />
                Existing Identity
              </label>
            </div>
          </div>

          <!-- PEM Key auth -->
          <div v-if="authMethod === 'key'" class="space-y-3 rounded-md border border-border p-3">
            <!-- Option: use already imported key -->
            <div v-if="keysStore.keys.length > 0">
              <label class="mb-1 block text-sm font-medium">Use an imported key</label>
              <select
                v-model="selectedKeyId"
                class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="">-- Import new PEM file instead --</option>
                <option v-for="key in keysStore.keys" :key="key.id" :value="key.id">
                  {{ key.name }} ({{ key.keyType }}) - {{ key.fingerprint.slice(0, 20) }}...
                </option>
              </select>
            </div>

            <!-- Or import a new PEM -->
            <template v-if="!selectedKeyId">
              <div>
                <label class="mb-1 block text-sm font-medium">PEM Key File</label>
                <input
                  type="file"
                  accept=".pem,.key,*"
                  class="w-full text-sm text-muted-foreground file:mr-4 file:rounded-md file:border-0 file:bg-primary file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-primary-foreground hover:file:bg-primary/90"
                  @change="handlePemFileSelect"
                />
              </div>
              <div v-if="pemFile" class="rounded bg-muted px-3 py-2 text-xs text-muted-foreground">
                Key loaded ({{ pemFile.length }} chars)
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium">Key Passphrase (if encrypted)</label>
                <Input v-model="pemPassphrase" type="password" placeholder="Leave empty if unencrypted" />
              </div>
            </template>
          </div>

          <!-- Password auth -->
          <div v-if="authMethod === 'password'" class="rounded-md border border-border p-3">
            <label class="mb-1 block text-sm font-medium">Password</label>
            <Input v-model="passwordValue" type="password" placeholder="SSH password" />
          </div>

          <!-- Existing identity -->
          <div v-if="authMethod === 'existing'" class="rounded-md border border-border p-3">
            <label class="mb-1 block text-sm font-medium">Select Identity</label>
            <select
              v-model="identityId"
              class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="">None</option>
              <option v-for="ident in hostsStore.identities" :key="ident.id" :value="ident.id">
                {{ ident.name }} ({{ ident.authType }})
              </option>
            </select>
          </div>

          <label class="flex items-center gap-2 text-sm">
            <input v-model="favorite" type="checkbox" class="accent-primary" />
            Add to favorites
          </label>

          <div>
            <label class="mb-1 block text-sm font-medium">Tags</label>
            <Input v-model="tags" placeholder="production, database, client-a" />
            <p class="mt-1 text-xs text-muted-foreground">Separate tags with commas.</p>
          </div>

          <div class="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" @click="emit('update:open', false)">Cancel</Button>
            <Button type="submit" :disabled="submitting">
              {{ submitting ? "Saving..." : "Add Host" }}
            </Button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>
