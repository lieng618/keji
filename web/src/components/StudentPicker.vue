<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  useId,
} from "vue";
import { Check, ChevronDown, Search, X } from "lucide-vue-next";
import type { Student } from "../types";

const props = withDefaults(
  defineProps<{
    modelValue: string;
    students: Student[];
    invalid?: boolean;
  }>(),
  {
    invalid: false,
  },
);
const emit = defineEmits<{ "update:modelValue": [value: string] }>();

const root = ref<HTMLElement | null>(null);
const searchInput = ref<HTMLInputElement | null>(null);
const trigger = ref<HTMLButtonElement | null>(null);
const listId = useId();
const errorId = useId();
const open = ref(false);
const search = ref("");

const selectedStudent = computed(
  () => props.students.find((student) => student.id === props.modelValue) ?? null,
);
const normalizedSearch = computed(() =>
  search.value.trim().toLocaleLowerCase("zh-CN"),
);
const filteredStudents = computed(() => {
  if (!normalizedSearch.value) return props.students;
  return props.students.filter((student) =>
    student.name.toLocaleLowerCase("zh-CN").includes(normalizedSearch.value),
  );
});
const visibleStudents = computed(() => filteredStudents.value);

async function toggle() {
  open.value = !open.value;
  if (!open.value) return;
  search.value = "";
  await nextTick();
  searchInput.value?.focus();
}

function close(restoreFocus = false) {
  open.value = false;
  search.value = "";
  if (restoreFocus) {
    void nextTick(() => trigger.value?.focus());
  }
}

function choose(student: Student) {
  emit("update:modelValue", student.id);
  close(true);
}

function handleSearchKeydown(event: KeyboardEvent) {
  if (event.key === "Escape") {
    event.preventDefault();
    close(true);
    return;
  }
  if (event.key === "ArrowDown") {
    event.preventDefault();
    void nextTick(() => {
      root.value
        ?.querySelector<HTMLElement>(".student-picker-list [role='option']")
        ?.focus();
    });
    return;
  }
  if (event.key === "Enter" && visibleStudents.value.length) {
    event.preventDefault();
    choose(visibleStudents.value[0]);
  }
}

function handleOutsidePointer(event: PointerEvent) {
  if (!open.value) return;
  if (event.target instanceof Node && root.value?.contains(event.target)) return;
  close();
}

onMounted(() => {
  document.addEventListener("pointerdown", handleOutsidePointer);
});

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", handleOutsidePointer);
});
</script>

<template>
  <div ref="root" class="student-picker" @keydown.esc.stop.prevent="close(true)">
    <button
      ref="trigger"
      type="button"
      class="student-picker-trigger"
      :class="{ 'is-open': open, 'has-error': invalid }"
      aria-haspopup="listbox"
      :aria-expanded="open"
      :aria-controls="listId"
      :aria-describedby="invalid ? errorId : undefined"
      @click="toggle"
    >
      <span :class="{ placeholder: !selectedStudent }">
        {{ selectedStudent?.name ?? "选择学生" }}
      </span>
      <ChevronDown :size="17" aria-hidden="true" />
    </button>

    <p v-if="invalid" :id="errorId" class="student-picker-error">
      请选择学生
    </p>

    <div v-if="open" class="student-picker-panel">
      <label class="student-picker-search">
        <Search :size="17" aria-hidden="true" />
        <input
          ref="searchInput"
          v-model="search"
          type="search"
          inputmode="search"
          autocomplete="off"
          placeholder="搜索"
          aria-label="搜索"
          :aria-controls="listId"
          aria-autocomplete="list"
          @keydown="handleSearchKeydown"
        />
        <button
          v-if="search"
          type="button"
          class="student-picker-clear"
          aria-label="清空搜索"
          @click="search = ''"
        >
          <X :size="15" aria-hidden="true" />
        </button>
      </label>

      <div :id="listId" class="student-picker-list" role="listbox" aria-label="学生">
        <button
          v-for="student in visibleStudents"
          :key="student.id"
          type="button"
          class="student-picker-option"
          :class="{ selected: student.id === modelValue }"
          role="option"
          :aria-selected="student.id === modelValue"
          @click="choose(student)"
        >
          <span>{{ student.name }}</span>
          <Check
            v-if="student.id === modelValue"
            :size="17"
            stroke-width="2.5"
            aria-hidden="true"
          />
        </button>

        <div v-if="!visibleStudents.length" class="student-picker-empty">
          {{ normalizedSearch ? "没有找到匹配的学生" : "还没有学生" }}
        </div>
      </div>
    </div>
  </div>
</template>
