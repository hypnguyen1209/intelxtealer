import { createApp } from 'vue'
import './assets/styles/global.css'
import App from './App.vue'

// Import styles
import './assets/styles/common.css'

// Import Ant Design Vue
import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'

// Create app with Ant Design Vue
const app = createApp(App)
app.use(Antd)
app.mount('#app')
