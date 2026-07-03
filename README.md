# CAO Logbook — EASA Part-66

A Part-66 maintenance engineer logbook for CAMO/CAO environments.

## Features

- **Maintenance Records** — log tasks with ATA chapters, work orders, task cards, and component tracking
- **Aircraft** — expanded fields (variant, MSN, engine type, MTOW, etc.)
- **Organizations** — CAMO, CAO, Part-145, ATO, line stations
- **Certifications** — Part-66 B1.1/B1.2/B2/B3/L1-3, type ratings, company authorizations
- **Supervision Tracking** — 70% supervision rule monitoring
- **Reports** — breakdown by aircraft, ATA chapter, work type, category, organization
- **PDF Export** — EASA Part-66 logbook format
- **Auth** — email/password, Estonia e-ID (Mobile-ID & Smart-ID), OAuth (Google/GitHub), magic links
- **PDF Export** — EASA Part-66 logbook format

## Quick Start

```bash
go run .
# Opens at http://localhost:3000
```

Use the **Demo Login** button to explore with sample data.

## Database

SQLite. Default path: `./data/cao.db`. Override with `DATABASE_PATH` env var.

## Tech Stack

Go, SQLite, HTML templates, CSS (Nord/Light/System themes), Lua (Fengari for frontend), gofpdf.
