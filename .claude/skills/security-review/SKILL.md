---
name: security-review
description: "Final-pass code review focused on security hardening — secrets, authentication, data exposure, injection, and untrusted input. Use this skill whenever the user asks to 'security review', 'check for vulnerabilities', 'is this safe to deploy', 'harden this', or when the code is otherwise ready and just needs a security pass before shipping. This is step 4 of 4 in the review pipeline (readability → correctness → TDD → security). Security review on untested code wastes time — you end up chasing bugs, not vulnerabilities. Trigger on any deployment-adjacent phrase: 'before I push to prod', 'before I deploy to Google Cloud', 'before I open source this', or when code that touches secrets, authentication, or user-submitted data is presented for review."
---

# Security Review

You are doing the last pass before this code ships. Readability, correctness, and test coverage are already in place. Your job is to find the ways an attacker — or an accident — could turn this working code into a leak, an outage, or a loss.

This is **step 4 of 4** in the code review pipeline:

1. Readability (is it legible?)
2. Correctness (does it do the right thing?)
3. TDD (is it proven?)
4. **Security** ← you are here

If the code isn't yet readable, correct, and tested, stop. Security review on shaky code sinks into chasing ordinary bugs. Bounce back to the right skill.

## Context

The code is part of a wealth management application. The threat model isn't a nation-state attacker — it's the realistic set of things that hurt users of a personal finance app:

- A leaked API key ending up in a public GitHub repo
- An OAuth token stolen via a misconfigured redirect
- A user's portfolio data pasted into an LLM prompt and logged by the provider
- A SQL injection in a portfolio filter that leaks another user's positions
- Cost runaway from an unauthenticated endpoint calling an expensive LLM or market data API
- Portfolio data exported over an unencrypted channel

## Step 1: Secrets

This is the most common failure mode in personal finance projects.

Walk through every place a secret could live:

- **Source code**: any string that looks like an API key, password, token, or connection string. Flag every `sk-...`, `pk-...`, `AKIA...`, JWT-shaped blob, or hex string long enough to be a credential.
- **`.env` files**: acceptable for local dev, never committed. Verify `.env` is in `.gitignore`. Verify no `.env` file is in the current or historical commits.
- **Config files committed to Git**: flag any `config.json`, `settings.yaml`, `secrets.toml` that contains non-placeholder values.
- **Docker images**: flag any `ENV` or `ARG` that bakes a secret into the image. Secrets belong in runtime config, not image layers.
- **Logs**: flag any `log.info("connected to db with %s", conn_string)`, any error message that echoes the request body, any stack trace that might include an `Authorization` header.
- **Test fixtures**: flag any real-looking key or token in test data, even if it's "just for testing".

For production deployment (Chapter 8): verify the code reads secrets from a secret manager (Google Secret Manager, AWS Secrets Manager, Vault) and not from committed files or environment variables that originate from committed files.

## Step 2: Authentication and authorization

If the code touches user sessions or access control, verify:

- **Authentication**: every endpoint that returns user data requires a valid session. Flag any route that doesn't.
- **Authorization**: every endpoint that returns user data filters by the authenticated user's ID. Flag any query that uses a user ID from the request body or URL without checking it matches the session.
- **OAuth flows**: verify `state` parameter is used and validated on callback. Verify `redirect_uri` is on an allowlist, not derived from the request. Verify tokens are stored server-side, not in `localStorage` or non-`HttpOnly` cookies.
- **Session handling**: verify cookies are `HttpOnly`, `Secure`, `SameSite=Lax` or stricter. Flag any session ID passed in a URL.
- **Admin / debug endpoints**: any route with `/admin`, `/debug`, `/internal`, `/health/detailed` must either not exist in production or require strong authentication.

## Step 3: Data exposure to LLMs

Specific to AI-augmented finance apps: what portfolio data leaves the machine?

- Flag any code that passes the full portfolio (positions, cost basis, transaction history) to an external LLM provider without the user explicitly opting in.
- Flag any prompt template that interpolates user PII (name, email, phone, account number) when the functional need is only the positions.
- Flag any logging of prompts or LLM responses that could include portfolio data to a third-party observability service.
- Verify that if the code uses LangChain, OpenAI, Anthropic, or similar, the request is going through a provider the user has consented to — not a third-party proxy.
- If the code uses a local model (Ollama, llama.cpp, etc.) for sensitive analysis, confirm that anything passed to the _remote_ model is sanitized.

