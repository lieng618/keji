<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";
import { RouterView, useRoute, useRouter } from "vue-router";
import {
  CalendarDays,
  LogOut,
  Plus,
  UsersRound,
  X,
} from "lucide-vue-next";
import { useAuthStore } from "../stores/auth";
import { useUiStore } from "../stores/ui";

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const ui = useUiStore();
const sidebarOpen = ref(false);
const profileOpen = ref(false);
const loggingOut = ref(false);
const profileDrawer = ref<HTMLElement | null>(null);
const profileClose = ref<HTMLElement | null>(null);
let profileSwipeStart: { x: number; y: number } | null = null;
let profilePreviousFocus: HTMLElement | null = null;

const navigation = [
  { name: "dashboard", label: "课程", icon: CalendarDays },
  { name: "students", label: "学生", icon: UsersRound },
];

const activeName = computed(() => String(route.name ?? "dashboard"));

function navigate(name: string) {
  sidebarOpen.value = false;
  router.push({ name });
}

function newLesson() {
  sidebarOpen.value = false;
  router.push({ name: "dashboard", query: { new: "1" } });
}

function openProfile() {
  sidebarOpen.value = false;
  profileOpen.value = true;
}

function closeProfile() {
  profileOpen.value = false;
}

function startProfileSwipe(event: PointerEvent) {
  if (activeName.value !== "dashboard" || profileOpen.value) return;
  if (event.pointerType === "mouse" && event.button !== 0) return;
  if (event.clientX < window.innerWidth - 28) return;
  profileSwipeStart = { x: event.clientX, y: event.clientY };
  (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);
}

function moveProfileSwipe(event: PointerEvent) {
  if (!profileSwipeStart) return;
  const deltaX = event.clientX - profileSwipeStart.x;
  const deltaY = event.clientY - profileSwipeStart.y;
  if (deltaX < -28 && Math.abs(deltaX) > Math.abs(deltaY)) {
    event.preventDefault();
    profileSwipeStart = null;
    openProfile();
  }
}

function finishProfileSwipe(event: PointerEvent) {
  profileSwipeStart = null;
  (event.currentTarget as HTMLElement).releasePointerCapture?.(event.pointerId);
}

