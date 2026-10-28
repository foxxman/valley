import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const port = Number(env.VITE_DEV_PORT || 5173);
  const proxy = {
    '/api': {
      target: env.VITE_API_PROXY || 'http://127.0.0.1:8080',
      changeOrigin: true,
      ws: true,
    },
  };

  return {
    esbuild: {
      jsx: 'automatic',
    },
    server: {
      port,
      strictPort: true,
      proxy,
    },
    preview: {
      port: Number(env.VITE_PREVIEW_PORT || 4173),
      strictPort: true,
      proxy,
    },
    build: {
      target: 'es2022',
      sourcemap: true,
    },
  };
});
