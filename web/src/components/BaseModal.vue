<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from "vue";
import { X } from "lucide-vue-next";

const props = withDefaults(
  defineProps<{
    open: boolean;
    title: string;
    wide?: boolean;
    className?: string;
  }>(),
  { wide: false, className: "" },
);

const emit = defineEmits<{ close: [] }>();
const dialog = ref<HTMLElement | null>(null);
let previousFocus: HTMLElement | null = null;

function keydown(event: KeyboardEvent) {
  if (!props.open) return;
  if (event.key === "Escape") {
    event.preventDefault();
    emit("close");
    return;
  }
  if (event.key !== "Tab" || !dialog.value) return;
  const focusable = dialog.value.querySelectorAll<HTMLElement>(
    'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])',
  );
  if (!focusable.length) return;
  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
  }
}

watch(
  () => props.open,
  async (open) => {
    if (open) {
      previousFocus = document.activeElement as HTMLElement | null;
      document.body.classList.add("modal-open");
      document.addEventListener("keydown", keydown);
      await nextTick();
      const first = dialog.value?.querySelector<HTMLElement>(
        "input, select, textarea, button",
      );
      first?.focus();
    } else {
      document.body.classList.remove("modal-open");
      document.removeEventListener("keydown", keydown);
      previousFocus?.focus();
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  document.body.classList.remove("modal-open");
  document.removeEventListener("keydown", keydown);
});
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="modal-layer">
      <button class="modal-backdrop" aria-label="关闭窗口" @click="emit('close')" />
      <section
        ref="dialog"
        class="modal-sheet"
        :class="[className, { 'student-detail-modal': wide }]"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
      >
        <div class="modal-header">
          <h2 class="sr-only">{{ title }}</h2>
          <button class="icon-button" aria-label="关闭" @click="emit('close')">
            <X :size="21" aria-hidden="true" />
          </button>
        </div>
        <slot />
      </section>
    </div>
  </Teleport>
</template>
