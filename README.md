# ⚖️ Balance Tracker Web

A modern, full-stack web application for tracking financial balances and maintaining target credit utilization. Built with a Go backend and a React frontend, designed for efficiency, simplicity, and automated financial management.

---

## ✨ Key Features

- **🎯 Target Utilization Tracking**: Automatically calculates the exact payment needed to reach a target credit utilization (defaulting to 10%).
- **📬 Automated Email Alerts**: Daily background checks (via a built-in scheduler) send email reminders when payments are due.
- **📥 Intelligent CSV Import**: Seamlessly import transactions from financial institutions with automatic deduplication using SHA256 hashing.
- **📊 Real-time Dashboard**: A responsive React interface for monitoring balances, statement periods, and upcoming deadlines.
- **🏠 Self-Hosted Optimized**: Designed for lightweight deployment on low-power hardware like Raspberry Pi.

---

## 🏗️ Project Structure

This repository is organized as a monorepo:

- **`backend/`**: A Go application using SQLite for data persistence. It handles API requests, background scheduling, and complex financial calculations.
- **`frontend/`**: A React application built with TypeScript, Vite, and Tailwind CSS (v4). It provides a high-performance, intuitive user interface.
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
- **Database**: [SQLite](https://www.sqlite.org/) (via [sqlc](https://sqlc.dev/))
- **Scheduling**: Native Go-based daily job scheduler
- **Mailing**: SMTP-based notification system

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

   The backend server starts a web server and a background alert scheduler using the configuration in `backend/config.yaml`.

---

## 🚢 Deployment & CI/CD

This project follows a **GitOps** pattern, separating application code from infrastructure configuration.

1. **CI/CD**: GitHub Actions builds and pushes multi-arch Docker images to the **GitHub Container Registry (GHCR)**.
2. **Infrastructure**: Deployment is managed via the [rocjay1-infrastructure](https://github.com/rocjay1/rocjay1-infrastructure) repository.
3. **Automated Updates**: Integrated with **Watchtower** for seamless container updates on the target host.

To manually build and push images (requires appropriate permissions):

```bash
./build_and_push.sh
```

---

## 📝 License

This project is private and intended for personal use unless otherwise specified.
