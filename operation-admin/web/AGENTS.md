# Frontend conventions for operation-admin

Stack mirrors `eagle-ops/frontend`:

- Vue 3 + TypeScript + Vite
- Element Plus + `@element-plus/icons-vue`
- Pinia + Vue Router（顶栏子系统 + 左侧菜单 + 多页签）
- axios（`/api` 代理到后端）
- Sass

## Modules

| Top tab | `meta.subsystem` | Notes |
| --- | --- | --- |
| 财务管理 | `finance` | Reconcile board etc. |
| 运维管理 | `ops` | Placeholder for future pages |
| 首页 | `common` | Always in left menu |

## Dev

```bash
cd operation-admin/web
cp .env.example .env
npm install
npm run dev
```

Vite listens on **http://localhost:3101**.
