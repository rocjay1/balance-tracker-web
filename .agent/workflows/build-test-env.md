---
description: Build and serve a local test environment for the application
---

This workflow automates the process of building the backend and frontend, and serving them for local validation.

1. Ensure all dependencies are installed:

// turbo

```bash
make install-frontend
```

1. Build the backend and frontend for production:

// turbo

```bash
make build-frontend && make build-backend
```

1. Start the backend server:

// turbo

```bash
make run-backend
```

1. Start the frontend in preview mode:

// turbo

```bash
cd frontend && npm run preview -- --port 5173 --host < /dev/null
```

1. Once both servers are running, validate the application by navigating to [http://localhost:5173](http://localhost:5173) in the agent browser.
