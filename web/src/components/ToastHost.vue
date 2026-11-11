<script setup lang="ts">
import { Check, CircleAlert } from "lucide-vue-next";
import { useUiStore } from "../stores/ui";

const ui = useUiStore();
</script>

<template>
  <div
    class="toast-region"
    :aria-live="ui.toast?.tone === 'error' ? 'assertive' : 'polite'"
    aria-atomic="true"
  >
    <div v-if="ui.toast" class="toast" :class="ui.toast.tone">
      <span class="toast-icon">
        <Check
          v-if="ui.toast.tone === 'success'"
          :size="15"
          :stroke-width="2.6"
          aria-hidden="true"
        />
        <CircleAlert v-else :size="15" :stroke-width="2.4" aria-hidden="true" />
      </span>
      {{ ui.toast.message }}
    </div>
  </div>
</template>
