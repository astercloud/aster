import { createRouter, createWebHistory } from 'vue-router';
import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Home',
    component: () => import('./views/Home.vue'),
  },
  {
    path: '/agent-demo',
    name: 'AgentDemo',
    component: () => import('./views/AgentChatUIDemo.vue'),
  },
  {
    path: '/docs',
    name: 'Docs',
    component: () => import('./views/ComponentDocs.vue'),
  },
  {
    path: '/docs/:component',
    name: 'ComponentDoc',
    component: () => import('./views/ComponentDocs.vue'),
  },
  {
    path: '/components',
    name: 'Components',
    component: () => import('./views/ChatUIComponents.vue'),
  },
  {
    path: '/agents',
    name: 'Agents',
    component: () => import('./views/AgentManagement.vue'),
  },
  {
    path: '/workflows',
    name: 'Workflows',
    component: () => import('./views/WorkflowManagement.vue'),
  },
  {
    path: '/rooms',
    name: 'Rooms',
    component: () => import('./views/RoomManagement.vue'),
  },
  {
    path: '/projects',
    name: 'Projects',
    component: () => import('./views/ProjectManagement.vue'),
  },
  {
    path: '/landing',
    name: 'Landing',
    component: () => import('./views/LandingPage.vue'),
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;
