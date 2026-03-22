# 📚 Project Documentation

This directory contains supplemental guides, and maintenance strategies for the Balance Tracker setup.

---

## 📑 Available Documentation

| File | Description |
| :--- | :--- |
| **[`pi_management_strategy.md`](./pi_management_strategy.md)** | Outlines the Infrastructure-as-Code (Terraform & Ansible) structure, security guardrails, host configuration, and automated GitOps deployment lifecycle. |
| **[`grafana_queries.md`](./grafana_queries.md)** | Provides useful LogQL queries for Grafana to format and query structured application (Go) and server (Nginx) logs effectively. |

---

## 🔐 Infrastructure Config & Secrets

To ensure secure configuration management:

- **Application Values**: Modeled with `backend/config.yaml` as detailed in the [Main README](../README.md).
- **Environment Secrets**: Sensitive details (tokens, passwords) are encrypted with **Ansible Vault** within the separate `rocjay1-infrastructure` management repository. Plaintext secrets never touch Git.
