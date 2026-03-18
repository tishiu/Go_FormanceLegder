# Formance Ledger

A high-performance, multi-tenant ledger service built with Go and React. Formance Ledger provides a robust system for tracking financial transactions, managing accounts, and maintaining accurate balances with double-entry accounting principles.

<img width="3006" height="1834" alt="image" src="https://github.com/user-attachments/assets/dedaca88-a04d-42ee-ae82-2dbf43f739a3" />



## System Architecture

```mermaid
graph TB
    subgraph "Client Layer"
        Dashboard[React + Vite Dashboard]
    end

    subgraph "API Service (cmd/api)"
        API[net/http Server :8080]
        DashboardHandlers["/api/auth, /api/ledgers, /api/ledgers/api-keys"]
        LedgerHandlers["/v1/transactions, /v1/accounts, /v1/events, /v1/balance, /v1/webhook-*"]
        Guard[dashboardauth.Guard<br/>Cookie JWT + org ownership checks]
        APIKeyMW[auth.Middleware<br/>Bearer API key auth]
        LedgerService[ledger.Service<br/>event append + webhook job enqueue]
        LedgerRead[ledger.SQLLedgerRead<br/>read-side queries]
    end

    subgraph "Worker Service (cmd/worker)"
        RiverWorker[River Worker<br/>webhook.Worker]
        Projector[projector.Projector<br/>poll events -> update read models]
    end

    subgraph "PostgreSQL"
        subgraph "IAM Tables"
            IAM[(users, organizations, org_users, projects, ledgers, api_keys)]
        end

        subgraph "Event Store"
            Events[(events<br/>Source of Truth)]
        end

        subgraph "Read Models"
            Accounts[(accounts)]
            Transactions[(transactions)]
            Postings[(postings)]
            Offsets[(projector_offsets)]
        end

        subgraph "Webhook + Queue"
            WebhookEP[(webhook_endpoints)]
            WebhookDel[(webhook_deliveries)]
            RiverJobs[(river_job)]
        end
    end

    subgraph "External"
        CustomerWebhook[Customer Webhook Endpoints]
    end

    Dashboard -->|HTTPS| API
    API --> DashboardHandlers
    API --> LedgerHandlers

    DashboardHandlers --> Guard
    LedgerHandlers --> APIKeyMW

    LedgerHandlers --> LedgerService
    LedgerHandlers --> LedgerRead
    LedgerService -->|INSERT| Events
    LedgerService -->|INSERT| RiverJobs
    LedgerRead -->|SELECT| Accounts
    LedgerRead -->|SELECT| Transactions
    LedgerRead -->|SELECT| Postings
    DashboardHandlers -->|SELECT/INSERT/UPDATE| IAM

    RiverWorker -->|POLL| RiverJobs
    RiverWorker -->|READ| Events
    RiverWorker -->|READ| WebhookEP
    RiverWorker -->|INSERT| WebhookDel
    RiverWorker -->|HTTPS POST (signed)| CustomerWebhook

    Projector -->|POLL| Events
    Projector -->|Update| Accounts
    Projector -->|Update| Transactions
    Projector -->|Update| Postings
    Projector -->|Checkpoint| Offsets

    style Events fill:#ff9999
    style Projector fill:#99ccff
    style RiverWorker fill:#99ccff
```

## Tech Stack

### Backend
- **Language:** Go (Golang) 1.25 (as configured in Dockerfiles)
- **HTTP Server:** Standard library `net/http` (`http.ServeMux`)
- **Database:** PostgreSQL 16
- **DB Driver/Pool:** `pgx/v5` + `pgxpool`
- **Queue/Jobs:** River (`github.com/riverqueue/river`)
- **Migrations:** custom SQL runner (`cmd/migrate`) + River migrations (`rivermigrate`)

### Frontend
- **Framework:** React 18
- **Build Tool:** Vite
- **Language:** TypeScript
- **Styling:** Tailwind CSS
- **State Management:** React Query (TanStack Query)
- **Icons:** Lucide React
- **HTTP Client:** Axios

### Infrastructure
- **Containerization:** Docker & Docker Compose
- **Tooling:** Make (optional), Air (live reload)

## Prerequisites

- **Go:** Version 1.22 or higher
- **Node.js:** Version 18 or higher
- **PostgreSQL:** Version 16 (or use Docker)
- **Git**

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/tishiu/Go_FormanceLegder.git
cd Go_FormanceLegder
```

### 2. Database Setup

You can set up the database using Docker or a local PostgreSQL installation.

**Using Docker (Recommended):**

```bash
docker-compose up -d postgres
```

**Using Local PostgreSQL:**
Ensure PostgreSQL is running and create a database named `ledger_kiro`.

### 3. Backend Setup

1.  **Configure Environment Variables:**
    Copy the example environment file:
    ```bash
    cp .env.example .env
    ```
    Update `.env` with your database credentials if different from defaults.

2.  **Run Migrations:**
    ```bash
    go run cmd/migrate/main.go
    ```

3.  **Start the API Server:**
    ```bash
    # Run directly
    go run cmd/api/main.go

    # OR with Air (for live reload)
    air
    ```
    The API will be available at `http://localhost:8080`.

4.  **Start the Worker:**
    Required for event projection and webhook delivery:
    ```bash
    go run cmd/worker/main.go
    ```

### 4. Frontend Setup

1.  **Navigate to the web directory:**
    ```bash
    cd web
    ```

2.  **Install Dependencies:**
    ```bash
    npm install
    ```

3.  **Start Development Server:**
    ```bash
    npm run dev
    ```
    The frontend will be available at `http://localhost:5173`.

## Project Structure

```
├── cmd/
│   ├── api/            # API server entry point
│   ├── migrate/        # Database migration tool
│   ├── worker/         # Background worker entry point (projector + River workers)
│   └── test-runner/    # Integration test runner container entry point
├── internal/
│   ├── auth/           # JWT, API key auth, password helpers
│   ├── config/         # Configuration loading
│   ├── dashboard/      # Dashboard handlers (/api/*)
│   ├── dashboardauth/  # Dashboard authorization guard
│   ├── db/             # Database connection pool
│   ├── events/         # Event store boundary + PGX implementation
│   ├── ledger/         # Ledger command/read handlers + services
│   ├── projector/      # Event projector to read models
│   └── webhook/        # Webhook delivery engine + River worker
├── migrations/         # SQL migration files
├── web/                # React frontend application
│   ├── src/
│   │   ├── api/        # HTTP client + endpoint wrappers
│   │   ├── components/ # Reusable UI components
│   │   ├── domain/     # Backend port abstraction
│   │   ├── features/   # Feature pages (Auth, Dashboard)
│   │   └── hooks/      # Custom React hooks
│   └── public/         # Static assets
└── .env                # Environment variables (gitignored)
```

## Features

- **Multi-tenancy:** Support for multiple ledgers (Books).
- **Double-Entry Accounting:** Ensures books always balance.
- **Account Management:** Create and manage asset, liability, equity, income, and expense accounts.
- **Transaction Recording:** Post complex transactions with multiple postings.
- **Real-time Dashboard:** View transaction volumes, recent activity, and system status.
- **Dark Mode:** Fully supported UI with theme toggling.
- **Responsive Design:** Optimized for desktop and mobile devices.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
