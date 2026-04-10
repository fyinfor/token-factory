// vite.config.js
import react from "file:///D:/install/GoProjects/token-factory/web/node_modules/@vitejs/plugin-react/dist/index.mjs";
import { defineConfig, transformWithEsbuild } from "file:///D:/install/GoProjects/token-factory/web/node_modules/vite/dist/node/index.js";
import pkg from "file:///D:/install/GoProjects/token-factory/web/node_modules/@douyinfe/vite-plugin-semi/lib/index.js";
import path from "path";
import { codeInspectorPlugin } from "file:///D:/install/GoProjects/token-factory/web/node_modules/code-inspector-plugin/dist/index.mjs";
var __vite_injected_original_dirname = "D:\\install\\GoProjects\\token-factory\\web";
var { vitePluginSemi } = pkg;
var vite_config_default = defineConfig({
  resolve: {
    alias: {
      "@": path.resolve(__vite_injected_original_dirname, "./src")
    }
  },
  plugins: [
    codeInspectorPlugin({
      bundler: "vite"
    }),
    {
      name: "treat-js-files-as-jsx",
      async transform(code, id) {
        if (!/src\/.*\.js$/.test(id)) {
          return null;
        }
        return transformWithEsbuild(code, id, {
          loader: "jsx",
          jsx: "automatic"
        });
      }
    },
    react(),
    vitePluginSemi({
      cssLayer: true
    })
  ],
  optimizeDeps: {
    force: true,
    esbuildOptions: {
      loader: {
        ".js": "jsx",
        ".json": "json"
      }
    }
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          "react-core": ["react", "react-dom", "react-router-dom"],
          "semi-ui": ["@douyinfe/semi-icons", "@douyinfe/semi-ui"],
          tools: ["axios", "history", "marked"],
          "react-components": [
            "react-dropzone",
            "react-fireworks",
            "react-telegram-login",
            "react-toastify",
            "react-turnstile"
          ],
          i18n: [
            "i18next",
            "react-i18next",
            "i18next-browser-languagedetector"
          ]
        }
      }
    }
  },
  server: {
    host: "0.0.0.0",
    proxy: {
      "/api": {
        target: "http://localhost:3000",
        changeOrigin: true
      },
      "/mj": {
        target: "http://localhost:3000",
        changeOrigin: true
      },
      "/pg": {
        target: "http://localhost:3000",
        changeOrigin: true
      }
    }
  }
});
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcuanMiXSwKICAic291cmNlc0NvbnRlbnQiOiBbImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCJEOlxcXFxpbnN0YWxsXFxcXEdvUHJvamVjdHNcXFxcdG9rZW4tZmFjdG9yeVxcXFx3ZWJcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIkQ6XFxcXGluc3RhbGxcXFxcR29Qcm9qZWN0c1xcXFx0b2tlbi1mYWN0b3J5XFxcXHdlYlxcXFx2aXRlLmNvbmZpZy5qc1wiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9pbXBvcnRfbWV0YV91cmwgPSBcImZpbGU6Ly8vRDovaW5zdGFsbC9Hb1Byb2plY3RzL3Rva2VuLWZhY3Rvcnkvd2ViL3ZpdGUuY29uZmlnLmpzXCI7LypcbkNvcHlyaWdodCAoQykgMjAyNSBRdWFudHVtTm91c1xuXG5UaGlzIHByb2dyYW0gaXMgZnJlZSBzb2Z0d2FyZTogeW91IGNhbiByZWRpc3RyaWJ1dGUgaXQgYW5kL29yIG1vZGlmeVxuaXQgdW5kZXIgdGhlIHRlcm1zIG9mIHRoZSBHTlUgQWZmZXJvIEdlbmVyYWwgUHVibGljIExpY2Vuc2UgYXNcbnB1Ymxpc2hlZCBieSB0aGUgRnJlZSBTb2Z0d2FyZSBGb3VuZGF0aW9uLCBlaXRoZXIgdmVyc2lvbiAzIG9mIHRoZVxuTGljZW5zZSwgb3IgKGF0IHlvdXIgb3B0aW9uKSBhbnkgbGF0ZXIgdmVyc2lvbi5cblxuVGhpcyBwcm9ncmFtIGlzIGRpc3RyaWJ1dGVkIGluIHRoZSBob3BlIHRoYXQgaXQgd2lsbCBiZSB1c2VmdWwsXG5idXQgV0lUSE9VVCBBTlkgV0FSUkFOVFk7IHdpdGhvdXQgZXZlbiB0aGUgaW1wbGllZCB3YXJyYW50eSBvZlxuTUVSQ0hBTlRBQklMSVRZIG9yIEZJVE5FU1MgRk9SIEEgUEFSVElDVUxBUiBQVVJQT1NFLiBTZWUgdGhlXG5HTlUgQWZmZXJvIEdlbmVyYWwgUHVibGljIExpY2Vuc2UgZm9yIG1vcmUgZGV0YWlscy5cblxuWW91IHNob3VsZCBoYXZlIHJlY2VpdmVkIGEgY29weSBvZiB0aGUgR05VIEFmZmVybyBHZW5lcmFsIFB1YmxpYyBMaWNlbnNlXG5hbG9uZyB3aXRoIHRoaXMgcHJvZ3JhbS4gSWYgbm90LCBzZWUgPGh0dHBzOi8vd3d3LmdudS5vcmcvbGljZW5zZXMvPi5cblxuRm9yIGNvbW1lcmNpYWwgbGljZW5zaW5nLCBwbGVhc2UgY29udGFjdCBzdXBwb3J0QHF1YW50dW1ub3VzLmNvbVxuKi9cblxuaW1wb3J0IHJlYWN0IGZyb20gJ0B2aXRlanMvcGx1Z2luLXJlYWN0JztcbmltcG9ydCB7IGRlZmluZUNvbmZpZywgdHJhbnNmb3JtV2l0aEVzYnVpbGQgfSBmcm9tICd2aXRlJztcbmltcG9ydCBwa2cgZnJvbSAnQGRvdXlpbmZlL3ZpdGUtcGx1Z2luLXNlbWknO1xuaW1wb3J0IHBhdGggZnJvbSAncGF0aCc7XG5pbXBvcnQgeyBjb2RlSW5zcGVjdG9yUGx1Z2luIH0gZnJvbSAnY29kZS1pbnNwZWN0b3ItcGx1Z2luJztcbmNvbnN0IHsgdml0ZVBsdWdpblNlbWkgfSA9IHBrZztcblxuLy8gaHR0cHM6Ly92aXRlanMuZGV2L2NvbmZpZy9cbmV4cG9ydCBkZWZhdWx0IGRlZmluZUNvbmZpZyh7XG4gIHJlc29sdmU6IHtcbiAgICBhbGlhczoge1xuICAgICAgJ0AnOiBwYXRoLnJlc29sdmUoX19kaXJuYW1lLCAnLi9zcmMnKSxcbiAgICB9LFxuICB9LFxuICBwbHVnaW5zOiBbXG4gICAgY29kZUluc3BlY3RvclBsdWdpbih7XG4gICAgICBidW5kbGVyOiAndml0ZScsXG4gICAgfSksXG4gICAge1xuICAgICAgbmFtZTogJ3RyZWF0LWpzLWZpbGVzLWFzLWpzeCcsXG4gICAgICBhc3luYyB0cmFuc2Zvcm0oY29kZSwgaWQpIHtcbiAgICAgICAgaWYgKCEvc3JjXFwvLipcXC5qcyQvLnRlc3QoaWQpKSB7XG4gICAgICAgICAgcmV0dXJuIG51bGw7XG4gICAgICAgIH1cblxuICAgICAgICAvLyBVc2UgdGhlIGV4cG9zZWQgdHJhbnNmb3JtIGZyb20gdml0ZSwgaW5zdGVhZCBvZiBkaXJlY3RseVxuICAgICAgICAvLyB0cmFuc2Zvcm1pbmcgd2l0aCBlc2J1aWxkXG4gICAgICAgIHJldHVybiB0cmFuc2Zvcm1XaXRoRXNidWlsZChjb2RlLCBpZCwge1xuICAgICAgICAgIGxvYWRlcjogJ2pzeCcsXG4gICAgICAgICAganN4OiAnYXV0b21hdGljJyxcbiAgICAgICAgfSk7XG4gICAgICB9LFxuICAgIH0sXG4gICAgcmVhY3QoKSxcbiAgICB2aXRlUGx1Z2luU2VtaSh7XG4gICAgICBjc3NMYXllcjogdHJ1ZSxcbiAgICB9KSxcbiAgXSxcbiAgb3B0aW1pemVEZXBzOiB7XG4gICAgZm9yY2U6IHRydWUsXG4gICAgZXNidWlsZE9wdGlvbnM6IHtcbiAgICAgIGxvYWRlcjoge1xuICAgICAgICAnLmpzJzogJ2pzeCcsXG4gICAgICAgICcuanNvbic6ICdqc29uJyxcbiAgICAgIH0sXG4gICAgfSxcbiAgfSxcbiAgYnVpbGQ6IHtcbiAgICByb2xsdXBPcHRpb25zOiB7XG4gICAgICBvdXRwdXQ6IHtcbiAgICAgICAgbWFudWFsQ2h1bmtzOiB7XG4gICAgICAgICAgJ3JlYWN0LWNvcmUnOiBbJ3JlYWN0JywgJ3JlYWN0LWRvbScsICdyZWFjdC1yb3V0ZXItZG9tJ10sXG4gICAgICAgICAgJ3NlbWktdWknOiBbJ0Bkb3V5aW5mZS9zZW1pLWljb25zJywgJ0Bkb3V5aW5mZS9zZW1pLXVpJ10sXG4gICAgICAgICAgdG9vbHM6IFsnYXhpb3MnLCAnaGlzdG9yeScsICdtYXJrZWQnXSxcbiAgICAgICAgICAncmVhY3QtY29tcG9uZW50cyc6IFtcbiAgICAgICAgICAgICdyZWFjdC1kcm9wem9uZScsXG4gICAgICAgICAgICAncmVhY3QtZmlyZXdvcmtzJyxcbiAgICAgICAgICAgICdyZWFjdC10ZWxlZ3JhbS1sb2dpbicsXG4gICAgICAgICAgICAncmVhY3QtdG9hc3RpZnknLFxuICAgICAgICAgICAgJ3JlYWN0LXR1cm5zdGlsZScsXG4gICAgICAgICAgXSxcbiAgICAgICAgICBpMThuOiBbXG4gICAgICAgICAgICAnaTE4bmV4dCcsXG4gICAgICAgICAgICAncmVhY3QtaTE4bmV4dCcsXG4gICAgICAgICAgICAnaTE4bmV4dC1icm93c2VyLWxhbmd1YWdlZGV0ZWN0b3InLFxuICAgICAgICAgIF0sXG4gICAgICAgIH0sXG4gICAgICB9LFxuICAgIH0sXG4gIH0sXG4gIHNlcnZlcjoge1xuICAgIGhvc3Q6ICcwLjAuMC4wJyxcbiAgICBwcm94eToge1xuICAgICAgJy9hcGknOiB7XG4gICAgICAgIHRhcmdldDogJ2h0dHA6Ly9sb2NhbGhvc3Q6MzAwMCcsXG4gICAgICAgIGNoYW5nZU9yaWdpbjogdHJ1ZSxcbiAgICAgIH0sXG4gICAgICAnL21qJzoge1xuICAgICAgICB0YXJnZXQ6ICdodHRwOi8vbG9jYWxob3N0OjMwMDAnLFxuICAgICAgICBjaGFuZ2VPcmlnaW46IHRydWUsXG4gICAgICB9LFxuICAgICAgJy9wZyc6IHtcbiAgICAgICAgdGFyZ2V0OiAnaHR0cDovL2xvY2FsaG9zdDozMDAwJyxcbiAgICAgICAgY2hhbmdlT3JpZ2luOiB0cnVlLFxuICAgICAgfSxcbiAgICB9LFxuICB9LFxufSk7XG4iXSwKICAibWFwcGluZ3MiOiAiO0FBbUJBLE9BQU8sV0FBVztBQUNsQixTQUFTLGNBQWMsNEJBQTRCO0FBQ25ELE9BQU8sU0FBUztBQUNoQixPQUFPLFVBQVU7QUFDakIsU0FBUywyQkFBMkI7QUF2QnBDLElBQU0sbUNBQW1DO0FBd0J6QyxJQUFNLEVBQUUsZUFBZSxJQUFJO0FBRzNCLElBQU8sc0JBQVEsYUFBYTtBQUFBLEVBQzFCLFNBQVM7QUFBQSxJQUNQLE9BQU87QUFBQSxNQUNMLEtBQUssS0FBSyxRQUFRLGtDQUFXLE9BQU87QUFBQSxJQUN0QztBQUFBLEVBQ0Y7QUFBQSxFQUNBLFNBQVM7QUFBQSxJQUNQLG9CQUFvQjtBQUFBLE1BQ2xCLFNBQVM7QUFBQSxJQUNYLENBQUM7QUFBQSxJQUNEO0FBQUEsTUFDRSxNQUFNO0FBQUEsTUFDTixNQUFNLFVBQVUsTUFBTSxJQUFJO0FBQ3hCLFlBQUksQ0FBQyxlQUFlLEtBQUssRUFBRSxHQUFHO0FBQzVCLGlCQUFPO0FBQUEsUUFDVDtBQUlBLGVBQU8scUJBQXFCLE1BQU0sSUFBSTtBQUFBLFVBQ3BDLFFBQVE7QUFBQSxVQUNSLEtBQUs7QUFBQSxRQUNQLENBQUM7QUFBQSxNQUNIO0FBQUEsSUFDRjtBQUFBLElBQ0EsTUFBTTtBQUFBLElBQ04sZUFBZTtBQUFBLE1BQ2IsVUFBVTtBQUFBLElBQ1osQ0FBQztBQUFBLEVBQ0g7QUFBQSxFQUNBLGNBQWM7QUFBQSxJQUNaLE9BQU87QUFBQSxJQUNQLGdCQUFnQjtBQUFBLE1BQ2QsUUFBUTtBQUFBLFFBQ04sT0FBTztBQUFBLFFBQ1AsU0FBUztBQUFBLE1BQ1g7QUFBQSxJQUNGO0FBQUEsRUFDRjtBQUFBLEVBQ0EsT0FBTztBQUFBLElBQ0wsZUFBZTtBQUFBLE1BQ2IsUUFBUTtBQUFBLFFBQ04sY0FBYztBQUFBLFVBQ1osY0FBYyxDQUFDLFNBQVMsYUFBYSxrQkFBa0I7QUFBQSxVQUN2RCxXQUFXLENBQUMsd0JBQXdCLG1CQUFtQjtBQUFBLFVBQ3ZELE9BQU8sQ0FBQyxTQUFTLFdBQVcsUUFBUTtBQUFBLFVBQ3BDLG9CQUFvQjtBQUFBLFlBQ2xCO0FBQUEsWUFDQTtBQUFBLFlBQ0E7QUFBQSxZQUNBO0FBQUEsWUFDQTtBQUFBLFVBQ0Y7QUFBQSxVQUNBLE1BQU07QUFBQSxZQUNKO0FBQUEsWUFDQTtBQUFBLFlBQ0E7QUFBQSxVQUNGO0FBQUEsUUFDRjtBQUFBLE1BQ0Y7QUFBQSxJQUNGO0FBQUEsRUFDRjtBQUFBLEVBQ0EsUUFBUTtBQUFBLElBQ04sTUFBTTtBQUFBLElBQ04sT0FBTztBQUFBLE1BQ0wsUUFBUTtBQUFBLFFBQ04sUUFBUTtBQUFBLFFBQ1IsY0FBYztBQUFBLE1BQ2hCO0FBQUEsTUFDQSxPQUFPO0FBQUEsUUFDTCxRQUFRO0FBQUEsUUFDUixjQUFjO0FBQUEsTUFDaEI7QUFBQSxNQUNBLE9BQU87QUFBQSxRQUNMLFFBQVE7QUFBQSxRQUNSLGNBQWM7QUFBQSxNQUNoQjtBQUFBLElBQ0Y7QUFBQSxFQUNGO0FBQ0YsQ0FBQzsiLAogICJuYW1lcyI6IFtdCn0K
