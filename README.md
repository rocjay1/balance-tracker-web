# ⚖️ Balance Tracker Web

A modern, full-stack web application for tracking financial balances. Built with a Go backend and a React frontend, designed for efficiency and simplicity.

---

## 🏗️ Project Structure

This repository is organized as a monorepo containing both the backend and frontend components.

- **`backend/`**: A Go application using SQLite for data persistence. It handles API requests, data management, and business logic.
- **`frontend/`**: A React application built with TypeScript, Vite, and Tailwind CSS (v4). It provides a responsive and intuitive user interface.
- **`docs/`**: Project documentation, including the [Raspberry Pi Deployment Strategy](./docs/pi_management_strategy.md).

---

## 🚀 Tech Stack

### Frontend

- **Framework**: [React 19](https://react.dev/)
- **Build Tool**: [Vite](https://vitejs.dev/)
- **Language**: [TypeScript](https://www.typescriptlang.org/)
- **Styling**: [Tailwind CSS v4](https://tailwindcss.com/)

### Backend

- **Language**: [Go (Golang)](https://go.dev/)
- **Database**: [SQLite](https://www.sqlite.org/) (embedded)
- **Configuration**: YAML-based

---

## 🛠️ Getting Started

The project includes a `Makefile` to simplify common development tasks.

### Frontend Development

1. **Install Dependencies**:

   ```bash
   make install-frontend
   ```

2. **Run Dev Server**:

   ```bash
   make dev-frontend
   ```

   The application will be available at `http://localhost:5173`.

### Backend Development

1. **Run the Server**:

   ```bash
   make run-backend
   ```

   The backend server will start using the configuration in `backend/config.yaml`.

---

## 🚢 Deployment & CI/CD

This project follows a **GitOps** pattern, separating application code from infrastructure configuration.

1. **CI/CD**: GitHub Actions builds and pushes Docker images to the **GitHub Container Registry (GHCR)**.
2. **Infrastructure**: Deployment is managed via the [rocjay1-infrastructure](https://github.com/rocjay1/rocjay1-infrastructure) repository.
3. **Hardware**: Currently deployed on a Raspberry Pi (linux/arm64) using Docker Compose and **Watchtower** for automated updates.

To manually build and push images (requires appropriate permissions):

```bash
./build_and_push.sh
```

---

## 📝 License

This project is private and intended for personal use unless otherwise specified.
