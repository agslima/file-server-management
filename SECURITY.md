# Security Policy

## Supported Versions

As this is a Capstone Project for demonstration purposes, only the latest code in the `master` (or `main`) branch is currently supported and maintained.

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| Older   | :x:                |

## Reporting a Vulnerability

We take the security of this microservice seriously. If you discover a security vulnerability within this project, please follow these steps:

### 1. Do NOT open a public issue
Publicly creating an issue may put the application at risk before a patch is released. Please keep the vulnerability details private.

### 2. How to contact
Please send an email to **[a.agnaldosilva at gmail.com]** with the subject: `[SECURITY] Vulnerability Report - DevOps Capstone`.

In your email, please include:
* Type of vulnerability (e.g., SQL Injection, XSS, RCE).
* Steps to reproduce the issue.
* The potential impact of the vulnerability.

You can also use the **Private Vulnerability Reporting** feature on GitHub. 
Go to the **Security** tab of this repository -> **Advisories** -> **New draft security advisory**.

### 3. Response Timeline
* **Acknowledgement:** I will attempt to acknowledge your report within 48 hours.
* **Fix:** If confirmed, I will work on a patch and release it via the CI/CD pipeline (Tekton/GitHub Actions).

## Security Measures in Place

This project implements several DevSecOps practices to prevent common vulnerabilities:

* **Security Headers:** Implemented using `Flask-Talisman` to protect against XSS and other attacks.
* **CORS Policies:** Configured via `Flask-Cors` to restrict resource sharing.
* **Dependency Scanning:** The project dependencies are monitored for known vulnerabilities.
* **Linting:** `Flake8` is used in the CI pipeline to enforce coding standards and catch potential errors early.
* **Container Security:** Docker images are built using standard base images.

## Disclaimer

This is an educational project created for the **IBM DevOps and Software Engineering Professional Certificate**. While best efforts have been made to secure the application, it should be deployed in a production environment with caution and appropriate additional security layers (Firewalls, WAFs, etc).

