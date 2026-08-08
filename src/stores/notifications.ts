import { defineStore } from "pinia";
import { ref } from "vue";

export type NotificationType = "info" | "success" | "error" | "warning";

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message?: string;
  persistent?: boolean;
}

export const useNotificationsStore = defineStore("notifications", () => {
  const items = ref<Notification[]>([]);
  const backendAvailable = ref(true);

  function add(notification: Omit<Notification, "id">) {
    const id = crypto.randomUUID();
    items.value.push({ ...notification, id });

    // Auto-remove non-persistent notifications after 5s
    if (!notification.persistent) {
      setTimeout(() => {
        remove(id);
      }, 5000);
    }

    return id;
  }

  function remove(id: string) {
    items.value = items.value.filter((n) => n.id !== id);
  }

  function setBackendAvailable(available: boolean) {
    backendAvailable.value = available;
    if (!available) {
      add({
        type: "error",
        title: "Backend unavailable",
        message: "The DevTerm backend process is not responding. Please restart the application.",
        persistent: true,
      });
    }
  }

  return { items, backendAvailable, add, remove, setBackendAvailable };
});
