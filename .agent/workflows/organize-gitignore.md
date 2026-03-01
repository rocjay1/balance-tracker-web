---
description: Organize .gitignore with consistent sections and sorting
---

1. Read the current `.gitignore` file.
2. Group entries into the following sections:

   - # OS & System

   - # Dependencies

   - # Environment & Config

   - # Build & Distributions

   - # Logs

   - # Project Specific

3. Sort entries alphabetically within each section.
4. Ensure `!.env.example` remains explicitly after `.env` or `.env.*`.
5. Remove any duplicate entries.
6. Commit the changes with `chore: organize .gitignore`.
