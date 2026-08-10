<script setup lang="ts">
import { ref, computed } from "vue";
import { useHostsStore, type Host } from "@/stores/hosts";

const hostsStore = useHostsStore();
const expandedGroups = ref<Set<number | string>>(new Set(["ungrouped"]));

const emit = defineEmits<{
  select: [hostId: string];
}>();

interface GroupNode {
  id: number | string;
  name: string;
  hosts: Host[];
}

const groupedHosts = computed<GroupNode[]>(() => {
  const groups: GroupNode[] = [];
  const groupMap = new Map<number, Host[]>();
  const ungrouped: Host[] = [];

  for (const host of hostsStore.filteredHosts) {
    if (host.groupId) {
      if (!groupMap.has(host.groupId)) groupMap.set(host.groupId, []);
      groupMap.get(host.groupId)!.push(host);
    } else {
      ungrouped.push(host);
    }
  }

  // Named groups
  for (const group of hostsStore.groups) {
    groups.push({
      id: group.id,
      name: group.name,
      hosts: groupMap.get(group.id) || [],
    });
  }

  // Ungrouped
  if (ungrouped.length > 0) {
    groups.push({ id: "ungrouped", name: "Ungrouped", hosts: ungrouped });
  }

  return groups.filter((g) => g.hosts.length > 0);
});

function toggleGroup(id: number | string) {
  if (expandedGroups.value.has(id)) {
    expandedGroups.value.delete(id);
  } else {
    expandedGroups.value.add(id);
  }
}
</script>

<template>
  <div class="space-y-1">
    <div v-for="group in groupedHosts" :key="group.id">
      <!-- Group header -->
      <button
        class="flex w-full items-center gap-2 rounded-lg px-3 py-1.5 text-xs font-medium text-muted-foreground transition-smooth hover:bg-accent hover:text-foreground"
        @click="toggleGroup(group.id)"
      >
        <svg
          class="h-3 w-3 transition-transform duration-200"
          :class="expandedGroups.has(group.id) ? 'rotate-90' : ''"
          fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
        </svg>
        <svg class="h-3.5 w-3.5" fill="currentColor" viewBox="0 0 20 20">
          <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
        </svg>
        <span>{{ group.name }}</span>
        <span class="ml-auto text-[10px] text-muted-foreground/60">{{ group.hosts.length }}</span>
      </button>

      <!-- Hosts in group -->
      <Transition name="expand">
        <div v-if="expandedGroups.has(group.id)" class="ml-5 space-y-0.5 overflow-hidden">
          <button
            v-for="host in group.hosts"
            :key="host.id"
            class="flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-xs transition-smooth hover:bg-accent"
            @click="emit('select', host.id)"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-muted-foreground/40" />
            <span class="truncate flex-1 text-left">{{ host.name }}</span>
            <span v-if="host.favorite" class="text-yellow-500 text-[10px]">&#9733;</span>
          </button>
        </div>
      </Transition>
    </div>
  </div>
</template>

<style scoped>
.expand-enter-active, .expand-leave-active {
  transition: all 0.2s ease;
  max-height: 500px;
}
.expand-enter-from, .expand-leave-to {
  opacity: 0;
  max-height: 0;
}
</style>
