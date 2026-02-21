package http

const loginTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoLinks — Login</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .card {
            background: white;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            padding: 2.5rem;
            width: 100%;
            max-width: 400px;
        }
        h1 { text-align: center; color: #333; margin-bottom: 1.5rem; }
        .form-group { margin-bottom: 1rem; }
        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            font-weight: 600;
            color: #333;
        }
        .form-group input {
            width: 100%;
            padding: 0.75rem;
            border: 2px solid #e0e0e0;
            border-radius: 6px;
            font-size: 1rem;
            transition: border-color 0.2s;
        }
        .form-group input:focus {
            outline: none;
            border-color: #667eea;
        }
        .btn {
            display: block;
            width: 100%;
            padding: 0.75rem;
            background: #667eea;
            color: white;
            border: none;
            border-radius: 6px;
            font-size: 1rem;
            cursor: pointer;
            font-weight: 600;
            margin-top: 1rem;
            transition: background 0.2s;
        }
        .btn:hover { background: #5a6fd6; }
    </style>
</head>
<body>
    <div class="card">
        <h1>🔗 GoLinks</h1>
        <form method="POST" action="/login">
            <div class="form-group">
                <label for="username">Username</label>
                <input type="text" id="username" name="username" required autofocus>
            </div>
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit" class="btn">Sign In</button>
        </form>
        <p style="text-align: center; margin-top: 1rem;">
            <a href="/register" style="color: #667eea; text-decoration: none;">Create an account</a>
        </p>
    </div>
</body>
</html>`

const loginErrorTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoLinks — Login</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .card {
            background: white;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            padding: 2.5rem;
            width: 100%;
            max-width: 400px;
        }
        h1 { text-align: center; color: #333; margin-bottom: 1.5rem; }
        .form-group { margin-bottom: 1rem; }
        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            font-weight: 600;
            color: #333;
        }
        .form-group input {
            width: 100%;
            padding: 0.75rem;
            border: 2px solid #e0e0e0;
            border-radius: 6px;
            font-size: 1rem;
            transition: border-color 0.2s;
        }
        .form-group input:focus {
            outline: none;
            border-color: #667eea;
        }
        .btn {
            display: block;
            width: 100%;
            padding: 0.75rem;
            background: #667eea;
            color: white;
            border: none;
            border-radius: 6px;
            font-size: 1rem;
            cursor: pointer;
            font-weight: 600;
            margin-top: 1rem;
            transition: background 0.2s;
        }
        .btn:hover { background: #5a6fd6; }
        .error {
            background: #f8d7da;
            color: #721c24;
            padding: 0.75rem;
            border-radius: 6px;
            margin-bottom: 1rem;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>🔗 GoLinks</h1>
        <div class="error">Invalid username or password</div>
        <form method="POST" action="/login">
            <div class="form-group">
                <label for="username">Username</label>
                <input type="text" id="username" name="username" required autofocus>
            </div>
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit" class="btn">Sign In</button>
        </form>
        <p style="text-align: center; margin-top: 1rem;">
            <a href="/register" style="color: #667eea; text-decoration: none;">Create an account</a>
        </p>
    </div>
</body>
</html>`

const homeTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoLinks</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container {
            text-align: center;
            color: white;
        }
        h1 { font-size: 4rem; margin-bottom: 1rem; }
        p { font-size: 1.5rem; margin-bottom: 2rem; opacity: 0.9; }
        .btn {
            display: inline-block;
            padding: 1rem 2rem;
            background: white;
            color: #667eea;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(0,0,0,0.2);
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔗 GoLinks</h1>
        <p>Create and manage short links for your team</p>
        <a href="/admin" class="btn">Open Admin Panel</a>
    </div>
</body>
</html>`

const adminTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoLinks Admin</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: #f5f5f5;
            min-height: 100vh;
            padding: 2rem;
        }
        .container { max-width: 1000px; margin: 0 auto; }
        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2rem;
        }
        h1 { color: #333; }
        .btn {
            padding: 0.75rem 1.5rem;
            background: #667eea;
            color: white;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-size: 1rem;
            transition: background 0.2s;
        }
        .btn:hover { background: #5a6fd6; }
        .btn-danger { background: #e74c3c; }
        .btn-danger:hover { background: #c0392b; }
        .btn-small { padding: 0.5rem 1rem; font-size: 0.875rem; }
        .card {
            background: white;
            border-radius: 12px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 1.5rem;
            margin-bottom: 1.5rem;
        }
        .form-group { margin-bottom: 1rem; }
        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            font-weight: 600;
            color: #333;
        }
        .form-group input, .form-group textarea {
            width: 100%;
            padding: 0.75rem;
            border: 2px solid #e0e0e0;
            border-radius: 6px;
            font-size: 1rem;
            transition: border-color 0.2s;
        }
        .form-group input:focus, .form-group textarea:focus {
            outline: none;
            border-color: #667eea;
        }
        .form-row { display: flex; gap: 1rem; }
        .form-row .form-group { flex: 1; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 1rem; text-align: left; border-bottom: 1px solid #e0e0e0; }
        th { font-weight: 600; color: #666; }
        .shortcode { font-family: monospace; font-weight: 600; color: #667eea; }
        .url {
            max-width: 300px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }
        .actions { display: flex; gap: 0.5rem; }
        .stats { color: #888; font-size: 0.875rem; }
        .empty { text-align: center; padding: 3rem; color: #888; }
        .message {
            padding: 1rem;
            border-radius: 6px;
            margin-bottom: 1rem;
            display: none;
        }
        .message.success { background: #d4edda; color: #155724; display: block; }
        .message.error { background: #f8d7da; color: #721c24; display: block; }
        .link-preview {
            font-size: 0.875rem;
            color: #888;
            margin-top: 0.25rem;
        }
        .modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0,0,0,0.5);
            align-items: center;
            justify-content: center;
            z-index: 1000;
        }
        .modal.active { display: flex; }
        .modal-content {
            background: white;
            padding: 2rem;
            border-radius: 12px;
            width: 100%;
            max-width: 500px;
        }
        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 1.5rem;
        }
        .modal-close {
            background: none;
            border: none;
            font-size: 1.5rem;
            cursor: pointer;
            color: #888;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🔗 GoLinks Admin</h1>
            <div style="display: flex; gap: 0.5rem; align-items: center;">
                <button class="btn" onclick="showCreateModal()">+ New Link</button>
                <form method="POST" action="/logout" style="margin: 0;">
                    <button type="submit" class="btn" style="background: #888;">Logout</button>
                </form>
            </div>
        </header>

        <div id="message" class="message"></div>

        <div class="card">
            <h2 style="margin-bottom: 1rem;">Your Links</h2>
            <div id="links-container">
                <div class="empty">Loading...</div>
            </div>
        </div>
    </div>

    <!-- Create/Edit Modal -->
    <div id="modal" class="modal">
        <div class="modal-content">
            <div class="modal-header">
                <h2 id="modal-title">Create New Link</h2>
                <button class="modal-close" onclick="hideModal()">&times;</button>
            </div>
            <form id="link-form" onsubmit="handleSubmit(event)">
                <input type="hidden" id="edit-mode" value="">
                <div class="form-group">
                    <label for="shortcode">Shortcode</label>
                    <input type="text" id="shortcode" placeholder="docs" required>
                    <div class="link-preview" id="preview"></div>
                </div>
                <div class="form-group">
                    <label for="url">Destination URL</label>
                    <input type="text" id="url" placeholder="https://example.com/documentation" required>
                </div>
                <div class="form-group">
                    <label for="description">Description (optional)</label>
                    <textarea id="description" rows="2" placeholder="What is this link for?"></textarea>
                </div>
                <button type="submit" class="btn" style="width: 100%;">Save Link</button>
            </form>
        </div>
    </div>

    <script>
        const API_BASE = '/api/links';
        let links = [];
        let currentUser = null;

        document.addEventListener('DOMContentLoaded', async () => {
            await loadCurrentUser();
            loadLinks();
        });

        async function loadCurrentUser() {
            try {
                const response = await fetch('/api/me', { credentials: 'same-origin' });
                if (response.ok) {
                    currentUser = await response.json();
                }
            } catch (err) {
                console.error('Failed to load current user', err);
            }
        }

        document.getElementById('shortcode').addEventListener('input', function() {
            const shortcode = this.value.trim();
            const preview = document.getElementById('preview');
            if (shortcode) {
                preview.textContent = 'go/' + shortcode + ' → ' + window.location.origin + '/' + shortcode;
            } else {
                preview.textContent = '';
            }
        });

        async function loadLinks() {
            try {
                const response = await fetch(API_BASE, { credentials: 'same-origin' });
                if (response.status === 401) { window.location.href = '/login'; return; }
                links = await response.json();
                renderLinks();
            } catch (err) {
                showMessage('Failed to load links', 'error');
            }
        }

        function renderLinks() {
            const container = document.getElementById('links-container');

            if (!links || links.length === 0) {
                container.innerHTML = '<div class="empty">No links yet. Create your first one!</div>';
                return;
            }

            container.innerHTML = ` + "`" + `
                <table>
                    <thead>
                        <tr>
                            <th>Shortcode</th>
                            <th>Destination</th>
                            <th>Description</th>
                            <th>Clicks</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${links.map(link => ` + "`" + `
                            <tr>
                                <td>
                                    <a href="/${link.shortcode}" target="_blank" class="shortcode">
                                        go/${link.shortcode}
                                    </a>
                                </td>
                                <td class="url" title="${escapeHtml(link.url)}">
                                    <a href="${escapeHtml(link.url)}" target="_blank">${escapeHtml(link.url)}</a>
                                </td>
                                <td>${escapeHtml(link.description || '-')}</td>
                                <td class="stats">${link.click_count}</td>
                                <td class="actions">
                                    ${(currentUser && (currentUser.is_admin || currentUser.username === link.owner)) ? ` + "`" + `
                                        <button class="btn btn-small" onclick="editLink('${link.shortcode}')">Edit</button>
                                        <button class="btn btn-small btn-danger" onclick="deleteLink('${link.shortcode}')">Delete</button>
                                    ` + "`" + ` : ''}
                                </td>
                            </tr>
                        ` + "`" + `).join('')}
                    </tbody>
                </table>
            ` + "`" + `;
        }

        function showCreateModal() {
            document.getElementById('modal-title').textContent = 'Create New Link';
            document.getElementById('edit-mode').value = '';
            document.getElementById('shortcode').value = '';
            document.getElementById('shortcode').disabled = false;
            document.getElementById('url').value = '';
            document.getElementById('description').value = '';
            document.getElementById('preview').textContent = '';
            document.getElementById('modal').classList.add('active');
        }

        function editLink(shortcode) {
            const link = links.find(l => l.shortcode === shortcode);
            if (!link) return;

            document.getElementById('modal-title').textContent = 'Edit Link';
            document.getElementById('edit-mode').value = shortcode;
            document.getElementById('shortcode').value = link.shortcode;
            document.getElementById('shortcode').disabled = true;
            document.getElementById('url').value = link.url;
            document.getElementById('description').value = link.description || '';
            document.getElementById('preview').textContent = 'go/' + shortcode;
            document.getElementById('modal').classList.add('active');
        }

        function hideModal() {
            document.getElementById('modal').classList.remove('active');
        }

        async function handleSubmit(event) {
            event.preventDefault();

            const editMode = document.getElementById('edit-mode').value;
            const shortcode = document.getElementById('shortcode').value.trim();
            const url = document.getElementById('url').value.trim();
            const description = document.getElementById('description').value.trim();

            try {
                let response;
                if (editMode) {
                    response = await fetch(API_BASE + '/' + editMode, {
                        method: 'PUT',
                        headers: { 'Content-Type': 'application/json' },
                        credentials: 'same-origin',
                        body: JSON.stringify({ url, description })
                    });
                } else {
                    response = await fetch(API_BASE, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        credentials: 'same-origin',
                        body: JSON.stringify({ shortcode, url, description })
                    });
                }

                if (response.status === 401) { window.location.href = '/login'; return; }

                if (!response.ok) {
                    const data = await response.json();
                    throw new Error(data.error || 'Failed to save link');
                }

                hideModal();
                showMessage(editMode ? 'Link updated!' : 'Link created!', 'success');
                loadLinks();
            } catch (err) {
                showMessage(err.message, 'error');
            }
        }

        async function deleteLink(shortcode) {
            if (!confirm('Are you sure you want to delete go/' + shortcode + '?')) {
                return;
            }

            try {
                const response = await fetch(API_BASE + '/' + shortcode, {
                    method: 'DELETE',
                    credentials: 'same-origin'
                });

                if (response.status === 401) { window.location.href = '/login'; return; }

                if (!response.ok) {
                    throw new Error('Failed to delete link');
                }

                showMessage('Link deleted!', 'success');
                loadLinks();
            } catch (err) {
                showMessage(err.message, 'error');
            }
        }

        function showMessage(text, type) {
            const el = document.getElementById('message');
            el.textContent = text;
            el.className = 'message ' + type;
            setTimeout(() => el.className = 'message', 3000);
        }

        function escapeHtml(str) {
            if (!str) return '';
            return str.replace(/&/g, '&amp;')
                      .replace(/</g, '&lt;')
                      .replace(/>/g, '&gt;')
                      .replace(/"/g, '&quot;');
        }

        document.getElementById('modal').addEventListener('click', function(e) {
            if (e.target === this) hideModal();
        });

        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') hideModal();
        });
    </script>
</body>
</html>`

const registerTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoLinks — Register</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .card {
            background: white;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            padding: 2.5rem;
            width: 100%;
            max-width: 400px;
        }
        h1 { text-align: center; color: #333; margin-bottom: 0.5rem; }
        p { text-align: center; color: #666; margin-bottom: 1.5rem; }
        .form-group { margin-bottom: 1rem; }
        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            font-weight: 600;
            color: #333;
        }
        .form-group input {
            width: 100%;
            padding: 0.75rem;
            border: 2px solid #e0e0e0;
            border-radius: 6px;
            font-size: 1rem;
            transition: border-color 0.2s;
        }
        .form-group input:focus { outline: none; border-color: #667eea; }
        .btn {
            display: block;
            width: 100%;
            padding: 0.75rem;
            background: #667eea;
            color: white;
            border: none;
            border-radius: 6px;
            font-size: 1rem;
            cursor: pointer;
            font-weight: 600;
            margin-top: 1rem;
            transition: background 0.2s;
        }
        .btn:hover { background: #5a6fd6; }
    </style>
</head>
<body>
    <div class="card">
        <h1>GoLinks Register</h1>
        <p>Create a new account.</p>
        <form method="POST" action="/register">
            <div class="form-group">
                <label for="username">Username</label>
                <input type="text" id="username" name="username" required autofocus>
            </div>
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit" class="btn">Create Account</button>
        </form>
        <p style="text-align: center; margin-top: 1rem;">
            <a href="/login" style="color: #667eea; text-decoration: none;">Already have an account? Sign in</a>
        </p>
    </div>
</body>
</html>`
