<script setup lang="ts">
import { computed } from "vue";
import type { WheelOption } from "../types";
import WheelSelect from "./WheelSelect.vue";

const props = withDefaults(
  defineProps<{
    modelValue: string;
    mode?: "month" | "date";
    minYear?: number;
    maxYear?: number;
    maxDate?: string;
    visibleItems?: number;
  }>(),
  { mode: "date", visibleItems: 5 },
);
const emit = defineEmits<{ "update:modelValue": [value: string] }>();

const current = new Date();
const currentYear = current.getFullYear();
const fallbackYear = currentYear;
const fallbackMonth = current.getMonth() + 1;
const fallbackDay = current.getDate();

const parsed = computed(() => {
  const pattern = props.mode === "month" ? /^(\d{4})-(\d{2})$/ : /^(\d{4})-(\d{2})-(\d{2})$/;
  const match = props.modelValue.match(pattern);
  if (!match) {
    return { year: fallbackYear, month: fallbackMonth, day: fallbackDay };
  }
  return {
    year: Number(match[1]),
    month: Number(match[2]),
    day: props.mode === "date" ? Number(match[3]) : 1,
  };
});

const minYear = computed(() => props.minYear ?? currentYear - 20);
const maxYear = computed(() => props.maxYear ?? currentYear + 10);
const maxParts = computed(() => {
  const match = props.maxDate?.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!match) {
    return { year: maxYear.value, month: 12, day: 31 };
  }
  return {
    year: Number(match[1]),
    month: Number(match[2]),
    day: Number(match[3]),
  };
});
const year = computed(() =>
  Math.max(minYear.value, Math.min(maxParts.value.year, parsed.value.year)),
);
const month = computed(() => {
  const upperBound = year.value === maxParts.value.year ? maxParts.value.month : 12;
  return Math.max(1, Math.min(upperBound, parsed.value.month));
});
const daysInMonth = computed(() => new Date(year.value, month.value, 0).getDate());
const day = computed(() => {
  const upperBound =
    year.value === maxParts.value.year && month.value === maxParts.value.month
      ? Math.min(daysInMonth.value, maxParts.value.day)
      : daysInMonth.value;
  return Math.max(1, Math.min(upperBound, parsed.value.day));
});
const normalizedValue = computed(() =>
  props.mode === "month"
    ? `${year.value}-${String(month.value).padStart(2, "0")}`
    : `${year.value}-${String(month.value).padStart(2, "0")}-${String(day.value).padStart(2, "0")}`,
);

const yearOptions = computed<WheelOption[]>(() =>
  Array.from({ length: maxParts.value.year - minYear.value + 1 }, (_, index) => {
    const value = minYear.value + index;
    return { value, label: `${value}年` };
  }),
);
const monthOptions = computed<WheelOption[]>(() =>
  Array.from(
    {
      length: year.value === maxParts.value.year ? maxParts.value.month : 12,
    },
    (_, index) => ({ value: index + 1, label: `${index + 1}月` }),
  ),
);
const dayUpperBound = computed(() =>
  year.value === maxParts.value.year && month.value === maxParts.value.month
    ? Math.min(daysInMonth.value, maxParts.value.day)
    : daysInMonth.value,
);
const dayOptions = computed<WheelOption[]>(() =>
  Array.from({ length: dayUpperBound.value }, (_, index) => ({
    value: index + 1,
    label: `${index + 1}日`,
  })),
);

if (normalizedValue.value !== props.modelValue) emit("update:modelValue", normalizedValue.value);

function update(part: "year" | "month" | "day", value: string | number) {
  const nextYear = part === "year" ? Number(value) : year.value;
  const nextMonth = part === "month" ? Number(value) : month.value;
  const maxDay = new Date(nextYear, nextMonth, 0).getDate();
  const nextDay = Math.min(part === "day" ? Number(value) : day.value, maxDay);
  const nextValue =
    props.mode === "month"
      ? `${nextYear}-${String(nextMonth).padStart(2, "0")}`
      : `${nextYear}-${String(nextMonth).padStart(2, "0")}-${String(nextDay).padStart(2, "0")}`;
  if (nextValue !== props.modelValue) emit("update:modelValue", nextValue);
}
</script>

<template>
  <div class="wheel-date-picker" :class="{ 'is-month': mode === 'month' }">
    <div class="wheel-date-column">
      <WheelSelect
        :model-value="year"
        :options="yearOptions"
        :visible-items="visibleItems"
        label="选择年份"
        @update:model-value="update('year', $event)"
      />
    </div>
    <div class="wheel-date-column">
      <WheelSelect
        :model-value="month"
        :options="monthOptions"
        :visible-items="visibleItems"
        label="选择月份"
        @update:model-value="update('month', $event)"
      />
    </div>
    <div v-if="mode === 'date'" class="wheel-date-column">
      <WheelSelect
        :model-value="day"
        :options="dayOptions"
        :visible-items="visibleItems"
        label="选择日期"
        @update:model-value="update('day', $event)"
      />
    </div>
  </div>
</template>
