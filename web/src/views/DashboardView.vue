<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  CalendarRange,
  Check,
  ChevronDown,
  ChevronRight,
  Trash2,
} from "lucide-vue-next";
import { api } from "../api";
import LessonModal from "../components/LessonModal.vue";
import PeriodPickerModal, {
  type PeriodMode,
} from "../components/PeriodPickerModal.vue";
import { useUiStore } from "../stores/ui";
import type { Lesson, Student } from "../types";
import {
  currentMonth,
  durationUnitLabel,
  endTime,
  formatDate,
  formatDateRange,
  formatNumber,
  unitsOf,
} from "../utils";

const route = useRoute();
const router = useRouter();
const ui = useUiStore();
const lessons = ref<Lesson[]>([]);
const students = ref<Student[]>([]);
const loading = ref(true);
const mode = ref<PeriodMode>("month");
const month = ref(currentMonth());
const startDate = ref(monthBounds(month.value).start);
const endDate = ref(monthBounds(month.value).end);
const periodOpen = ref(false);
const modalOpen = ref(false);
const editingLesson = ref<Lesson | null>(null);
const swipeOffsets = reactive<Record<string, number>>({});
const openSwipeId = ref("");
const swipingLessonId = ref("");
const deletingLessonId = ref("");
const updatingLessonId = ref("");
const SWIPE_ACTION_WIDTH = 88;
let swipeStart: {
  id: string;
  x: number;
  y: number;
  base: number;
  horizontal: boolean;
} | null = null;
let suppressClickUntil = 0;

const visibleLessons = computed(() =>
  lessons.value
    .filter((lesson) =>
      mode.value === "month"
        ? lesson.date.startsWith(month.value)
        : lesson.date >= startDate.value && lesson.date <= endDate.value,
    )
    .slice()
    .sort((a, b) => `${b.date}${b.startTime}`.localeCompare(`${a.date}${a.startTime}`)),
);

const grouped = computed(() => {
  const map = new Map<string, Lesson[]>();
  for (const lesson of visibleLessons.value) {
    if (!map.has(lesson.date)) map.set(lesson.date, []);
    map.get(lesson.date)?.push(lesson);
  }
  return [...map.entries()]
    .sort(([left], [right]) => right.localeCompare(left))
    .map(([date, items]) => ({
      date,
      lessons: items.slice().sort((a, b) => a.startTime.localeCompare(b.startTime)),
    }));
});

const totalUnits = computed(() =>
  visibleLessons.value.reduce((sum, lesson) => sum + unitsOf(lesson.durationMinutes), 0),
);
const attendedUnits = computed(() =>
  visibleLessons.value
    .filter((lesson) => lesson.status === "attended")
    .reduce((sum, lesson) => sum + unitsOf(lesson.durationMinutes), 0),
);

const periodLabel = computed(() => {
  if (mode.value === "month") {
    const [year, monthNumber] = month.value.split("-");
    return `${year}年${Number(monthNumber)}月`;
  }
  return formatDateRange(startDate.value, endDate.value);
});

async function load() {
  loading.value = true;
  try {
    [lessons.value, students.value] = await Promise.all([
      api.lessons({ all: true }),
      api.students(),
    ]);
  } catch (caught) {
    ui.notify(caught instanceof Error ? caught.message : "加载失败", "error");
  } finally {
    loading.value = false;
  }
}

function openNew() {
  editingLesson.value = null;
  modalOpen.value = true;
}

function openLesson(lesson: Lesson) {
  if (Date.now() < suppressClickUntil) return;
  if (openSwipeId.value) {
    closeSwipe();
    return;
  }
  editingLesson.value = lesson;
  modalOpen.value = true;
}

function startSwipe(lesson: Lesson, event: PointerEvent) {
  if (event.pointerType === "mouse" && event.button !== 0) return;
  if ((event.target as HTMLElement).closest("select, button")) return;
  swipeStart = {
    id: lesson.id,
    x: event.clientX,
    y: event.clientY,
    base: swipeOffsets[lesson.id] ?? 0,
    horizontal: false,
  };
}

