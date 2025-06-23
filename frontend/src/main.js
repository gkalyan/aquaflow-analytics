import "./primevue-base.css";
import "./style.css";

import { createApp } from "vue";
import { createPinia } from 'pinia'
import router from './router'
import PrimeVue from "primevue/config";
import ToastService from 'primevue/toastservice'
import App from "./App.vue";
import Aura from "@primeuix/themes/aura";

const app = createApp(App);
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.use(PrimeVue, {
	theme: {
		preset: Aura,
		options: {
			darkModeSelector: ".p-dark",
		}
	},
});
app.use(ToastService)

app.mount("#app");