<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { ChevronRight, Pencil } from "lucide-vue-next";
import { api } from "../api";
import { useUiStore } from "../stores/ui";
import type { Lesson, Student } from "../types";
import { durationUnitLabel, formatNumber, unitsOf } from "../utils";
import BaseModal from "./BaseModal.vue";

const props = defineProps<{
  open: boolean;
  student: Student | null;
  month: string;
}>();
const emit = defineEmits<{
  close: [];
  edit: [student: Student];
  editLesson: [lesson: Lesson];
}>();
const ui = useUiStore();

const lessons = ref<Lesson[]>([]);
const loading = ref(false);

const monthLessons = computed(() =>
  lessons.value.filter((lesson) => lesson.date.startsWith(props.month)),
);
const monthUnits = computed(() =>
  monthLessons.value.reduce((sum, lesson) => sum + unitsOf(lesson.durationMinutes), 0),
);
const totalUnits = computed(() =>
  lessons.value.reduce((sum, lesson) => sum + unitsOf(lesson.durationMinutes), 0),
);

watch(
  () => [props.open, props.student?.id],
  async () => {
    if (!props.open || !props.student) return;
    loading.value = true;
    try {
      lessons.value = await api.lessons({ studentId: props.student.id, all: true });
    } catch (caught) {
      ui.notify(caught instanceof Error ? caught.message : "加载失败", "error");
    } finally {
      loading.value = false;
    }
  },
  { immediate: true },
);

</script>

<template>
  <BaseModal :open="open" :title="student?.name ?? '学生'" wide @close="emit('close')">
    <template v-if="student">
      <div class="student-detail-hero">
        <span class="student-detail-pattern" aria-hidden="true" />
        <div class="student-detail-identity">
          <div>
            <h2>{{ student.name }}</h2>
          </div>
        </div>
        <button class="button detail-edit" @click="emit('edit', student)">
          <Pencil :size="16" aria-hidden="true" />
          编辑
        </button>
        <div class="student-detail-metrics">
          <div><strong>{{ formatNumber(monthUnits) }}</strong><span>本月课时</span></div>
          <div><strong>{{ formatNumber(totalUnits) }}</strong><span>累计课时</span></div>
        </div>
      </div>

      <div class="student-detail-body">
        <div v-if="loading" class="detail-skeletons">
          <span /><span /><span />
        </div>
        <template v-else>
          <div v-if="lessons.length" class="detail-timeline">
            <div
              v-for="lesson in lessons"
              :key="lesson.id"
              class="timeline-item"
            >
              <button class="timeline-edit" @click="emit('editLesson', lesson)">
                <span class="timeline-date">
                  <strong>{{ lesson.date.slice(5, 7) }}/{{ lesson.date.slice(8, 10) }}</strong>
                  <small>{{ lesson.startTime }}</small>
                </span>
                <span class="timeline-line"><span /></span>
                <span class="timeline-summary">
                  <strong>{{ durationUnitLabel(lesson.durationMinutes) }}</strong>
                </span>
                <ChevronRight :size="17" aria-hidden="true" />
              </button>
            </div>
          </div>
          <div v-else class="inline-empty detail-empty">暂无记录</div>
        </template>
      </div>
    </template>
  </BaseModal>
</template>
