# Raspberry Pi Configuration & Deployment Strategy

This document summarizes the management and deployment strategy for the Balance Tracker setup, which has evolved into a fully declarative **Infrastructure-as-Code (IaC)** stack using Terraform and Ansible.

## 1. Core Philosophy: Declarative Lifecycle

The system is designed to be entirely reproducible, moving away from manual configuration to a dual-layer IaC approach:

- **Provisioning (Cloud)**: Managed by **Terraform**.
- **Configuration & Deployment (Host)**: Managed by **Ansible**.
- **Container Orchestration**: Standardized with **Docker Compose**.

---

## 2. Infrastructure as Code (the "GitOps" Pattern)

The "Source of Truth" is split specifically to separate concerns:

### Repository Split

1. **`balance-tracker-web` (Application Repo)**:
    - Contains the Source Code (Go backend, React frontend).
    - Responsible for building and pushing multi-arch (linux/arm64) images to **GitHub Container Registry (GHCR)**.
    - Uses GitHub Actions for automated CI/CD.

2. **`rocjay1-infrastructure` (Management Repo)**:
    - The "Orchestrator" for the entire ecosystem.
    - **Terraform**: Provisions Cloudflare Tunnels, DNS records, and Azure AD (Entra ID) for secure access.
    - **Ansible**: Handles host hardening (UFW, SSH), Docker installation, and the deployment of application stacks.

---

## 3. Deployment Stack

The deployment lifecycle is managed via Ansible playbooks located in `balance-tracker/ansible/`.

### Core Services

- **`frontend`**: React 19 application served via Nginx.
- **`backend`**: Go service with SQLite persistence.
- **`tunnel`**: `cloudflared` agent for secure ingress without open ports.
- **`watchtower`**: Automated image updates from GHCR with restricted label-based scope.

### 📊 Observability & Monitoring

We implement a full observability stack for real-time health monitoring:

- **Prometheus**: Metric collection and storage.
- **Grafana**: Vizualization dashboards for system and application health.
- **Loki & Promtail**: Centralized log aggregation and distribution.

---

## 4. Security & Maintenance

### Secret Management

- **Ansible Vault**: All sensitive variables (Tunnel tokens, GHCR credentials, SMTP passwords) are encrypted using Ansible Vault.
- **No Plaintext**: Secrets never touch the disk in plaintext on the management machine or in Git.

### Host Hardening

Ansible automatically ensures:

- **Firewall (UFW)**: Default deny-all policy with specific exceptions.
- **SSH Hardening**: Disabled password authentication and root login.
- **Unattended Upgrades**: Automatic security patching for the underlying OS.

---

## 5. Usage Summary

### To Update Code

Push to `main`. GitHub Actions builds the images. **Watchtower** detects the change on the Pi and restarts the containers automatically within 5 minutes.

### To Update Infrastructure/Config

1. Edit the relevant Terraform (`.tf`) or Ansible (`.yml`) files.
2. For Cloud changes: `terraform apply`.
3. For Host/App changes:

   ```bash
   ansible-playbook -i hosts.ini deploy_balance_tracker.yml --vault-password-file .vault_pass
   ```

---

## 6. Maintenance Tasks

Backups are orchestrated via Ansible, capturing the SQLite `finance.db` and configuration state regularly, ensuring disaster recovery capability for the persistent data.
