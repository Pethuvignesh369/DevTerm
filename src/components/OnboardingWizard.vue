<script setup lang="ts">
import { ref, onMounted } from "vue";
import { Button } from "@/components/ui/button";

const STORAGE_KEY = "devterm_onboarding_complete";
const visible = ref(false);
const step = ref(0);

const steps = [
  {
    title: "Welcome to DevTerm",
    description: "A modern SSH client built for DevOps engineers, SREs, and Platform teams.",
    icon: "M4 17l6-6-6-6M12 19h8",
  },
  {
    title: "Add Your First Host",
    description: "Go to Connections and click 'Add Host' — or use the quick-connect bar to type user@hostname and hit Enter.",
    icon: "M5 12h14M12 5l7 7-7 7",
  },
  {
    title: "Import SSH Config",
    description: "Already have hosts in ~/.ssh/config? Click 'Import SSH Config' to bring them all in with one click.",
    icon: "M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12",
  },
  {
    title: "Manage SSH Keys",
    description: "Generate new ED25519/RSA keys or import existing PEM files. All keys are stored securely in your OS keychain.",
    icon: "M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z",
  },
  {
    title: "Command Palette",
    description: "Press Ctrl+K anytime to quickly navigate, search hosts, or connect — all without leaving the keyboard.",
    icon: "M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z",
  },
  {
    title: "You're All Set!",
    description: "DevTerm is ready. Your data stays local, encrypted, and offline. No telemetry, no accounts, no cloud — just SSH.",
    icon: "M5 13l4 4L19 7",
  },
];

onMounted(() => {
  const done = localStorage.getItem(STORAGE_KEY);
  if (!done) {
    visible.value = true;
  }
});

function next() {
  if (step.value < steps.length - 1) {
    step.value++;
  } else {
    finish();
  }
}

function skip() {
  finish();
}

function finish() {
  visible.value = false;
  localStorage.setItem(STORAGE_KEY, "true");
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="fixed inset-0 z-[70] flex items-center justify-center">
        <div class="fixed inset-0 bg-black/70 backdrop-blur-sm" />
        <div class="relative z-10 w-full max-w-md rounded-2xl border border-border bg-card p-8 shadow-2xl animate-scale-in">
          <!-- Progress dots -->
          <div class="flex justify-center gap-1.5 mb-6">
            <div
              v-for="(_, idx) in steps"
              :key="idx"
              class="h-1.5 rounded-full transition-all duration-300"
              :class="idx === step ? 'w-6 bg-primary' : idx < step ? 'w-1.5 bg-primary/50' : 'w-1.5 bg-muted'"
            />
          </div>

          <!-- Icon -->
          <div class="mx-auto mb-5 flex h-16 w-16 items-center justify-center rounded-2xl" :class="step === steps.length - 1 ? 'bg-green-500/10' : 'bg-primary/10'">
            <svg
              class="h-8 w-8"
              :class="step === steps.length - 1 ? 'text-green-500' : 'text-primary'"
              fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75"
            >
              <path stroke-linecap="round" stroke-linejoin="round" :d="steps[step].icon" />
            </svg>
          </div>

          <!-- Content -->
          <div class="text-center">
            <h2 class="text-xl font-bold">{{ steps[step].title }}</h2>
            <p class="mt-3 text-sm text-muted-foreground leading-relaxed">{{ steps[step].description }}</p>
          </div>

          <!-- Actions -->
          <div class="mt-8 flex items-center justify-between">
            <button
              v-if="step < steps.length - 1"
              class="text-xs text-muted-foreground hover:text-foreground transition-smooth"
              @click="skip"
            >
              Skip tour
            </button>
            <div v-else />
            <Button @click="next" class="ml-auto gap-2">
              {{ step === steps.length - 1 ? "Get Started" : "Next" }}
              <svg v-if="step < steps.length - 1" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
              </svg>
            </Button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active, .modal-leave-active { transition: all 0.3s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
</style>