function moveSwipe(lesson: Lesson, event: PointerEvent) {
  if (!swipeStart || swipeStart.id !== lesson.id) return;
  const deltaX = event.clientX - swipeStart.x;
  const deltaY = event.clientY - swipeStart.y;
  if (!swipeStart.horizontal) {
    if (Math.abs(deltaX) < 8 && Math.abs(deltaY) < 8) return;
    if (Math.abs(deltaY) >= Math.abs(deltaX)) {
      swipeStart = null;
      return;
    }
    swipeStart.horizontal = true;
    swipingLessonId.value = lesson.id;
    (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);
  }
  event.preventDefault();
  swipeOffsets[lesson.id] = Math.max(
    -SWIPE_ACTION_WIDTH,
    Math.min(0, swipeStart.base + deltaX),
  );
}

function finishSwipe(lesson: Lesson, event: PointerEvent) {
  if (!swipeStart || swipeStart.id !== lesson.id) return;
  if (swipeStart.horizontal) {
    const open = (swipeOffsets[lesson.id] ?? 0) < -SWIPE_ACTION_WIDTH / 2;
    swipeOffsets[lesson.id] = open ? -SWIPE_ACTION_WIDTH : 0;
    openSwipeId.value = open ? lesson.id : "";
    suppressClickUntil = Date.now() + 350;
    (event.currentTarget as HTMLElement).releasePointerCapture?.(event.pointerId);
  }
  swipeStart = null;
  swipingLessonId.value = "";
}

function closeSwipe() {
  Object.keys(swipeOffsets).forEach((id) => {
    swipeOffsets[id] = 0;
  });
  openSwipeId.value = "";
}

function handleDocumentClick(event: MouseEvent) {
  if (!openSwipeId.value) return;
  if (event.target instanceof Element && event.target.closest(".course-swipe")) return;
  closeSwipe();
}

function handleDocumentKeydown(event: KeyboardEvent) {
  if (event.key === "Escape" && openSwipeId.value) {
    closeSwipe();
  }
}

function revealSwipe(lessonId: string) {
  closeSwipe();
  swipeOffsets[lessonId] = -SWIPE_ACTION_WIDTH;
  openSwipeId.value = lessonId;
}

async function deleteLesson(lesson: Lesson) {
  if (deletingLessonId.value) return;
  deletingLessonId.value = lesson.id;
  try {
    await api.deleteLesson(lesson.id);
    lessons.value = lessons.value.filter((item) => item.id !== lesson.id);
    delete swipeOffsets[lesson.id];
    openSwipeId.value = "";
    ui.notify("已删除");
  } catch (caught) {
    ui.notify(caught instanceof Error ? caught.message : "删除失败", "error");
  } finally {
    deletingLessonId.value = "";
  }
}

async function saved(lesson?: Lesson) {
  modalOpen.value = false;
  editingLesson.value = null;
  if (!lesson) {
    await load();
    return;
  }
  const index = lessons.value.findIndex((item) => item.id === lesson.id);
  if (index >= 0) {
    lessons.value[index] = lesson;
  } else {
    lessons.value = [lesson, ...lessons.value];
  }
}

async function setStatus(lesson: Lesson, status: string) {
  if (updatingLessonId.value) return;
  const nextStatus = status === "attended" ? "attended" : "scheduled";
  updatingLessonId.value = lesson.id;
  try {
    const updated = await api.updateLesson(lesson.id, {
      studentId: lesson.studentId,
      date: lesson.date,
      startTime: lesson.startTime,
      durationMinutes: lesson.durationMinutes,
      status: nextStatus,
    });
    const index = lessons.value.findIndex((item) => item.id === lesson.id);
    if (index >= 0) lessons.value[index] = updated;
  } catch (caught) {
    ui.notify(caught instanceof Error ? caught.message : "更新失败", "error");
  } finally {
    updatingLessonId.value = "";
  }
}

function applyPeriod(value: {
  mode: PeriodMode;
  month: string;
  startDate: string;
  endDate: string;
}) {
  mode.value = value.mode;
  month.value = value.month;
  if (value.mode === "month") {
    const bounds = monthBounds(value.month);
    startDate.value = bounds.start;
    endDate.value = bounds.end;
  } else {
    startDate.value = value.startDate;
    endDate.value = value.endDate;
  }
  periodOpen.value = false;
}

