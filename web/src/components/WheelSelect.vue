<script setup lang="ts">
import { computed, nextTick, onMounted, ref, useId, watch } from "vue";
import type { WheelOption } from "../types";

const props = withDefaults(
  defineProps<{
    modelValue: string | number;
    options: WheelOption[];
    label: string;
    itemHeight?: number;
    visibleItems?: number;
    invalid?: boolean;
  }>(),
  {
    itemHeight: 38,
    visibleItems: 5,
    invalid: false,
  },
);
const emit = defineEmits<{ "update:modelValue": [value: string | number] }>();
const track = ref<HTMLElement | null>(null);
const listId = useId();
let scrollTimer: number | undefined;
let lastEmitted: string | number | undefined;

const selectedIndex = computed(() => {
  const match = props.options.findIndex((option) => option.value === props.modelValue);
  return match >= 0 ? match : 0;
});
const viewportHeight = computed(() => props.itemHeight * props.visibleItems);
const viewportStyle = computed(() => ({
  "--wheel-item-height": `${props.itemHeight}px`,
  "--wheel-viewport-height": `${viewportHeight.value}px`,
}));

function scrollToIndex(index: number, behavior: ScrollBehavior = "smooth") {
  const element = track.value;
  if (!element) return;
  element.scrollTo({
    top: Math.max(0, index) * props.itemHeight,
    behavior,
  });
}

function emitIndex(index: number) {
  const option = props.options[Math.max(0, Math.min(index, props.options.length - 1))];
  if (!option || option.value === props.modelValue) return;
  lastEmitted = option.value;
  emit("update:modelValue", option.value);
}

function handleScroll() {
  const element = track.value;
  if (!element) return;
  window.clearTimeout(scrollTimer);
  scrollTimer = window.setTimeout(() => {
    emitIndex(Math.round(element.scrollTop / props.itemHeight));
  }, 70);
}

function choose(index: number) {
  emitIndex(index);
  scrollToIndex(index);
}

function keydown(event: KeyboardEvent) {
  const keys: Record<string, number> = {
    ArrowUp: selectedIndex.value - 1,
    ArrowDown: selectedIndex.value + 1,
    PageUp: selectedIndex.value - 3,
    PageDown: selectedIndex.value + 3,
    Home: 0,
    End: props.options.length - 1,
  };
  const next = keys[event.key];
  if (next === undefined) return;
  event.preventDefault();
  const index = Math.max(0, Math.min(next, props.options.length - 1));
  emitIndex(index);
  scrollToIndex(index);
}

watch(
  () => [props.modelValue, props.options.length] as const,
  ([value]) => {
    if (value === lastEmitted) return;
    void nextTick(() => scrollToIndex(selectedIndex.value, "auto"));
  },
  { flush: "post" },
);

onMounted(() => {
  void nextTick(() => scrollToIndex(selectedIndex.value, "auto"));
});
</script>

<template>
  <div class="wheel-select" :style="viewportStyle">
    <span class="wheel-select-highlight" aria-hidden="true" />
    <div
      ref="track"
      class="wheel-select-viewport"
      role="listbox"
      tabindex="0"
      :aria-label="label"
      :aria-invalid="invalid || undefined"
      :aria-activedescendant="`${listId}-${selectedIndex}`"
      @scroll.passive="handleScroll"
      @keydown="keydown"
    >
      <button
        v-for="(option, index) in options"
        :id="`${listId}-${index}`"
        :key="option.value"
        type="button"
        class="wheel-select-option"
        :class="{ selected: option.value === modelValue }"
        role="option"
        :aria-selected="option.value === modelValue"
        tabindex="-1"
        @click="choose(index)"
      >
        {{ option.label }}
      </button>
    </div>
  </div>
</template>
