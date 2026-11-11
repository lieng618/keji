import { defineStore } from "pinia";
import { api } from "../api";
import type { User } from "../types";

export const useAuthStore = defineStore("auth", {
  state: () => ({
    user: null as User | null,
    initialized: false,
    authenticated: false,
    ready: false,
    lastName: localStorage.getItem("drum-last-name") ?? "",
  }),
  actions: {
    async loadStatus() {
      try {
        const status = await api.authStatus();
        this.initialized = status.initialized;
        this.authenticated = status.authenticated;
        this.user = status.user ?? null;
      } catch {
        this.initialized = false;
        this.authenticated = false;
        this.user = null;
      } finally {
        this.ready = true;
      }
    },
    async login(name: string, password: string) {
      const result = await api.login({ name, password });
      this.user = result.user;
      this.authenticated = true;
      this.initialized = true;
      this.lastName = result.user.name;
      localStorage.setItem("drum-last-name", result.user.name);
    },
    async setup(payload: {
      name: string;
      password: string;
    }) {
      const result = await api.setup(payload);
      this.user = result.user;
      this.authenticated = true;
      this.initialized = true;
      this.lastName = result.user.name;
      localStorage.setItem("drum-last-name", result.user.name);
    },
    async logout() {
      await api.logout();
      this.user = null;
      this.authenticated = false;
    },
  },
});
