/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import react from '@vitejs/plugin-react';
import { defineConfig, transformWithEsbuild, loadEnv } from 'vite';
import pkg from '@douyinfe/vite-plugin-semi';
import path from 'path';
import { fileURLToPath } from 'url';
import { codeInspectorPlugin } from 'code-inspector-plugin';
const { vitePluginSemi } = pkg;

/** 本配置文件所在目录（ESM 下无全局 __dirname） */
const __dirname = path.dirname(fileURLToPath(import.meta.url));

/**
 * 解析 Vite 开发代理指向的后端地址（与 Go 网关同一主机端口）。
 * 可在 web/.env.development.local 中设置 VITE_DEV_PROXY_TARGET（例如 http://192.168.0.169:3000）。
 * @param {string} mode Vite 运行模式
 * @returns {string} 代理 target，例如 http://127.0.0.1:3000
 */
function resolveDevProxyTarget(mode) {
  const env = loadEnv(mode, __dirname, '');
  const raw = (env.VITE_DEV_PROXY_TARGET || '').trim();
  if (raw) {
    return raw.replace(/\/$/, '');
  }
  return 'http://127.0.0.1:3000';
}

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const devProxyTarget = resolveDevProxyTarget(mode);

  return {
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      // semi-ui 的 package exports 未声明 dist/css 子路径，Vite/Rollup 会报 Missing specifier；
      // 仍指向包内同一份 semi.css，不增加或替换样式内容。
      '@douyinfe/semi-ui/dist/css/semi.css': path.resolve(
        __dirname,
        'node_modules/@douyinfe/semi-ui/dist/css/semi.css',
      ),
    },
  },
  plugins: [
    codeInspectorPlugin({
      bundler: 'vite',
    }),
    {
      name: 'treat-js-files-as-jsx',
      async transform(code, id) {
        if (!/src\/.*\.js$/.test(id)) {
          return null;
        }

        // Use the exposed transform from vite, instead of directly
        // transforming with esbuild
        return transformWithEsbuild(code, id, {
          loader: 'jsx',
          jsx: 'automatic',
        });
      },
    },
    react(),
    vitePluginSemi({
      cssLayer: true,
    }),
  ],
  optimizeDeps: {
    force: true,
    esbuildOptions: {
      loader: {
        '.js': 'jsx',
        '.json': 'json',
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'react-core': ['react', 'react-dom', 'react-router-dom'],
          'semi-ui': ['@douyinfe/semi-icons', '@douyinfe/semi-ui'],
          tools: ['axios', 'history', 'marked'],
          'react-components': [
            'react-dropzone',
            'react-fireworks',
            'react-telegram-login',
            'react-toastify',
            'react-turnstile',
          ],
          i18n: [
            'i18next',
            'react-i18next',
            'i18next-browser-languagedetector',
          ],
        },
      },
    },
  },
  server: {
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: devProxyTarget,
        changeOrigin: true,
      },
      '/mj': {
        target: devProxyTarget,
        changeOrigin: true,
      },
      '/pg': {
        target: devProxyTarget,
        changeOrigin: true,
      },
    },
  },
  };
});
