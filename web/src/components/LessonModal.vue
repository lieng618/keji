<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { Check } from "lucide-vue-next";
import { api } from "../api";
import { useUiStore } from "../stores/ui";
import type { Lesson, LessonPayload, Student, WheelOption } from "../types";
import {
  TIME_OPTIONS,
  endTime,
  oneMonthFromToday,
  todayKey,
  timeRangeMinutes,
} from "../utils";
import BaseModal from "./BaseModal.vue";
import WheelDatePicker from "./WheelDatePicker.vue";
import WheelSelect from "./WheelSelect.vue";

const props = defineProps<{
  open: boolean;
  students: Student[];
  lesson?: Lesson | null;
  presetStudentId?: string;
}>();
const emit = defineEmits<{ close: []; saved: [lesson?: Lesson] }>();
const ui = useUiStore();

const form = reactive<LessonPayload>({
  studentId: "",
  date: todayKey(),
  startTime: "18:00",
  durationMinutes: 60,
  status: "scheduled",
});
const endTimeValue = ref("19:00");
const saving = ref(false);
const fieldErrors = reactive<Record<string, string>>({});
const maxLessonDate = ref(oneMonthFromToday());

const isEdit = computed(() => Boolean(props.lesson?.id));
const startOptions = computed<WheelOption[]>(() =>
  TIME_OPTIONS.slice(0, -1).map((time) => ({ value: time, label: time })),
);
const endOptions = computed<WheelOption[]>(() =>
  TIME_OPTIONS.slice(1).map((time) => ({ value: time, label: time })),
);
const studentOptions = computed<WheelOption[]>(() =>
  props.students.map((student) => ({
    value: student.id,
    label: student.name,
  })),
);

watch(
  () => [props.open, props.lesson, props.presetStudentId, props.students] as const,
  () => {
    if (!props.open) return;
    maxLessonDate.value = oneMonthFromToday();
    Object.keys(fieldErrors).forEach((key) => delete fieldErrors[key]);

    if (props.lesson) {
      form.studentId = props.lesson.studentId;
      form.date = props.lesson.date;
      form.startTime = props.lesson.startTime;
      endTimeValue.value = endTime(props.lesson.startTime, props.lesson.durationMinutes);
      form.durationMinutes = props.lesson.durationMinutes;
      form.status = props.lesson.status;
    } else {
      const now = new Date();
      const startMinutes = Math.min(
        23 * 60,
        Math.ceil((now.getHours() * 60 + now.getMinutes()) / 30) * 30,
      );
      const preferredStudent = props.students.some(
        (student) => student.id === props.presetStudentId,
      )
        ? props.presetStudentId
        : props.students[0]?.id;
      form.studentId = preferredStudent ?? "";
      form.date = todayKey();
      form.startTime = minutesToTime(startMinutes);
      endTimeValue.value = minutesToTime(Math.min(startMinutes + 60, 23 * 60 + 30));
      form.durationMinutes = 60;
      form.status = "scheduled";
    }
  },
  { immediate: true },
);

watch(
  () => form.startTime,
  (value) => {
    if (timeRangeMinutes(value, endTimeValue.value) <= 0) {
      endTimeValue.value = minutesToTime(
        Math.min(timeToMinutes(value) + 60, 23 * 60 + 30),
      );
    }
  },
);

function validate() {
  Object.keys(fieldErrors).forEach((key) => delete fieldErrors[key]);
  if (!form.studentId) fieldErrors.studentId = "选择学生";
  if (!form.date) fieldErrors.date = "选择日期";
  if (!form.startTime) fieldErrors.startTime = "选择开始时间";
  if (!endTimeValue.value) fieldErrors.endTime = "选择结束时间";
  if (timeRangeMinutes(form.startTime, endTimeValue.value) <= 0) {
    fieldErrors.endTime = "结束时间必须晚于开始时间";
  }
  return Object.keys(fieldErrors).length === 0;
}

