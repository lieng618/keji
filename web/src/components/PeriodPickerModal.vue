<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useUiStore } from "../stores/ui";
import { oneMonthFromToday } from "../utils";
import BaseModal from "./BaseModal.vue";
import WheelDatePicker from "./WheelDatePicker.vue";

export type PeriodMode = "month" | "custom";

const props = defineProps<{
  open: boolean;
  mode: PeriodMode;
  month: string;
  startDate: string;
  endDate: string;
}>();
const emit = defineEmits<{
  close: [];
  apply: [
    value: {
      mode: PeriodMode;
      month: string;
      startDate: string;
      endDate: string;
    },
  ];
}>();
const ui = useUiStore();

const selectedMode = ref<PeriodMode>(props.mode);
const selectedMonth = ref(props.month);
const start = ref(props.startDate);
const end = ref(props.endDate);
const maxPeriodDate = ref(oneMonthFromToday());
const activeDateField = ref<"start" | "end">("start");
const activeDate = computed({
  get: () => (activeDateField.value === "start" ? start.value : end.value),
  set: (value: string) => {
    if (activeDateField.value === "start") start.value = value;
    else end.value = value;
  },
});

watch(
  () => [props.open, props.mode, props.month, props.startDate, props.endDate] as const,
  () => {
    if (!props.open) return;
    maxPeriodDate.value = oneMonthFromToday();
    selectedMode.value = props.mode;
    const maxMonth = maxPeriodDate.value.slice(0, 7);
    selectedMonth.value = props.month > maxMonth ? maxMonth : props.month;
    start.value =
      props.startDate > maxPeriodDate.value ? maxPeriodDate.value : props.startDate;
    end.value =
      props.endDate > maxPeriodDate.value ? maxPeriodDate.value : props.endDate;
    activeDateField.value = "start";
  },
);

function formatDateValue(value: string) {
  return value ? value.replaceAll("-", ".") : "选择日期";
}

function apply() {
  if (selectedMonth.value > maxPeriodDate.value.slice(0, 7)) {
    ui.notify("最多只能选择未来一个月", "error");
    return;
  }
  if (selectedMode.value === "custom") {
    if (!start.value || !end.value) {
      ui.notify("选择起止日期", "error");
      return;
    }
    if (start.value > end.value) {
      ui.notify("结束日期不能早于开始日期", "error");
      return;
    }
    if (end.value > maxPeriodDate.value) {
      ui.notify("结束日期最多只能选择未来一个月", "error");
      return;
    }
  }
  emit("apply", {
    mode: selectedMode.value,
    month: selectedMonth.value,
    startDate: start.value,
    endDate: end.value,
  });
}
</script>

<template>
  <BaseModal :open="open" title="时间范围" class-name="period-modal" @close="emit('close')">
    <div class="period-modal-body">
      <div class="segmented period-tabs">
        <button :class="{ active: selectedMode === 'month' }" @click="selectedMode = 'month'">
          按月
        </button>
        <button :class="{ active: selectedMode === 'custom' }" @click="selectedMode = 'custom'">
          自定义
        </button>
      </div>

      <WheelDatePicker
        v-if="selectedMode === 'month'"
        v-model="selectedMonth"
        mode="month"
        class="period-month-wheel"
        :max-date="maxPeriodDate"
      />

      <div v-else class="period-custom-picker">
        <div class="period-range-switch" role="tablist" aria-label="选择起止日期">
          <button
            type="button"
            role="tab"
            :aria-selected="activeDateField === 'start'"
            :class="{ active: activeDateField === 'start' }"
            @click="activeDateField = 'start'"
          >
            <span>开始</span>
            <strong>{{ formatDateValue(start) }}</strong>
          </button>
          <button
            type="button"
            role="tab"
            :aria-selected="activeDateField === 'end'"
            :class="{ active: activeDateField === 'end' }"
            @click="activeDateField = 'end'"
          >
            <span>结束</span>
            <strong>{{ formatDateValue(end) }}</strong>
          </button>
        </div>
        <WheelDatePicker v-model="activeDate" mode="date" :max-date="maxPeriodDate" />
      </div>

      <div class="modal-actions modal-actions-right">
        <button class="button button-secondary" @click="emit('close')">取消</button>
        <button class="button button-primary" @click="apply">确定</button>
      </div>
    </div>
  </BaseModal>
</template>
