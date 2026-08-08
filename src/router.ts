import { createRouter, createWebHistory } from "vue-router";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      name: "connections",
      component: () => import("./views/ConnectionsView.vue"),
    },
    {
      path: "/terminal",
      name: "terminal",
      component: () => import("./views/TerminalView.vue"),
    },
    {
      path: "/keys",
      name: "keys",
      component: () => import("./views/KeysView.vue"),
    },
    {
      path: "/snippets",
      name: "snippets",
      component: () => import("./views/SnippetsView.vue"),
    },
    {
      path: "/history",
      name: "history",
      component: () => import("./views/HistoryView.vue"),
    },
    {
      path: "/files",
      name: "files",
      component: () => import("./views/FilesView.vue"),
    },
    {
      path: "/forwards",
      name: "forwards",
      component: () => import("./views/ForwardsView.vue"),
    },
    {
      path: "/dashboard",
      name: "dashboard",
      component: () => import("./views/DashboardView.vue"),
    },
    {
      path: "/settings",
      name: "settings",
      component: () => import("./views/SettingsView.vue"),
    },
  ],
});

export default router;