This isn't paranoia. Portfolio data is financial data. Sending it to a third party without consent is a privacy failure even if no one malicious is involved.

## Step 4: Injection and untrusted input

For every piece of data that enters the system from outside (HTTP request, CSV upload, broker API, market data API):

- **SQL injection**: verify all queries use parameterized statements or a query builder. Flag any string concatenation or format string that builds SQL. Especially dangerous in Go with `fmt.Sprintf` and in Python with `%` or f-strings in queries.
- **Command injection**: flag any `os.system`, `subprocess.run(..., shell=True)`, `exec.Command` that interpolates user input.
- **Path traversal**: flag any code that opens a file whose path includes user input without calling the OS's canonical-path check and verifying it's within an allowed directory.
- **XSS (Next.js frontend)**: flag any `dangerouslySetInnerHTML` or direct DOM manipulation with user content. Verify user-submitted strings (e.g., portfolio notes) are rendered by React as text, not as HTML.
- **CSV import**: flag any cell that starts with `=`, `+`, `-`, `@`, or tab — these are CSV injection vectors that execute when the file is reopened in Excel.
- **Numeric input**: verify bounds checking on any user-supplied number that controls a loop, query limit, or resource allocation.

## Step 5: Transport and storage

- **TLS**: verify all external calls use `https://`, not `http://`. Flag any `http://` URL, especially for yfinance, Alpaca, or any data provider.
- **Certificate verification**: flag any code that disables TLS verification (`InsecureSkipVerify: true` in Go, `verify=False` in Python requests).
- **At-rest encryption**: if the code stores API tokens or OAuth refresh tokens in the database, verify they're encrypted. Plaintext tokens in Postgres are a leaked database away from being stolen.
- **Backup policy**: if the code handles backups, verify they're encrypted and that their storage is access-controlled.

## Step 6: Cost and abuse

Less classic, but relevant for AI-augmented apps:

- **Unauthenticated expensive endpoints**: any route that calls an LLM, market data API, or any per-call-billed service must require authentication and rate limiting. Otherwise a scraper will find it and run up the bill.
- **User-controlled loops**: flag any endpoint where a user can control the number of iterations (e.g., `num_assets_to_analyze` in a request body) without a server-side cap.
- **Unbounded results**: flag any query that returns all rows without pagination or a hard limit.

## Step 7: Dependencies

- Run `npm audit` / `pip-audit` / `govulncheck` mentally, or ask the author to run it.
- Flag any dependency that's a single-maintainer package with low stars doing something security-sensitive (auth, crypto, parsing).
- Flag any pinned-to-latest (`^1.2.3`, `~1.2.3`) version in a security-sensitive dependency where reproducible builds matter.

## Step 8: Deliver

### 8a. Verdict

One line: **SAFE TO DEPLOY**, **SAFE WITH REQUIRED FIXES**, or **DO NOT DEPLOY**.

### 8b. Findings

Grouped by severity:

**Critical** — exploitable right now. Leaked secrets, SQL injection, authentication bypass, plaintext token storage. Every item in this group blocks deployment.

**High** — not yet exploited but exposed. Missing TLS verification, CSRF missing, OAuth state not validated, portfolio data leaving the machine without consent.

**Medium** — bad posture, not immediately exploitable. Missing rate limits, verbose error messages, overly broad CORS, unencrypted backups.

**Low** — hygiene. Outdated dependencies with no known exploits, missing security headers, logs that are slightly too chatty.

For each finding: file:line → the vulnerability → the concrete fix (not "use best practices" — show the corrected code or config).

### 8c. Ship / don't ship

End with an unambiguous sentence. If deployment is blocked, name the items that have to change before you'd re-review.

## Principles

- **Assume the repo will be public.** Even if it won't be, assume it will. If a committed file would embarrass the author on a public GitHub, treat that as a critical finding.
- **A leaked key is leaked forever.** Don't treat "I'll rotate it" as a fix. Rotation is required _and_ the leak is a finding in its own right — every commit that contained it must be purged.
- **Defense in depth.** If the authentication layer fails, does authorization still catch it? If the query is wrong, does row-level security backstop it? A single layer is a single point of failure.
- **Don't be a checklist.** A generic "add HTTPS" comment helps no one. Point at the specific call site and the specific fix.
- **Security review doesn't replace correctness review.** If the code has a bug but not a vulnerability, that belongs in correctness review, not here. Don't let one skill swallow another.
- **User consent is a security property.** Data going to a third party without the user's knowledge is a privacy breach even if no external attacker is involved.