function profileKeydown(event: KeyboardEvent) {
  if (!profileOpen.value) return;
  if (event.key === "Escape") {
    event.preventDefault();
    closeProfile();
    return;
  }
  if (event.key !== "Tab" || !profileDrawer.value) return;
  const focusable = profileDrawer.value.querySelectorAll<HTMLElement>(
    'button:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])',
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

async function logout() {
  if (loggingOut.value) return;
  loggingOut.value = true;
  try {
    await auth.logout();
    profileOpen.value = false;
    await router.push({ name: "login" });
  } catch (caught) {
    ui.notify(caught instanceof Error ? caught.message : "退出登录失败", "error");
  } finally {
    loggingOut.value = false;
  }
}

watch(profileOpen, async (open) => {
  document.body.classList.toggle("profile-drawer-open", open);
  if (open) {
    profilePreviousFocus = document.activeElement as HTMLElement | null;
    document.addEventListener("keydown", profileKeydown);
    await nextTick();
    profileClose.value?.focus();
  } else {
    document.removeEventListener("keydown", profileKeydown);
    profilePreviousFocus?.focus();
    profilePreviousFocus = null;
  }
});

watch(activeName, () => {
  profileOpen.value = false;
});

onBeforeUnmount(() => {
  document.body.classList.remove("profile-drawer-open");
  document.removeEventListener("keydown", profileKeydown);
});
</script>

<template>
  <div class="app-shell">
    <a class="skip-link" href="#main-content">跳到主要内容</a>
    <button
      v-if="sidebarOpen"
      class="drawer-backdrop"
      aria-label="关闭菜单"
      @click="sidebarOpen = false"
    />
    <aside class="sidebar" :class="{ 'sidebar-open': sidebarOpen }">
      <div class="brand">
        <span class="brand-mark" aria-hidden="true"><span /><span /><span /></span>
        <span class="brand-copy">
          <strong>课迹</strong>
        </span>
        <button class="icon-button sidebar-close" aria-label="关闭菜单" @click="sidebarOpen = false">
          <X :size="20" aria-hidden="true" />
        </button>
      </div>

      <nav class="side-nav" aria-label="主导航">
        <button
          v-for="item in navigation.slice(0, 1)"
          :key="item.name"
          class="nav-item"
          :class="{ active: activeName === item.name }"
          :aria-current="activeName === item.name ? 'page' : undefined"
          @click="navigate(item.name)"
        >
          <component :is="item.icon" :size="19" :stroke-width="1.8" aria-hidden="true" />
          <span>{{ item.label }}</span>
          <span v-if="activeName === item.name" class="nav-pip" aria-hidden="true" />
        </button>
        <button
          type="button"
          class="nav-item desktop-record"
          aria-label="记录一节课"
          @click="newLesson"
        >
          <Plus :size="19" :stroke-width="1.8" aria-hidden="true" />
          <span>记录</span>
        </button>
        <button
          v-for="item in navigation.slice(1)"
          :key="item.name"
          class="nav-item"
          :class="{ active: activeName === item.name }"
          :aria-current="activeName === item.name ? 'page' : undefined"
          @click="navigate(item.name)"
        >
          <component :is="item.icon" :size="19" :stroke-width="1.8" aria-hidden="true" />
          <span>{{ item.label }}</span>
          <span v-if="activeName === item.name" class="nav-pip" aria-hidden="true" />
        </button>
      </nav>

      <div class="sidebar-bottom">
        <div class="teacher-card">
          <button class="teacher-profile" aria-label="打开个人中心" @click="openProfile">
            <strong>{{ auth.user?.name }}</strong>
          </button>
          <button class="icon-button subtle" aria-label="退出登录" :disabled="loggingOut" @click="logout">
            <LogOut :size="17" aria-hidden="true" />
          </button>
        </div>
      </div>
    </aside>

    <div class="app-main">
      <main id="main-content" class="page-content" tabindex="-1">
        <RouterView />
      </main>
    </div>

    <div
      v-if="activeName === 'dashboard' && !profileOpen"
      class="profile-swipe-zone"
      aria-hidden="true"
      @pointerdown="startProfileSwipe"
      @pointermove="moveProfileSwipe"
      @pointerup="finishProfileSwipe"
      @pointercancel="finishProfileSwipe"
    />

    <template v-if="profileOpen">
      <button class="profile-backdrop" aria-label="关闭个人中心" @click="closeProfile" />
      <aside
        ref="profileDrawer"
        class="profile-drawer"
        role="dialog"
        aria-modal="true"
        aria-label="个人中心"
      >
        <div class="profile-drawer-head">
          <button ref="profileClose" class="icon-button" aria-label="关闭" @click="closeProfile">
            <X :size="20" aria-hidden="true" />
          </button>
        </div>
        <div class="profile-card">
          <h2>{{ auth.user?.name }}</h2>
        </div>
        <div class="profile-drawer-actions">
          <button class="button button-primary profile-logout" :disabled="loggingOut" @click="logout">
            <span v-if="loggingOut" class="spinner light" />
            <LogOut v-else :size="18" aria-hidden="true" />
            {{ loggingOut ? "正在退出" : "退出登录" }}
          </button>
        </div>
      </aside>
    </template>

    <nav class="mobile-nav" aria-label="移动端主导航">
      <button
        v-for="item in navigation.slice(0, 1)"
        :key="item.name"
        :class="{ active: activeName === item.name }"
        @click="navigate(item.name)"
      >
        <component :is="item.icon" :size="20" aria-hidden="true" />
        <span>{{ item.label }}</span>
      </button>
      <button class="mobile-add" aria-label="记一节课" @click="newLesson">
        <Plus :size="27" :stroke-width="2.3" aria-hidden="true" />
        <span class="sr-only">记课</span>
      </button>
      <button
        v-for="item in navigation.slice(1)"
        :key="item.name"
        :class="{ active: activeName === item.name }"
        @click="navigate(item.name)"
      >
        <component :is="item.icon" :size="20" aria-hidden="true" />
        <span>{{ item.label }}</span>
      </button>
    </nav>
  </div>
</template>
