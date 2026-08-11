<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/lib/utils";
import { buttonVariants } from "./variants";
import type { VariantProps } from "class-variance-authority";

type ButtonVariants = VariantProps<typeof buttonVariants>;

interface Props {
  variant?: ButtonVariants["variant"];
  size?: ButtonVariants["size"];
  as?: string;
  class?: string;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  variant: "default",
  size: "default",
  as: "button",
});

const classes = computed(() =>
  cn(buttonVariants({ variant: props.variant, size: props.size }), props.class)
);
</script>

<template>
  <component :is="as" :class="classes" :disabled="disabled" :aria-disabled="disabled || undefined">
    <slot />
  </component>
</template>
