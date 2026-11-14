<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ChevronRight,
  MoreHorizontal,
  Search,
  Trash2,
  X,
} from "lucide-vue-next";
import { api } from "../api";
import LessonModal from "../components/LessonModal.vue";
import StudentDetailModal from "../components/StudentDetailModal.vue";
import StudentModal from "../components/StudentModal.vue";
import { useUiStore } from "../stores/ui";
import type { Lesson, Student } from "../types";
import {
  currentMonth,
  formatNumber,
  unitsOf,
} from "../utils";

const route = useRoute();
const router = useRouter();
const ui = useUiStore();
const students = ref<Student[]>([]);
const lessons = ref<Lesson[]>([]);
const loading = ref(true);
const search = ref("");
const studentModalOpen = ref(false);
const editingStudent = ref<Student | null>(null);
const detailStudent = ref<Student | null>(null);
const lessonModalOpen = ref(false);
const editingLesson = ref<Lesson | null>(null);
const month = currentMonth();
const swipeOffsets = reactive<Record<string, number>>({});
const openSwipeId = ref("");
const swipingStudentId = ref("");
const deletingStudentId = ref("");
const SWIPE_ACTION_WIDTH = 88;
let swipeStart: {
  id: string;
  x: number;
  y: number;
  base: number;
  horizontal: boolean;
} | null = null;
let suppressClickUntil = 0;

const filtered = computed(() => {
  const term = search.value.trim();
  if (!term) return students.value;
  return students.value.filter(
    (student) => student.name.includes(term),
  );
});
const monthUnits = computed(() =>
  lessons.value.reduce((sum, lesson) => sum + unitsOf(lesson.durationMinutes), 0),
);

function metrics(studentId: string) {
  const items = lessons.value.filter((lesson) => lesson.studentId === studentId);
  return {
    totalUnits: items.reduce((sum, lesson) => sum + unitsOf(lesson.durationMinutes), 0),
  };
}

async function load() {
  closeSwipe();
  loading.value = true;
  try {
    [students.value, lessons.value] = await Promise.all([
      api.students(),
      api.lessons({ month }),
    ]);
    const queryId = String(route.query.student ?? "");
    if (queryId) {
      detailStudent.value = students.value.find((student) => student.id === queryId) ?? null;
    }
  } catch (caught) {
    ui.notify(caught instanceof Error ? caught.message : "加载失败", "error");
  } finally {
    loading.value = false;
  }
}

function openAdd() {
  editingStudent.value = null;
  studentModalOpen.value = true;
}

function openEdit(student: Student) {
  closeSwipe();
  detailStudent.value = null;
  editingStudent.value = student;
  studentModalOpen.value = true;
}

async function studentSaved() {
  studentModalOpen.value = false;
  editingStudent.value = null;
  await load();
}

function openDetail(student: Student) {
  if (Date.now() < suppressClickUntil) return;
  if (openSwipeId.value) {
    closeSwipe();
    return;
  }
  detailStudent.value = student;
  router.replace({ name: "students", query: { student: student.id } });
}

function closeDetail() {
  detailStudent.value = null;
  router.replace({ name: "students" });
}

async function lessonSaved() {
  lessonModalOpen.value = false;
  await load();
}

function editLessonFromDetail(lesson: Lesson) {
  detailStudent.value = null;
  editingLesson.value = lesson;
  lessonModalOpen.value = true;
}

function startSwipe(student: Student, event: PointerEvent) {
  if (event.pointerType === "mouse" && event.button !== 0) return;
  if ((event.target as HTMLElement).closest("button, input, select, textarea, a")) return;
  swipeStart = {
    id: student.id,
    x: event.clientX,
    y: event.clientY,
    base: swipeOffsets[student.id] ?? 0,
    horizontal: false,
  };
}

function moveSwipe(student: Student, event: PointerEvent) {
  if (!swipeStart || swipeStart.id !== student.id) return;
  const deltaX = event.clientX - swipeStart.x;
  const deltaY = event.clientY - swipeStart.y;
  if (!swipeStart.horizontal) {
    if (Math.abs(deltaX) < 8 && Math.abs(deltaY) < 8) return;
    if (Math.abs(deltaY) >= Math.abs(deltaX)) {
      swipeStart = null;
      return;
    }
    swipeStart.horizontal = true;
    swipingStudentId.value = student.id;
    (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);
  }
  event.preventDefault();
  swipeOffsets[student.id] = Math.max(
    -SWIPE_ACTION_WIDTH,
    Math.min(0, swipeStart.base + deltaX),
  );
}

function finishSwipe(student: Student, event: PointerEvent) {
  if (!swipeStart || swipeStart.id !== student.id) return;
  if (swipeStart.horizontal) {
    const open = (swipeOffsets[student.id] ?? 0) < -SWIPE_ACTION_WIDTH / 2;
    swipeOffsets[student.id] = open ? -SWIPE_ACTION_WIDTH : 0;
    openSwipeId.value = open ? student.id : "";
    suppressClickUntil = Date.now() + 350;
    (event.currentTarget as HTMLElement).releasePointerCapture?.(event.pointerId);
  }
  swipeStart = null;
  swipingStudentId.value = "";
}

