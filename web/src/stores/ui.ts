import { defineStore } from "pinia";

let timer: number | undefined;

export type ToastTone = "success" | "error";

export interface ToastMessage {
  message: string;
  tone: ToastTone;
}

export const useUiStore = defineStore("ui", {
  state: () => ({
    toast: null as ToastMessage | null,
  }),
  actions: {
    notify(message: string, tone: ToastTone = "success") {
      this.toast = { message, tone };
      window.clearTimeout(timer);
      timer = window.setTimeout(() => {
        this.toast = null;
      }, tone === "error" ? 3000 : 3200);
    },
    clear() {
      this.toast = null;
      window.clearTimeout(timer);
    },
  },
});
