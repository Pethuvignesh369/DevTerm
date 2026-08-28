import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { rpcClient } from "@/lib/rpc-client";
import { useNotificationsStore } from "@/stores/notifications";
import { useRecentsStore } from "@/stores/recents";
import { playConnectSound, playDisconnectSound, playErrorSound } from "@/lib/sounds";

export type ConnectionStatus =
  | "disconnected"
  | "connecting"
  | "connected"
  | "error";

export interface Session {
  id: string;
  hostId: string;
  hostName: string;
  status: ConnectionStatus;
  error?: string;
}

export interface Tab {
  id: string;
  sessionId: string;
  title: string;
}

export interface QuickConnectParams {
  hostname: string;
  port: number;
  username: string;
  password: string;
}

export const useSessionsStore = defineStore("sessions", () => {
  // Use a plain reactive object instead of Map for Vue reactivity
  const sessions = ref<Record<string, Session>>({});
  const tabs = ref<Tab[]>([]);
  const activeTabId = ref<string | null>(null);
  // Split pane state per tab: null = no split, otherwise holds the second session ID
  const splits = ref<Record<string, { sessionId: string; direction: "horizontal" | "vertical" }>>({});

  const activeSession = computed(() => {
    if (!activeTabId.value) return null;
    const tab = tabs.value.find((t) => t.id === activeTabId.value);
    if (!tab) return null;
    return sessions.value[tab.sessionId] ?? null;
  });

  // Subscribe to SSH status notifications
  rpcClient.subscribe("ssh.status", (params: unknown) => {
    const p = params as { sessionId: string; status: ConnectionStatus; error?: string };
    const session = sessions.value[p.sessionId];
    if (session) {
      session.status = p.status;
      if (p.error) session.error = p.error;
      if (p.status === "disconnected") {
        playDisconnectSound();
        const notifications = useNotificationsStore();
        notifications.add({
          type: "warning",
          title: "Disconnected",
          message: `${session.hostName} connection lost`,
        });
      }
    }
  });

  async function connect(hostId: string, hostName: string) {
    const tempId = crypto.randomUUID();
    const session: Session = {
      id: tempId,
      hostId,
      hostName,
      status: "connecting",
    };
    sessions.value[tempId] = session;

    // Create a tab
    const tab: Tab = {
      id: crypto.randomUUID(),
      sessionId: tempId,
      title: hostName,
    };
    tabs.value.push(tab);
    activeTabId.value = tab.id;

    try {
      const result = await rpcClient.call<
        { hostId: string },
        { sessionId: string; hostId: string }
      >("ssh.connect", { hostId });

      // Update session with the real session ID from backend
      delete sessions.value[tempId];
      session.id = result.sessionId;
      session.status = "connected";
      sessions.value[result.sessionId] = session;
      tab.sessionId = result.sessionId;

      const notifications = useNotificationsStore();
      const recentsStore = useRecentsStore();
      notifications.add({
        type: "success",
        title: "Connected",
        message: `Successfully connected to ${hostName}`,
      });
      playConnectSound();
      recentsStore.addRecent({ hostId, hostName, hostname: hostName, username: "", port: 22 });
    } catch (e) {
      session.status = "error";
      session.error = e instanceof Error ? e.message : String(e);

      const notifications = useNotificationsStore();
      notifications.add({
        type: "error",
        title: "Connection Failed",
        message: session.error,
      });
      playErrorSound();
    }
  }

  async function connectQuick(params: QuickConnectParams): Promise<boolean> {
    const hostName = `${params.username}@${params.hostname}:${params.port}`;
    const tempId = crypto.randomUUID();
    const session: Session = { id: tempId, hostId: `quick-${tempId}`, hostName, status: "connecting" };
    sessions.value[tempId] = session;
    const tab: Tab = { id: crypto.randomUUID(), sessionId: tempId, title: hostName };
    tabs.value.push(tab);
    activeTabId.value = tab.id;

    try {
      const result = await rpcClient.call<QuickConnectParams, { sessionId: string; hostId: string }>("ssh.connect", params);
      delete sessions.value[tempId];
      session.id = result.sessionId;
      session.hostId = result.hostId;
      session.status = "connected";
      sessions.value[result.sessionId] = session;
      tab.sessionId = result.sessionId;
      useNotificationsStore().add({ type: "success", title: "Quick connection established", message: `Connected to ${hostName}` });
      playConnectSound();
      return true;
    } catch (e) {
      session.status = "error";
      session.error = e instanceof Error ? e.message : String(e);
      useNotificationsStore().add({ type: "error", title: "Quick connection failed", message: session.error });
      playErrorSound();
      return false;
    }
  }

  async function disconnect(sessionId: string) {
    try {
      await rpcClient.call("ssh.disconnect", { sessionId });
    } catch {
      // Ignore errors on disconnect
    }
    delete sessions.value[sessionId];
    tabs.value = tabs.value.filter((t) => t.sessionId !== sessionId);
    if (activeTabId.value && !tabs.value.find((t) => t.id === activeTabId.value)) {
      activeTabId.value = tabs.value[0]?.id ?? null;
    }
  }

  function closeTab(tabId: string) {
    const tab = tabs.value.find((t) => t.id === tabId);
    if (!tab) return;
    disconnect(tab.sessionId);
  }

  function setActiveTab(tabId: string) {
    activeTabId.value = tabId;
  }

  function splitTab(tabId: string, direction: "horizontal" | "vertical", newSessionId: string) {
    splits.value[tabId] = { sessionId: newSessionId, direction };
  }

  function unsplitTab(tabId: string) {
    delete splits.value[tabId];
  }

  return {
    sessions,
    tabs,
    activeTabId,
    activeSession,
    splits,
    connect,
    connectQuick,
    disconnect,
    closeTab,
    setActiveTab,
    splitTab,
    unsplitTab,
  };
});
