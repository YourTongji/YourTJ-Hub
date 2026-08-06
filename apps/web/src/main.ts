import { createApp } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import App from "./App.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", component: () => import("./pages/Home.vue") },
    // M4 起挂载：/p/post/:id、/u/:userId、/c/:slug/:id 等
  ],
});

createApp(App).use(router).mount("#app");
