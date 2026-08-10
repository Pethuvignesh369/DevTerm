<script setup lang="ts">
import { ref } from "vue";

const props = defineProps<{
  direction: "horizontal" | "vertical";
}>();

const splitRatio = ref(50); // percentage for first pane
const isDragging = ref(false);
const containerRef = ref<HTMLElement | null>(null);

function startDrag(event: MouseEvent) {
  event.preventDefault();
  isDragging.value = true;

  const onMove = (e: MouseEvent) => {
    if (!containerRef.value) return;
    const rect = containerRef.value.getBoundingClientRect();
    let ratio: number;

    if (props.direction === "horizontal") {
      ratio = ((e.clientX - rect.left) / rect.width) * 100;
    } else {
      ratio = ((e.clientY - rect.top) / rect.height) * 100;
    }

    splitRatio.value = Math.min(80, Math.max(20, ratio));
  };

  const onUp = () => {
    isDragging.value = false;
    document.removeEventListener("mousemove", onMove);
    document.removeEventListener("mouseup", onUp);
  };

  document.addEventListener("mousemove", onMove);
  document.addEventListener("mouseup", onUp);
}
</script>

<template>
  <div
    ref="containerRef"
    class="flex h-full w-full"
    :class="direction === 'horizontal' ? 'flex-row' : 'flex-col'"
  >
    <!-- First pane -->
    <div
      class="min-h-0 min-w-0 overflow-hidden"
      :style="direction === 'horizontal'
        ? { width: `${splitRatio}%` }
        : { height: `${splitRatio}%` }"
    >
      <slot name="first" />
    </div>

    <!-- Divider -->
    <div
      class="relative z-10 flex shrink-0 items-center justify-center transition-colors"
      :class="[
        direction === 'horizontal' ? 'w-1 cursor-col-resize hover:bg-primary/30' : 'h-1 cursor-row-resize hover:bg-primary/30',
        isDragging ? 'bg-primary/50' : 'bg-border/50'
      ]"
      @mousedown="startDrag"
    >
      <div
        class="rounded-full bg-muted-foreground/30"
        :class="direction === 'horizontal' ? 'h-8 w-0.5' : 'h-0.5 w-8'"
      />
    </div>

    <!-- Second pane -->
    <div class="min-h-0 min-w-0 flex-1 overflow-hidden">
      <slot name="second" />
    </div>
  </div>
</template>
