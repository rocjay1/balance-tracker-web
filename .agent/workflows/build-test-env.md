---
description: Build and serve a local test environment for the application
---

This workflow automates the process of building the backend and frontend, and serving them for local validation.

1. Ensure all dependencies are installed:

```bash
make install-frontend
```

1. Build the backend and frontend for production:

```bash
make build-frontend && make build-backend
```

1. Start the backend server:

```bash
make run-backend
```

1. Start the frontend in preview mode:

```bash
cd frontend && npm run preview -- --port 5173 --host
```

1. Once both servers are running, validate the application by navigating to [http://localhost:5173](http://localhost:5173) in the agent browser.
