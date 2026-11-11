import { createRouter, createWebHashHistory } from "vue-router";
import AppLayout from "./components/AppLayout.vue";
import LoginView from "./views/LoginView.vue";
import DashboardView from "./views/DashboardView.vue";
import StudentsView from "./views/StudentsView.vue";
import { useAuthStore } from "./stores/auth";

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: "/login",
      name: "login",
      component: LoginView,
      meta: { public: true, title: "登录" },
    },
    {
      path: "/",
      component: AppLayout,
      children: [
        {
          path: "",
          name: "dashboard",
          component: DashboardView,
          meta: { title: "课程" },
        },
        {
          path: "students",
          name: "students",
          component: StudentsView,
          meta: { title: "学生" },
        },
      ],
    },
    { path: "/:pathMatch(.*)*", redirect: "/" },
  ],
});

router.beforeEach(async (to) => {
  const auth = useAuthStore();
  if (!auth.ready) await auth.loadStatus();
  if (to.meta.public) {
    if (auth.authenticated && to.name === "login") return { name: "dashboard" };
    return true;
  }
  if (!auth.authenticated) {
    return { name: "login" };
  }
  return true;
});

export default router;
