package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/george/golinks/models"
	"github.com/george/golinks/storage"
	"github.com/gorilla/mux"
)

// Handler contains the HTTP handlers for the application
type Handler struct {
	store storage.Store
}

// NewHandler creates a new Handler with the given storage
func NewHandler(store storage.Store) *Handler {
	return &Handler{store: store}
}

// validShortcode checks if a shortcode is valid
func validShortcode(s string) bool {
	// Allow alphanumeric, hyphens, and underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, s)
	return matched && len(s) >= 1 && len(s) <= 100
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// Redirect handles the redirect for a shortcode
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortcode := vars["shortcode"]

	link, err := h.store.GetLink(shortcode)
	if err == storage.ErrNotFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("Error getting link: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Increment click count asynchronously
	go func() {
		if err := h.store.IncrementClickCount(shortcode); err != nil {
			log.Printf("Error incrementing click count: %v", err)
		}
	}()

	http.Redirect(w, r, link.URL, http.StatusFound)
}

// ListLinks returns all links
func (h *Handler) ListLinks(w http.ResponseWriter, r *http.Request) {
	links, err := h.store.ListLinks()
	if err != nil {
		log.Printf("Error listing links: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to list links")
		return
	}

	if links == nil {
		links = []*models.Link{}
	}

	respondJSON(w, http.StatusOK, links)
}

// CreateLink creates a new link
func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request) {
	var req models.CreateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate shortcode
	req.Shortcode = strings.TrimSpace(req.Shortcode)
	if !validShortcode(req.Shortcode) {
		respondError(w, http.StatusBadRequest, "Invalid shortcode. Use only letters, numbers, hyphens, and underscores.")
		return
	}

	// Validate URL
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "URL is required")
		return
	}

	// Add protocol if missing
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		req.URL = "https://" + req.URL
	}

	link := &models.Link{
		Shortcode:   req.Shortcode,
		URL:         req.URL,
		Description: strings.TrimSpace(req.Description),
	}

	if err := h.store.CreateLink(link); err != nil {
		if err == storage.ErrAlreadyExists {
			respondError(w, http.StatusConflict, "Shortcode already exists")
			return
		}
		log.Printf("Error creating link: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create link")
		return
	}

	respondJSON(w, http.StatusCreated, link)
}

// GetLink retrieves a single link
func (h *Handler) GetLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortcode := vars["shortcode"]

	link, err := h.store.GetLink(shortcode)
	if err == storage.ErrNotFound {
		respondError(w, http.StatusNotFound, "Link not found")
		return
	}
	if err != nil {
		log.Printf("Error getting link: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to get link")
		return
	}

	respondJSON(w, http.StatusOK, link)
}

// UpdateLink updates an existing link
func (h *Handler) UpdateLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortcode := vars["shortcode"]

	var req models.UpdateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Get existing link
	existing, err := h.store.GetLink(shortcode)
	if err == storage.ErrNotFound {
		respondError(w, http.StatusNotFound, "Link not found")
		return
	}
	if err != nil {
		log.Printf("Error getting link: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to get link")
		return
	}

	// Update fields if provided
	if req.URL != "" {
		url := strings.TrimSpace(req.URL)
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}
		existing.URL = url
	}
	if req.Description != "" {
		existing.Description = strings.TrimSpace(req.Description)
	}

	if err := h.store.UpdateLink(shortcode, existing); err != nil {
		log.Printf("Error updating link: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to update link")
		return
	}

	respondJSON(w, http.StatusOK, existing)
}

// DeleteLink deletes a link
func (h *Handler) DeleteLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortcode := vars["shortcode"]

	if err := h.store.DeleteLink(shortcode); err == storage.ErrNotFound {
		respondError(w, http.StatusNotFound, "Link not found")
		return
	} else if err != nil {
		log.Printf("Error deleting link: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to delete link")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetLinkStats returns statistics for a link
func (h *Handler) GetLinkStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortcode := vars["shortcode"]

	stats, err := h.store.GetStats(shortcode)
	if err == storage.ErrNotFound {
		respondError(w, http.StatusNotFound, "Link not found")
		return
	}
	if err != nil {
		log.Printf("Error getting stats: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// HomePage serves the home page
func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("home").Parse(homeTemplate))
	tmpl.Execute(w, nil)
}

// AdminPage serves the admin page
func (h *Handler) AdminPage(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("admin").Parse(adminTemplate))
	tmpl.Execute(w, nil)
}

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
            <button class="btn" onclick="showCreateModal()">+ New Link</button>
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

        // Load links on page load
        document.addEventListener('DOMContentLoaded', loadLinks);

        // Update preview as user types
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
                const response = await fetch(API_BASE);
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
                                    <button class="btn btn-small" onclick="editLink('${link.shortcode}')">Edit</button>
                                    <button class="btn btn-small btn-danger" onclick="deleteLink('${link.shortcode}')">Delete</button>
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
                        body: JSON.stringify({ url, description })
                    });
                } else {
                    response = await fetch(API_BASE, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ shortcode, url, description })
                    });
                }

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
                    method: 'DELETE'
                });

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

        // Close modal on outside click
        document.getElementById('modal').addEventListener('click', function(e) {
            if (e.target === this) hideModal();
        });

        // Close modal on Escape
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') hideModal();
        });
    </script>
</body>
</html>`
