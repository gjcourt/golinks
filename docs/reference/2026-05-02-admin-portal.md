---
title: GoLinks admin portal
status: Stable
created: 2026-05-02
updated: 2026-05-02
updated_by: gjcourt
tags: [reference, admin, ui]
---

# Admin Portal

The Admin Portal provides a user-friendly interface for managing your short links.

## Access

Navigate to `http://localhost:8080/admin` in your web browser.

- **Authentication**: By default (`GOLINKS_AUTH_MODE=none`), the admin portal is public.
- **Enable Auth**: Set `GOLINKS_AUTH_MODE=local` to require login.
  - If enabled, you will be redirected to `/login`.
  - First-time users can register at `/register`.

## Features

### Create Links
Click the **+ New Link** button.
1. **Shortcode**: The memorable part (e.g., `docs`).
2. **Destination URL**: Where it redirects (e.g., `https://docs.google.com/...`).
3. **Description**: Optional note about the link.

### Edit Links
Click **Edit** next to any link you own (or all links if you are an admin).
- Modify the destination URL or description.
- Shortcodes cannot be changed once created (delete and recreate instead).

### Delete Links
Click **Delete** to permanently remove a link.

### Themes
Use the theme switcher in the bottom-right corner (or top-right on login/register pages) to customize the appearance.
- **Default**: Purple/Blue gradient.
- **Ocean**: Blue/Cyan.
- **Sunset**: Pink/Orange.
- **Forest**: Green/Teal.
- **Dark**: Dark gray/Black mode.

The selected theme is saved to your browser's local storage.
