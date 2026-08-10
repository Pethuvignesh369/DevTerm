<script setup lang="ts">
import { ref, watch } from "vue";
import { useHostsStore, type Host } from "@/stores/hosts";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const props = defineProps<{ open: boolean; host: Host | null }>();
const emit = defineEmits<{ "update:open": [value: boolean] }>();

const hostsStore = useHostsStore();
const name = ref("");
const hostname = ref("");
const port = ref(22);
const username = ref("");
const submitting = ref(false);
const error = ref("");

watch(
  () => props.open,
  (val) => {
    if (val && props.host) {
      name.value = props.host.name;
      hostname.value = props.host.hostname;
      port.value = props.host.port;
      username.value = props.host.username;
      error.value = "";
    }
  }
);

async function submit() {
  if (!props.host) return;
  submitting.value = true;
  error.value = "";
  try {
    await hostsStore.updateHost(props.host.id, {
      name: name.value.trim(),
      hostname: hostname.value.trim(),
      port: port.value,
      username: username.value.trim(),
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
      <div class="fixed inset-0 bg-black/60 backdrop-blur-sm" @click="emit('update:open', false)" />
      <div class="relative z-10 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-2xl animate-scale-in">
        <h3 class="text-lg font-semibold">Edit Host</h3>

        <div v-if="error" class="mt-3 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{{ error }}</div>

        <form class="mt-4 space-y-4" @submit.prevent="submit">
          <div>
            <label class="mb-1.5 block text-xs font-medium text-muted-foreground">Display Name</label>
            <Input v-model="name" />
          </div>
          <div class="grid grid-cols-3 gap-3">
            <div class="col-span-2">
              <label class="mb-1.5 block text-xs font-medium text-muted-foreground">Hostname</label>
              <Input v-model="hostname" />
            </div>
            <div>
              <label class="mb-1.5 block text-xs font-medium text-muted-foreground">Port</label>
              <Input v-model.number="port" type="number" />
            </div>
          </div>
          <div>
            <label class="mb-1.5 block text-xs font-medium text-muted-foreground">Username</label>
            <Input v-model="username" />
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" size="sm" @click="emit('update:open', false)">Cancel</Button>
            <Button type="submit" size="sm" :disabled="submitting">{{ submitting ? "Saving..." : "Save Changes" }}</Button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>
