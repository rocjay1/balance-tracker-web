# 🎨 Balance Tracker Frontend

A modern dashboard interface for the Balance Tracker application, designed to manage financial balances, monitor credit utilization, and handle automated transaction imports.

---

## 🚀 Tech Stack

- **Framework**: [React 19](https://react.dev/)
- **Build Tool**: [Vite](https://vitejs.dev/)
- **Navigation**: [React Router v7](https://reactrouter.com/)
- **Styling**: [Tailwind CSS v4](https://tailwindcss.com/)
- **Language**: [TypeScript](https://www.typescriptlang.org/)

---

## 🛠️ Getting Started

To run the frontend locally during development, navigate to the `frontend/` directory and perform the following steps:

1. **Install Dependencies**:

   ```bash
   npm install
   ```

2. **Run Dev Server**:

   ```bash
   npm run dev
   ```

   The application will be available at `http://localhost:5173`.

> [!NOTE]
> For common dev tasks, you can also use the top-level `Makefile` with targets like `make install-frontend` and `make dev-frontend`.

---

## ⚙️ Development Proxy

To simplify local development and avoid CORS issues, this frontend is configured with a **Vite proxy** (in `vite.config.ts`) that forwards API requests:

- **Local Path**: `/api/*`
- **Forwards to**: `http://127.0.0.1:8080` (Default Backend Address)

Ensure your **Go Backend** is running and listening on port `8080` for the dashboard to successfully pull information.

---

## 🚢 Production & Deployment

This directory includes a `Dockerfile` for containerization.

The image is automatically built and pushed to the **GitHub Container Registry (GHCR)** via the repository's `.github/workflows/deploy.yml` pipeline on pushes to the `main` branch. It is highly optimized for deployment on lightweight hardware (ARM64 architecture).
