# Authentication

GoLinks supports three authentication modes: **`none`**, **`local`**, and **`proxy`**.

## Modes Overview

### `none` (Default)
Authentication is completely disabled. The admin portal (`/admin`) and API (`/api/*`) are public and accessible without credentials. This is useful for:
- Testing and development.
- Secure, private networks where everyone is trusted.
- Environments where authentication is handled externally by a proxy that intercepts all requests.

### `local` (Database-backed)
Authentication uses a username and password stored in the database.
- **Enable**: Set `GOLINKS_AUTH_MODE=local`.
- **First-time Setup**: When `DATABASE_URL` is set (PostgreSQL), authentication persists. If using the default In-Memory store, user accounts are lost on restart.
- **Registration**: Navigate to `http://localhost:8080/register` to create a new user account.
- **Login**: Navigate to `http://localhost:8080/login` to sign in.
- **Session**: A session cookie (`golinks_session`) is used to maintain login state.

**Note**: In `local` mode, if no users exist, the server log will print a hint to visit `/register`.

### `proxy` (Header-based SSO)
Authentication relies on a trusted HTTP header set by a reverse proxy (e.g., Authelia, OAuth2 Proxy, Nginx).
- **Enable**: Set `GOLINKS_AUTH_MODE=proxy`.
- **Header**: Set `GOLINKS_AUTH_HEADER` to the header name containing the username (default: `Remote-User`).
- **Trusted Proxies**: Set `GOLINKS_AUTH_TRUSTED_PROXIES` to a comma-separated list of trusted proxy IP addresses/CIDRs (e.g., `10.0.0.1,172.16.0.0/12`).

## API Access

When authentication is enabled (`local` or `proxy`), API endpoints are protected. You can authenticate API requests using an API Key.

1.  Set the API Key environment variable:
    ```bash
    export GOLINKS_API_KEY=your-secret-api-key
    ```
2.  Include the key in the `Authorization` header of your requests:
    ```bash
    curl -H "Authorization: Bearer your-secret-api-key" http://localhost:8080/api/links
    ```

## Session Security

- **`GOLINKS_AUTH_SECRET`**: A secret key used to sign session cookies. If not set, a random secret is generated on startup.
    - **Important**: In production (especially with multiple replicas), always set this variable to a persistent secret string. Otherwise, sessions will be invalidated on every server restart.
- **`GOLINKS_COOKIE_SECURE`**: Set to `true` to mark session cookies as `Secure` (requires HTTPS).

## Summary of Environment Variables

| Variable | Description | Default |
| :--- | :--- | :--- |
| `GOLINKS_AUTH_MODE` | Authentication mode (`none`, `local`, `proxy`) | `none` |
| `GOLINKS_AUTH_SECRET` | Secret key for signing session cookies | (random) |
| `GOLINKS_API_KEY` | Bearer token for API access | (none) |
| `GOLINKS_COOKIE_SECURE` | Set `true` for Secure cookies (HTTPS only) | `false` |
| `GOLINKS_AUTH_HEADER` | Header name for proxy auth (e.g., `Remote-User`, `X-Forwarded-User`) | `Remote-User` |
| `GOLINKS_AUTH_TRUSTED_PROXIES` | Comma-separated trusted proxy IPs/CIDRs | (trust all) |
