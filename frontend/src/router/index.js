import { createRouter, createWebHashHistory } from 'vue-router'
import GalleryView from '../views/GalleryView.vue'
import RecommendedView from '../views/RecommendedView.vue'
import SettingsView from '../views/SettingsView.vue'

export default createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'gallery', component: GalleryView },
    { path: '/recommended', name: 'recommended', component: RecommendedView },
    { path: '/settings', name: 'settings', component: SettingsView },
  ],
})
