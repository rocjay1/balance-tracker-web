---
description: Update Go module dependencies and keep go.mod/go.sum current
---

This workflow automates the process of keeping the backend Go module dependencies up to date.

1. Navigate to the backend directory and update all dependencies to their latest minor or patch versions:

```bash
cd backend && go get -u ./...
```

1. Clean up and synchronize `go.mod` and `go.sum` with the actual source code:

```bash
cd backend && go mod tidy
```
