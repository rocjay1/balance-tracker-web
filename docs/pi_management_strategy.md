# Raspberry Pi Configuration & Deployment Strategy

This document summarizes the recommended management and deployment strategy for the Raspberry Pi setup, as established in February 2026.

## 1. Core Philosophy: Declarative over Imperative

Instead of using ad-hoc `docker run` commands (which are hard to track and update), we use **Docker Compose**. This allows for:

* **Version Control**: Configuration is stored in Git.
* **Reproducibility**: One command (`docker compose up -d`) restores the entire environment.
* **Orchestration**: Managing multiple connected services (Backend, Frontend, Tunnel) as a single unit.

---

## 2. Infrastructure as Code (the "GitOps" Pattern)

We have separated the **Code** from the **Infrastructure**.

### Repository Split

1. **`balance-tracker-web` (Application Repo)**:
    * Contains the Source Code (Go backend, React frontend).
    * Responsible for building and pushing images to **GitHub Container Registry (GHCR)**.
    * Uses GitHub Actions to automate builds on every push to `main`.

2. **`rocjay1-infrastructure` (Management Repo)**:
    * The "Source of Truth" for the Raspberry Pi's state.
    * Contains `docker-compose.yml`, environment variables (`.env`), and application secrets (`config.yaml`).
    * Manages Cloudflare Tunnels via Terraform.

---

## 3. Automated Updates with Watchtower

To keep the Raspberry Pi up to date without manual intervention, we use **Watchtower**.

* **How it works**: Watchtower runs as a sidecar container on the Pi. It polls GHCR every 5 minutes.
* **Automatic Deployment**: When it detects a new version of your image, it pulls it, gracefully stops the old container, and starts the new one.
* **Selective Updates**: We use labels (`com.centurylinklabs.watchtower.enable=true`) to ensure Watchtower only updates specific application containers and doesn't interrupt critical infrastructure like the Cloudflare Tunnel unnecessarily.

---

## 4. Security & Best Practices

### Secret Management

* **No Secrets in Git**: Sensitive files like `.env` and `config.yaml` are added to `.gitignore`.
* **Keychain Integration**: Use the `load-cloudflare-token.sh` and `store-cloudflared-token.sh` scripts to keep Cloudflare API tokens in the macOS Keychain rather than plain text.

### Deployment Location

* **Standardized Paths**: Services are deployed to `/opt/<project-name>` instead of the home directory. This follows Linux conventions and keeps the `admin` user's home directory clean.

### OS-Level Maintenance

* **Automatic Security Patches**: Enable `unattended-upgrades` on the Raspberry Pi OS.

    ```bash
    sudo apt install unattended-upgrades
    sudo dpkg-reconfigure --priority=low unattended-upgrades
    ```

---

## 5. Usage Summary

### To Update Code

Push to GitHub or run `./build_and_push.sh`. Watchtower handles the rest on the Pi.

### To Update Infrastructure/Config

Edit files in `rocjay1-infrastructure/balance-tracker` and run `./deploy.sh`.

---

## 6. Future Evolution

While the current system uses a combination of **Shell Scripts** and **Terraform**, the long-term goal is to transition to an **Ansible + Terraform** stack:

* **Terraform**: Will continue to manage cloud infrastructure (Cloudflare, Entra ID, DNS).
* **Ansible**: Will replace the legacy `.sh` deployment scripts to manage the Raspberry Pi's internal state (OS configuration, Docker installs, and Compose deployments) in a more robust and idempotent way.
