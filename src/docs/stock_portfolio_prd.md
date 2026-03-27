# Product Requirements Document (PRD)
## Stock Portfolio Manager (Engineer Edition)

Version: 1.0  
Date: 2026-03-05  
Author: Stefan

---

# 1. Overview

The goal of this project is to build a **local-first stock portfolio management application** designed for engineers.

The system allows users to track stock holdings, perform CRUD operations on transactions, and automatically fetch external market data.

Architecture is fully **containerized**.

Core technologies:

- Frontend: React + Next.js
- Backend API: Go
- Data Fetch Service: Python
- Database: PostgreSQL
- Containerization: Docker / Docker Compose
- Deployment: Local environment

---

# 2. Objectives

The system should:

1. Allow CRUD operations on stock transactions.
2. Persist data in a local PostgreSQL database.
3. Fetch external market prices via a Python service.
4. Run entirely via Docker containers.
5. Provide a clean web UI.

---

# 3. Non‑Goals (MVP)

Excluded from the first version:

- Multi‑user support
- Authentication
- Options trading
- Real‑time streaming prices
- Complex analytics

---

# 4. High Level Architecture

Frontend Container (Next.js)

→ Backend API Container (Go)

→ PostgreSQL Database Container

External data flow:

Backend → Python Data Service → External Market APIs

---

# 5. System Components

## Frontend

Technology: React + Next.js

Responsibilities:

- Display portfolio
- Forms for adding transactions
- Editing transactions
- Deleting transactions
- Displaying stock prices

Key views:

- Portfolio overview
- Add transaction
- Edit transaction
- Transaction history

---

## Backend API

Technology: Go

Responsibilities:

- REST API
- Business logic
- Portfolio aggregation
- Database interaction
- Communication with Python service

---

## Data Fetch Service

Technology: Python

Responsibilities:

- Fetch stock prices
- Normalize financial data
- Provide a simple API to the Go backend

Suggested libraries:

- yfinance
- requests
- pandas

Example endpoints:

GET /price/{ticker}

GET /historical/{ticker}

---

## Database

Technology: PostgreSQL

Responsibilities:

- Store transactions
- Store stock metadata
- Store cached prices

Runs inside Docker container.

---

# 6. Data Model

## stocks

| column | type | description |
|------|------|-------------|
| id | uuid | primary key |
| ticker | text | stock symbol |
| name | text | company name |
| created_at | timestamp | creation timestamp |

---

## transactions

| column | type | description |
|------|------|-------------|
| id | uuid | primary key |
| ticker | text | stock symbol |
| transaction_type | text | buy or sell |
| shares | numeric | number of shares |
| price | numeric | transaction price |
| transaction_date | date | trade date |

---

## prices_cache

| column | type | description |
|------|------|-------------|
| ticker | text | stock symbol |
| price | numeric | latest price |
| fetched_at | timestamp | fetch timestamp |

---

# 7. Core Features

## Add Transaction

User enters:

- ticker
- number of shares
- price
- transaction date

System stores record in database.

---

## Edit Transaction

User updates an existing transaction.

---

## Delete Transaction

User removes a transaction.

---

## Portfolio View

System aggregates:

- total shares
- average cost basis
- current market value

---

## Market Price Fetching

Python service retrieves prices.

Backend caches them.

---

# 8. API Design

Transactions

POST /transactions

GET /transactions

PUT /transactions/{id}

DELETE /transactions/{id}

---

Portfolio

GET /portfolio

---

Prices

GET /price/{ticker}

---

# 9. Container Architecture

Containers:

frontend  
backend  
python-data-service  
postgres

All containers communicate through a Docker network.

---

# 10. Local Development

Run system with:

docker-compose up

Services:

Frontend: http://localhost:3000

Backend: http://localhost:8080

---

# 11. Future Extensions

Potential improvements:

- ETF support
- Crypto assets
- Portfolio analytics
- Risk simulation
- Scenario testing
- Cloud deployment

---

# 12. Acceptance Criteria

MVP is complete when:

- User can add transactions
- Transactions persist in PostgreSQL
- Portfolio aggregates correctly
- Prices can be fetched
- Entire system runs with docker-compose

---

# 13. Success Metrics

- System starts with one command
- CRUD operations function reliably
- Portfolio loads in < 1 second
- Price fetch < 2 seconds

---

# 14. Repository Structure

project-root/

frontend/

backend/

data-service/

database/

docker-compose.yml

README.md
