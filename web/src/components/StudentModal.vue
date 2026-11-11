<script setup lang="ts">
import { reactive, ref, watch } from "vue";
import { Check } from "lucide-vue-next";
import { api } from "../api";
import { useUiStore } from "../stores/ui";
import type { Student } from "../types";
import BaseModal from "./BaseModal.vue";

const props = defineProps<{
  open: boolean;
  student?: Student | null;
}>();
const emit = defineEmits<{ close: []; saved: [] }>();
const ui = useUiStore();
const form = reactive<Omit<Student, "id">>({
  name: "",
});
const saving = ref(false);
const nameInvalid = ref(false);

watch(
  () => [props.open, props.student] as const,
  () => {
    if (!props.open) return;
    nameInvalid.value = false;
    if (props.student) {
      form.name = props.student.name;
    } else {
      form.name = "";
    }
  },
  { immediate: true },
);

async function save() {
  if (!form.name.trim()) {
    nameInvalid.value = true;
    ui.notify("请输入学生姓名", "error");
    return;
  }
  nameInvalid.value = false;
  saving.value = true;
  try {
    if (props.student) {
      await api.updateStudent(props.student.id, { ...form });
      ui.notify("已更新");
    } else {
      await api.createStudent({ ...form });
      ui.notify("已添加");
    }
    emit("saved");
  } catch (caught) {
    ui.notify(
      caught instanceof Error ? caught.message : "保存学生资料失败",
      "error",
    );
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <BaseModal
    :open="open"
    :title="student ? '编辑学生' : '添加学生'"
    class-name="student-modal"
    @close="emit('close')"
  >
    <form class="student-form" novalidate @submit.prevent="save">
      <label class="form-field" :class="{ 'has-error': nameInvalid }">
        <span>学生姓名 <i>*</i></span>
        <input
          v-model="form.name"
          autocomplete="off"
          :aria-invalid="nameInvalid"
          @input="nameInvalid = false"
        />
      </label>
      <div class="modal-actions modal-actions-right">
        <button type="button" class="button button-secondary" @click="emit('close')">取消</button>
        <button type="submit" class="button button-primary" :disabled="saving">
          <span v-if="saving" class="spinner light" />
          <Check v-else :size="17" aria-hidden="true" />
          {{ saving ? "保存中" : "保存" }}
        </button>
      </div>
    </form>
  </BaseModal>
</template>
