<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import {
  ArrowRight,
  Eye,
  EyeOff,
  LockKeyhole,
  UserRound,
} from "lucide-vue-next";
import { ApiError } from "../api";
import { useAuthStore } from "../stores/auth";
import { useUiStore } from "../stores/ui";

const auth = useAuthStore();
const ui = useUiStore();
const router = useRouter();

const setupMode = ref(!auth.initialized);
const name = ref(auth.lastName);
const password = ref("");
const showPassword = ref(false);
const loading = ref(false);

const submitLabel = computed(() => {
  if (loading.value) return setupMode.value ? "创建中" : "登录中";
  return setupMode.value ? "创建账号" : "登录";
});

async function submit() {
  if (loading.value) return;
  if (!name.value.trim()) {
    ui.notify("请输入姓名", "error");
    return;
  }
  if (password.value.length < 8) {
    ui.notify("密码至少需要 8 位", "error");
    return;
  }

  loading.value = true;
  try {
    if (setupMode.value) {
      await auth.setup({
        name: name.value.trim(),
        password: password.value,
      });
      ui.notify("账号已创建");
    } else {
      await auth.login(name.value.trim(), password.value);
      ui.notify("已登录");
    }
    await router.push({ name: "dashboard" });
  } catch (caught) {
    if (
      setupMode.value &&
      caught instanceof ApiError &&
      caught.status === 409
    ) {
      auth.initialized = true;
      setupMode.value = false;
      password.value = "";
      ui.notify("账号已创建，请直接登录");
      return;
    }
    ui.notify(
      caught instanceof Error ? caught.message : "操作失败，请稍后再试",
      "error",
    );
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <main class="auth-page">
    <section class="auth-panel" :aria-label="setupMode ? '账号设置' : '登录'">
      <div class="auth-brand">
        <span class="brand-mark" aria-hidden="true"><span /><span /><span /></span>
        <span class="brand-copy">
          <strong>课迹</strong>
        </span>
      </div>

      <form novalidate @submit.prevent="submit">
        <label class="form-field">
          <span>姓名</span>
          <div class="input-with-icon">
            <UserRound :size="17" aria-hidden="true" />
            <input
              v-model="name"
              type="text"
              autocomplete="username"
              :autofocus="auth.initialized"
              :disabled="loading"
            />
          </div>
        </label>

        <label class="form-field">
          <span>密码</span>
          <div class="input-with-icon has-action">
            <LockKeyhole :size="17" aria-hidden="true" />
            <input
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              :autocomplete="setupMode ? 'new-password' : 'current-password'"
              placeholder="至少 8 位"
              :disabled="loading"
            />
            <button
              type="button"
              class="icon-button password-toggle"
              :aria-label="showPassword ? '隐藏密码' : '显示密码'"
              @click="showPassword = !showPassword"
            >
              <EyeOff v-if="showPassword" :size="17" aria-hidden="true" />
              <Eye v-else :size="17" aria-hidden="true" />
            </button>
          </div>
        </label>

        <button class="button button-primary auth-submit" type="submit" :disabled="loading">
          <span v-if="loading" class="spinner light" />
          <template v-else>
            {{ submitLabel }}
            <ArrowRight :size="17" aria-hidden="true" />
          </template>
        </button>
      </form>

      <button class="auth-switch" type="button" @click="setupMode = !setupMode">
        {{ setupMode ? "去登录" : "创建账号" }}
      </button>
    </section>
  </main>
</template>
