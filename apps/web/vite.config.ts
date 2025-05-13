import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 8081,
    // Разрешаем доступ из ngrok-хоста для Telegram WebApp
    allowedHosts: ['2664-212-111-88-56.ngrok-free.app'],
    // Опционально, если нужно разрешить все хосты
    // allowedHosts: 'all',
    proxy: {
        // Перенаправляем /api/* на localhost:8080
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true,
          secure: false,
        },
    },
  },
});