function monthBounds(value: string) {
  const [year, monthNumber] = value.split("-").map(Number);
  const lastDay = new Date(year, monthNumber, 0).getDate();
  return {
    start: `${value}-01`,
    end: `${value}-${String(lastDay).padStart(2, "0")}`,
  };
}

watch(
  () => route.query.new,
  (newLesson) => {
    if (newLesson !== "1") return;
    openNew();
    router.replace({ name: "dashboard" });
  },
  { immediate: true },
);

onMounted(() => {
  void load();
  document.addEventListener("click", handleDocumentClick);
  document.addEventListener("keydown", handleDocumentKeydown);
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleDocumentClick);
  document.removeEventListener("keydown", handleDocumentKeydown);
});
</script>

<template>
  <div class="courses-page">
    <section class="courses-header">
      <button
        class="period-badge"
        :aria-label="`当前时间范围：${periodLabel}，点击修改`"
        @click="periodOpen = true"
      >
        <CalendarRange :size="16" aria-hidden="true" />
        <span class="period-badge-label">{{ periodLabel }}</span>
        <ChevronDown :size="14" aria-hidden="true" />
      </button>
      <div class="course-totals">
        <span>总课时 <strong>{{ formatNumber(totalUnits) }}</strong></span>
        <span>已上 <strong>{{ formatNumber(attendedUnits) }}</strong></span>
      </div>
    </section>

    <div v-if="loading" class="loading-panel">
      <span class="spinner" />
      加载中
    </div>

    <div v-else-if="grouped.length" class="courses-list">
      <section v-for="group in grouped" :key="group.date" class="course-date-group">
        <div class="course-date">{{ formatDate(group.date) }}</div>
        <div class="course-group-list">
          <div
            v-for="lesson in group.lessons"
            :key="lesson.id"
            class="course-swipe"
          >
            <button
              class="course-swipe-delete"
              :aria-label="`删除 ${lesson.studentName} 的课程`"
              :disabled="deletingLessonId === lesson.id"
              @focus="revealSwipe(lesson.id)"
              @click.stop="deleteLesson(lesson)"
            >
              <Trash2 :size="18" aria-hidden="true" />
              删除
            </button>
            <article
              class="course-row"
              :class="{ swiping: swipingLessonId === lesson.id }"
              :style="{ transform: `translateX(${swipeOffsets[lesson.id] ?? 0}px)` }"
              @pointerdown="startSwipe(lesson, $event)"
              @pointermove="moveSwipe(lesson, $event)"
              @pointerup="finishSwipe(lesson, $event)"
              @pointercancel="finishSwipe(lesson, $event)"
              @click="openLesson(lesson)"
            >
              <div class="course-time">
                <strong>{{ lesson.startTime }}-{{ endTime(lesson.startTime, lesson.durationMinutes) }}</strong>
              </div>
              <div class="course-main">
                <strong>{{ lesson.studentName }}</strong>
                <small>{{ durationUnitLabel(lesson.durationMinutes) }}</small>
              </div>
              <button
                type="button"
                class="attendance-toggle"
                :class="{ attended: lesson.status === 'attended' }"
                role="checkbox"
                :aria-checked="lesson.status === 'attended'"
                :aria-label="`${lesson.studentName}：${lesson.status === 'attended' ? '已上，点击切换为未上' : '未上，点击切换为已上'}`"
                :disabled="updatingLessonId === lesson.id"
                @click.stop="setStatus(lesson, lesson.status === 'attended' ? 'scheduled' : 'attended')"
              >
                <span class="attendance-state-icon" aria-hidden="true">
                  <Check v-if="lesson.status === 'attended'" :size="13" :stroke-width="3" />
                </span>
                <span>{{ lesson.status === "attended" ? "已上" : "未上" }}</span>
              </button>
              <ChevronRight :size="16" class="row-chevron" aria-hidden="true" />
            </article>
          </div>
        </div>
      </section>
    </div>

    <div v-else class="empty-state">
      <h3>暂无课程</h3>
    </div>

    <PeriodPickerModal
      :open="periodOpen"
      :mode="mode"
      :month="month"
      :start-date="startDate"
      :end-date="endDate"
      @close="periodOpen = false"
      @apply="applyPeriod"
    />

    <LessonModal
      :open="modalOpen"
      :students="students"
      :lesson="editingLesson"
      @close="modalOpen = false"
      @saved="saved"
    />
  </div>
</template>
