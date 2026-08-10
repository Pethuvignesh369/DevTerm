import { defineStore } from "pinia";
import { ref } from "vue";

export const useAppLockStore = defineStore("applock", () => {
  const locked = ref(false);
  const lockTimeout = ref(5 * 60 * 1000); // 5 minutes default
  const enabled = ref(false);
  let idleTimer: ReturnType<typeof setTimeout> | null = null;

  function lock() {
    locked.value = true;
  }

  function unlock() {
    locked.value = false;
    resetIdleTimer();
  }

  function resetIdleTimer() {
    if (!enabled.value) return;
    if (idleTimer) clearTimeout(idleTimer);
    idleTimer = setTimeout(() => {
      lock();
    }, lockTimeout.value);
  }

  function setEnabled(val: boolean) {
    enabled.value = val;
    if (val) {
      resetIdleTimer();
      // Listen for user activity
      document.addEventListener("mousemove", resetIdleTimer);
      document.addEventListener("keydown", resetIdleTimer);
      document.addEventListener("click", resetIdleTimer);
    } else {
      if (idleTimer) clearTimeout(idleTimer);
      document.removeEventListener("mousemove", resetIdleTimer);
      document.removeEventListener("keydown", resetIdleTimer);
      document.removeEventListener("click", resetIdleTimer);
    }
  }

  function setLockTimeout(ms: number) {
    lockTimeout.value = ms;
    if (enabled.value) resetIdleTimer();
  }

  return { locked, enabled, lockTimeout, lock, unlock, setEnabled, setLockTimeout };
});
