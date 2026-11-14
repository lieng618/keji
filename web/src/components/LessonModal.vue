<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { api } from "../api";
import { useUiStore } from "../stores/ui";
import type { Lesson, LessonPayload, Student, WheelOption } from "../types";
import {
  TIME_OPTIONS,
  endTime,
  todayKey,
  timeRangeMinutes,
} from "../utils";
import BaseModal from "./BaseModal.vue";
import StudentPicker from "./StudentPicker.vue";
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
});
const endTimeValue = ref("19:00");
const saving = ref(false);
const fieldErrors = reactive<Record<string, string>>({});
const maxLessonDate = ref(todayKey());

const isEdit = computed(() => Boolean(props.lesson?.id));
const startOptions = computed<WheelOption[]>(() =>
  TIME_OPTIONS.slice(0, -1).map((time) => ({ value: time, label: time })),
);
const endOptions = computed<WheelOption[]>(() =>
  TIME_OPTIONS.slice(1).map((time) => ({ value: time, label: time })),
);
watch(
  () => [props.open, props.lesson, props.presetStudentId, props.students] as const,
  () => {
    if (!props.open) return;
    maxLessonDate.value = todayKey();
    Object.keys(fieldErrors).forEach((key) => delete fieldErrors[key]);

    if (props.lesson) {
      form.studentId = props.lesson.studentId;
      form.date = props.lesson.date;
      form.startTime = props.lesson.startTime;
      endTimeValue.value = endTime(props.lesson.startTime, props.lesson.durationMinutes);
      form.durationMinutes = props.lesson.durationMinutes;
    } else {
      const now = new Date();
      const latestEndMinutes = Math.floor(
        (now.getHours() * 60 + now.getMinutes()) / 30,
      ) * 30;
      const preferredStudent = props.students.some(
        (student) => student.id === props.presetStudentId,
      )
        ? props.presetStudentId
        : props.students[0]?.id;
      form.studentId = preferredStudent ?? "";
      if (latestEndMinutes >= 60) {
        form.date = todayKey();
        form.startTime = minutesToTime(latestEndMinutes - 60);
        endTimeValue.value = minutesToTime(latestEndMinutes);
        form.durationMinutes = 60;
      } else if (latestEndMinutes >= 30) {
        form.date = todayKey();
        form.startTime = "00:00";
        endTimeValue.value = "00:30";
        form.durationMinutes = 30;
      } else {
        const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1);
        form.date = `${yesterday.getFullYear()}-${String(yesterday.getMonth() + 1).padStart(
          2,
          "0",
        )}-${String(yesterday.getDate()).padStart(2, "0")}`;
        form.startTime = "22:30";
        endTimeValue.value = "23:30";
        form.durationMinutes = 60;
      }
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

function selectStudent(studentId: string) {
  form.studentId = studentId;
  delete fieldErrors.studentId;
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
        <StudentPicker
          :model-value="form.studentId"
          :students="students"
          :invalid="Boolean(fieldErrors.studentId)"
          @update:model-value="selectStudent"
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
