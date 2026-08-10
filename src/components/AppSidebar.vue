<script setup lang="ts">
import { ref } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useSessionsStore } from "@/stores/sessions";

const router = useRouter();
const route = useRoute();
const sessionsStore = useSessionsStore();
const expanded = ref(false);

const navItems = [
  { name: "Connections", path: "/", icon: "M5 12h14M12 5l7 7-7 7" },
  { name: "Terminal", path: "/terminal", icon: "M4 17l6-6-6-6M12 19h8" },
  { name: "Keys", path: "/keys", icon: "M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" },
  { name: "Snippets", path: "/snippets", icon: "M16 18l2-2-2-2M8 18l-2-2 2-2M12 2v20" },
  { name: "History", path: "/history", icon: "M12 8v4l3 3m6-3a9 9 0 1 1-18 0 9 9 0 0 1 18 0z" },
  { name: "Files", path: "/files", icon: "M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" },
  { name: "Forwards", path: "/forwards", icon: "M8 7h12m0 0l-4-4m4 4l-4 4M16 17H4m0 0l4-4m-4 4l4 4" },
  { name: "Dashboard", path: "/dashboard", icon: "M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" },
  { name: "Settings", path: "/settings", icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0z" },
];

function isActive(path: string): boolean {
  return route.path === path;
}

const activeSessions = () => {
  return Object.values(sessionsStore.sessions).filter(s => s.status === 'connected').length;
};
</script>

<template>
  <aside
    class="flex h-full flex-col border-r border-border/50 bg-card/80 backdrop-blur-sm transition-all duration-200 ease-out"
    :class="expanded ? 'w-48' : 'w-14'"
    @mouseenter="expanded = true"
    @mouseleave="expanded = false"
  >
    <!-- Logo -->
    <div class="flex h-12 items-center justify-center border-b border-border/50">
      <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M4 17l6-6-6-6M12 19h8" />
        </svg>
      </div>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 flex flex-col gap-1 py-2 px-2 overflow-y-auto scrollbar-thin">
      <button
        v-for="item in navItems"
        :key="item.path"
        class="group relative flex items-center gap-3 rounded-lg px-2.5 py-2 text-sm font-medium transition-all duration-150"
        :class="[
          isActive(item.path)
            ? 'bg-primary/10 text-primary'
            : 'text-muted-foreground hover:bg-accent hover:text-foreground',
        ]"
        :title="!expanded ? item.name : undefined"
        @click="router.push(item.path)"
      >
        <!-- Active indicator -->
        <div
          v-if="isActive(item.path)"
          class="absolute left-0 top-1/2 h-4 w-[3px] -translate-y-1/2 rounded-r-full bg-primary"
        />

        <svg class="h-[18px] w-[18px] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
          <path stroke-linecap="round" stroke-linejoin="round" :d="item.icon" />
        </svg>

        <span
          v-if="expanded"
          class="truncate text-xs whitespace-nowrap animate-fade-in"
        >
          {{ item.name }}
        </span>

        <!-- Session badge -->
        <span
          v-if="item.path === '/terminal' && activeSessions() > 0 && expanded"
          class="ml-auto flex h-4 min-w-4 items-center justify-center rounded-full bg-primary/15 px-1 text-[9px] font-bold text-primary"
        >
          {{ activeSessions() }}
        </span>

        <!-- Dot badge when collapsed -->
        <span
          v-if="item.path === '/terminal' && activeSessions() > 0 && !expanded"
          class="absolute top-1 right-1 h-2 w-2 rounded-full bg-green-500"
        />
      </button>
    </nav>

    <!-- Footer -->
    <div class="border-t border-border/50 p-2 flex justify-center">
      <span v-if="expanded" class="text-[9px] text-muted-foreground/50 animate-fade-in">v0.1.0</span>
    </div>
  </aside>
</template>