async function save() {
  if (!validate()) {
    ui.notify(Object.values(fieldErrors)[0] ?? "请检查课程信息", "error");
    return;
  }
  saving.value = true;
  try {
    const payload: LessonPayload = {
      studentId: form.studentId,
      date: form.date,
      startTime: form.startTime,
      durationMinutes: timeRangeMinutes(form.startTime, endTimeValue.value),
      status: form.status,
    };
    const savedLesson = props.lesson
      ? await api.updateLesson(props.lesson.id, payload)
      : await api.createLesson(payload);
    ui.notify(isEdit.value ? "已更新" : "已保存");
    emit("saved", savedLesson);
  } catch (caught) {
    ui.notify(caught instanceof Error ? caught.message : "保存失败", "error");
  } finally {
    saving.value = false;
  }
}

function timeToMinutes(value: string) {
  const [hours, minutes] = value.split(":").map(Number);
  return hours * 60 + minutes;
}

function minutesToTime(value: number) {
  return `${String(Math.floor(value / 60) % 24).padStart(2, "0")}:${String(
    value % 60,
  ).padStart(2, "0")}`;
}

</script>

<template>
  <BaseModal
    :open="open"
    :title="isEdit ? '课程详情' : '记录课程'"
    class-name="lesson-modal"
    @close="emit('close')"
  >
    <form class="lesson-form" novalidate @submit.prevent="save">
      <div class="form-field" :class="{ 'has-error': fieldErrors.studentId }">
        <span>学生 <i>*</i></span>
        <WheelSelect
          :model-value="form.studentId"
          :options="studentOptions"
          :invalid="Boolean(fieldErrors.studentId)"
          :visible-items="3"
          label="选择学生"
          @update:model-value="form.studentId = String($event)"
        />
      </div>

      <div class="form-field" :class="{ 'has-error': fieldErrors.date }">
        <span>日期 <i>*</i></span>
        <WheelDatePicker
          v-model="form.date"
          mode="date"
          :max-date="maxLessonDate"
          :visible-items="3"
        />
      </div>

      <div class="form-row">
        <div class="form-field" :class="{ 'has-error': fieldErrors.startTime }">
          <span>开始 <i>*</i></span>
          <WheelSelect
            :model-value="form.startTime"
            :options="startOptions"
            :invalid="Boolean(fieldErrors.startTime)"
            :visible-items="3"
            label="选择开始时间"
            @update:model-value="form.startTime = String($event)"
          />
        </div>
        <div class="form-field" :class="{ 'has-error': fieldErrors.endTime }">
          <span>结束 <i>*</i></span>
          <WheelSelect
            :model-value="endTimeValue"
            :options="endOptions"
            :invalid="Boolean(fieldErrors.endTime)"
            :visible-items="3"
            label="选择结束时间"
            @update:model-value="endTimeValue = String($event)"
          />
        </div>
      </div>

      <div class="form-field status-field" role="group" aria-labelledby="lesson-status-label">
        <span id="lesson-status-label">状态</span>
        <button
          type="button"
          class="attendance-toggle"
          :class="{ attended: form.status === 'attended' }"
          role="checkbox"
          :aria-checked="form.status === 'attended'"
          :aria-label="form.status === 'attended' ? '已上，点击切换为未上' : '未上，点击切换为已上'"
          @click="form.status = form.status === 'attended' ? 'scheduled' : 'attended'"
        >
          <span class="attendance-state-icon" aria-hidden="true">
            <Check v-if="form.status === 'attended'" :size="13" :stroke-width="3" />
          </span>
          <span>{{ form.status === "attended" ? "已上" : "未上" }}</span>
        </button>
      </div>

      <div class="modal-actions modal-actions-right">
        <div>
          <button type="button" class="button button-secondary" @click="emit('close')">取消</button>
          <button type="submit" class="button button-primary" :disabled="saving">
            <span v-if="saving" class="spinner light" />
            {{ saving ? "保存中" : "保存" }}
          </button>
        </div>
      </div>
    </form>
  </BaseModal>
</template>
