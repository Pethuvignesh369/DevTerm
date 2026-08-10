<script setup lang="ts">
defineProps<{
  class?: string;
}>();
</script>

<template>
  <div :class="['animate-pulse rounded-md bg-muted', $props.class]" />
</template>
