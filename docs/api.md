# GoLinks API

The API allows programmatic interaction with GoLinks for creating, listing, and managing links.

## Access

Requests to the API must be authenticated if authentication is enabled (`none` default).

When `GOLINKS_AUTH_MODE=local` or `GOLINKS_AUTH_MODE=proxy`:
1. Use the `Authorization: Bearer <GOLINKS_API_KEY>` header.
2. The key must match the `GOLINKS_API_KEY` environment variable on the server.

### Base URL

```http
http://localhost:8080
```

## Endpoints

### `GET /api/me`

Returns the currently authenticated user information.

**Response (JSON)**:
```json
{
  "username": "admin",
  "is_admin": true
}
```

### `GET /api/links`

Lists all links in the repository.

**Authorization**: Required.

**Response (JSON)**:
```json
[
  {
    "id": 1,
    "shortcode": "docs",
    "url": "https://docs.google.com/...",
    "description": "Team Documentation",
    "owner": "admin",
    "click_count": 42,
    "created_at": "2024-02-21T09:00:00Z",
    "updated_at": "2024-02-21T09:00:00Z"
  }
]
```

### `POST /api/links`

Creates a new link.

**Authorization**: Required.

**Request (JSON)**:
```json
{
  "shortcode": "status",
  "url": "https://status.example.com",
  "description": "Company Status Page"
}
```

**Response (JSON)**:
```json
{
  "id": 2,
  "shortcode": "status",
  "url": "https://status.example.com"
}
```

### `GET /api/links/{shortcode}`

Retrieves details for a specific link.

**Authorization**: Required.

**Response (JSON)**:
```json
{
  "id": 2,
  "shortcode": "status",
  "url": "https://status.example.com",
  "description": "Company Status Page",
  "click_count": 0
}
```

### `PUT /api/links/{shortcode}`

Updates an existing link with new URL or Description.

**Authorization**: Required (unless you own the link).

**Request (JSON)**:
```json
{
  "url": "https://newdestination.com",
  "description": "Updated destination"
}
```

**Response (JSON)**:
```json
{
  "id": 2,
  "shortcode": "status",
  "url": "https://newdestination.com",
  "description": "Updated destination"
}
```

### `DELETE /api/links/{shortcode}`

Removes a shortened link permanently.

**Authorization**: Required (unless you own the link).

**Response (Status)**:
- `204 No Content`: Successful deletion.
- `404 Not Found`: Link does not exist.
- `401 Unauthorized`: Permission denied.

### Public Endpoints

The following are accessible without auth:

- **`GET /{shortcode}`**: Performs the redirection.
- **`GET /`**: Home page.
- **`GET /login`, `GET /register`**: Authentication routes.