function closeSwipe() {
  Object.keys(swipeOffsets).forEach((id) => {
    swipeOffsets[id] = 0;
  });
  openSwipeId.value = "";
}

function revealSwipe(studentId: string) {
  closeSwipe();
  swipeOffsets[studentId] = -SWIPE_ACTION_WIDTH;
  openSwipeId.value = studentId;
}

async function deleteStudent(student: Student) {
  if (deletingStudentId.value) return;
  deletingStudentId.value = student.id;
  try {
    await api.deleteStudent(student.id);
    students.value = students.value.filter((item) => item.id !== student.id);
    lessons.value = lessons.value.filter((lesson) => lesson.studentId !== student.id);
    if (detailStudent.value?.id === student.id) {
      closeDetail();
    }
    delete swipeOffsets[student.id];
    openSwipeId.value = "";
    ui.notify("已删除");
  } catch (caught) {
    ui.notify(caught instanceof Error ? caught.message : "删除失败", "error");
  } finally {
    deletingStudentId.value = "";
  }
}

function handleDocumentClick(event: MouseEvent) {
  if (!openSwipeId.value) return;
  if (event.target instanceof Element && event.target.closest(".student-swipe")) return;
  closeSwipe();
}

function handleDocumentKeydown(event: KeyboardEvent) {
  if (event.key === "Escape" && openSwipeId.value) {
    closeSwipe();
  }
}

watch(
  () => route.query.student,
  (studentId) => {
    if (!studentId) {
      detailStudent.value = null;
      return;
    }
    detailStudent.value = students.value.find((student) => student.id === String(studentId)) ?? null;
  },
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
  <div class="students-page">
    <section class="students-toolbar">
      <div class="student-search">
        <Search :size="18" aria-hidden="true" />
        <input v-model="search" placeholder="搜索" aria-label="搜索学生" />
        <button v-if="search" class="icon-button" aria-label="清空搜索" @click="search = ''">
          <X :size="16" aria-hidden="true" />
        </button>
      </div>
      <div class="student-toolbar-summary">
        <span class="student-lesson-total" aria-live="polite">
          <span class="student-lesson-period">本月</span>
          <strong>{{ formatNumber(monthUnits) }}</strong>
          <span>课时</span>
        </span>
        <span class="student-count" aria-live="polite">
          <span class="student-count-label">共</span>
          <strong>{{ students.length }}</strong>
          <span class="student-count-unit-desktop">名学生</span>
          <span class="student-count-unit-mobile">人</span>
        </span>
      </div>
      <button
        class="button button-primary student-add-action"
        @click="openAdd"
      >
        添加学生
      </button>
    </section>

    <div v-if="loading" class="loading-panel">
      <span class="spinner" />
      加载中
    </div>

    <section v-else-if="filtered.length" class="panel student-list">
      <div
        v-for="student in filtered"
        :key="student.id"
        class="student-swipe"
      >
        <button
          class="student-swipe-delete"
          :aria-label="`删除 ${student.name} 及其课程记录`"
          :disabled="deletingStudentId === student.id"
          @focus="revealSwipe(student.id)"
          @click.stop="deleteStudent(student)"
        >
          <Trash2 :size="18" aria-hidden="true" />
          删除
        </button>
        <article
          class="student-row"
          :class="{ swiping: swipingStudentId === student.id }"
          :style="{ transform: `translateX(${swipeOffsets[student.id] ?? 0}px)` }"
          @pointerdown="startSwipe(student, $event)"
          @pointermove="moveSwipe(student, $event)"
          @pointerup="finishSwipe(student, $event)"
          @pointercancel="finishSwipe(student, $event)"
          @click="openDetail(student)"
        >
          <span class="student-row-main">
            <span class="student-row-name">
              <strong>{{ student.name }}</strong>
            </span>
          </span>
          <span class="student-row-metric">
            <strong>{{ formatNumber(metrics(student.id).totalUnits) }}</strong>
            <small>本月课时</small>
          </span>
          <button
            class="icon-button subtle"
            :aria-label="`编辑 ${student.name}`"
            @click.stop="openEdit(student)"
          >
            <MoreHorizontal :size="19" aria-hidden="true" />
          </button>
          <ChevronRight :size="16" class="row-chevron" aria-hidden="true" />
        </article>
      </div>
    </section>

    <div v-else class="empty-state">
      <h3>暂无学生</h3>
    </div>

    <StudentModal
      :open="studentModalOpen"
      :student="editingStudent"
      @close="studentModalOpen = false"
      @saved="studentSaved"
    />
    <StudentDetailModal
      :open="Boolean(detailStudent)"
      :student="detailStudent"
      :month="month"
      @close="closeDetail"
      @edit="openEdit"
      @edit-lesson="editLessonFromDetail"
    />
    <LessonModal
      :open="lessonModalOpen"
      :students="students"
      :lesson="editingLesson"
      @close="lessonModalOpen = false"
      @saved="lessonSaved"
    />
  </div>
</template>
