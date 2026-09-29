# Terminus — web

Next.js (App Router) client for the Terminus API (Go, `../backend`).

```bash
pnpm install
pnpm dev            # http://localhost:3210, proxies /api/* to BACKEND_URL
pnpm build && pnpm start
pnpm lint
pnpm test:e2e       # Playwright; needs the Go server on BACKEND_URL
```

`BACKEND_URL` defaults to `http://127.0.0.1:8080`.
