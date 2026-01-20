package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/hivedeck-agent/config"
)

// SetupHandlers handles the setup and settings endpoints
type SetupHandlers struct {
	cfg *config.Config
}

// NewSetupHandlers creates setup handlers
func NewSetupHandlers(cfg *config.Config) *SetupHandlers {
	return &SetupHandlers{cfg: cfg}
}

// SetupPage serves the initial setup HTML page (no auth required)
func (h *SetupHandlers) SetupPage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, setupPageHTML)
}

// SettingsPage serves the settings HTML page (requires auth)
func (h *SetupHandlers) SettingsPage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, settingsPageHTML)
}

// GetSettings returns current settings (requires auth)
func (h *SetupHandlers) GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"port":             h.cfg.Port,
		"host":             h.cfg.Host,
		"allowed_origins":  h.cfg.AllowedOrigins,
		"allowed_services": h.cfg.AllowedServices,
		"allowed_paths":    h.cfg.AllowedPaths,
		"docker_enabled":   h.cfg.DockerEnabled,
		"log_level":        h.cfg.LogLevel,
		"rate_limit_rps":   h.cfg.RateLimitRPS,
		"env_file":         h.cfg.EnvFile,
		"setup_mode":       h.cfg.SetupMode,
		// Don't expose the actual API key, just indicate if it's set
		"api_key_configured": h.cfg.APIKey != "",
	})
}

// GenerateKey generates a new API key
func (h *SetupHandlers) GenerateKey(c *gin.Context) {
	apiKey, err := config.GenerateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate API key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key": apiKey,
	})
}

// SaveKey saves the API key to the .env file
func (h *SetupHandlers) SaveKey(c *gin.Context) {
	var req struct {
		APIKey string `json:"api_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: api_key is required",
		})
		return
	}

	// Validate API key length
	if len(req.APIKey) < 32 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "API key must be at least 32 characters",
		})
		return
	}

	// Save the API key
	if err := h.cfg.SaveAPIKey(req.APIKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save API key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "API key saved successfully",
		"api_key":  req.APIKey,
		"env_file": h.cfg.EnvFile,
		"note":     "Restart the agent to apply the new API key for authentication",
	})
}

// UpdateSettings updates agent settings
func (h *SetupHandlers) UpdateSettings(c *gin.Context) {
	var req struct {
		AllowedPaths    []string `json:"allowed_paths"`
		AllowedServices []string `json:"allowed_services"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}

	// Update settings in config
	updates := make(map[string]string)

	if len(req.AllowedPaths) > 0 {
		h.cfg.AllowedPaths = req.AllowedPaths
		updates["ALLOWED_PATHS"] = joinSlice(req.AllowedPaths)
	}

	if len(req.AllowedServices) > 0 {
		h.cfg.AllowedServices = req.AllowedServices
		updates["ALLOWED_SERVICES"] = joinSlice(req.AllowedServices)
	}

	// Save to .env file
	if err := config.UpdateEnvFile(h.cfg.EnvFile, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save settings: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Settings updated",
		"allowed_paths":    h.cfg.AllowedPaths,
		"allowed_services": h.cfg.AllowedServices,
		"note":             "Some settings may require restart to take effect",
	})
}

func joinSlice(s []string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ","
		}
		result += v
	}
	return result
}

const setupPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hivedeck Agent Setup</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: #ffffff;
            border-radius: 16px;
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
            padding: 40px;
            max-width: 500px;
            width: 100%;
        }
        .logo {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo svg {
            width: 64px;
            height: 64px;
            color: #3b82f6;
        }
        h1 {
            text-align: center;
            color: #1f2937;
            font-size: 24px;
            margin-bottom: 8px;
        }
        .subtitle {
            text-align: center;
            color: #6b7280;
            font-size: 14px;
            margin-bottom: 30px;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            color: #374151;
            font-size: 14px;
            font-weight: 500;
            margin-bottom: 8px;
        }
        input[type="text"] {
            width: 100%;
            padding: 12px 16px;
            border: 2px solid #e5e7eb;
            border-radius: 8px;
            font-size: 14px;
            font-family: 'Monaco', 'Menlo', monospace;
            transition: border-color 0.2s;
        }
        input[type="text"]:focus {
            outline: none;
            border-color: #3b82f6;
        }
        .btn-row {
            display: flex;
            gap: 12px;
            margin-bottom: 20px;
        }
        button {
            flex: 1;
            padding: 12px 20px;
            border: none;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
        }
        .btn-primary {
            background: #3b82f6;
            color: white;
        }
        .btn-primary:hover {
            background: #2563eb;
        }
        .btn-secondary {
            background: #f3f4f6;
            color: #374151;
        }
        .btn-secondary:hover {
            background: #e5e7eb;
        }
        .btn-success {
            background: #10b981;
            color: white;
            width: 100%;
        }
        .btn-success:hover {
            background: #059669;
        }
        .btn-success:disabled {
            background: #9ca3af;
            cursor: not-allowed;
        }
        .alert {
            padding: 12px 16px;
            border-radius: 8px;
            margin-bottom: 20px;
            font-size: 14px;
        }
        .alert-success {
            background: #d1fae5;
            color: #065f46;
            border: 1px solid #a7f3d0;
        }
        .alert-error {
            background: #fee2e2;
            color: #991b1b;
            border: 1px solid #fecaca;
        }
        .alert-info {
            background: #dbeafe;
            color: #1e40af;
            border: 1px solid #bfdbfe;
        }
        .hidden { display: none; }
        .copy-hint {
            font-size: 12px;
            color: #6b7280;
            margin-top: 8px;
        }
        .divider {
            text-align: center;
            color: #9ca3af;
            font-size: 12px;
            margin: 20px 0;
            position: relative;
        }
        .divider::before, .divider::after {
            content: '';
            position: absolute;
            top: 50%;
            width: 40%;
            height: 1px;
            background: #e5e7eb;
        }
        .divider::before { left: 0; }
        .divider::after { right: 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
            </svg>
        </div>
        <h1>Hivedeck Agent Setup</h1>
        <p class="subtitle">Configure your agent's API key to get started</p>

        <div id="alert" class="alert hidden"></div>

        <div class="form-group">
            <label for="apiKey">API Key</label>
            <input type="text" id="apiKey" placeholder="Enter or generate an API key">
            <p class="copy-hint">Save this key - you'll need it to connect from the dashboard</p>
        </div>

        <div class="btn-row">
            <button type="button" class="btn-secondary" onclick="generateKey()">Generate Key</button>
            <button type="button" class="btn-secondary" onclick="copyKey()">Copy</button>
        </div>

        <button type="button" class="btn-success" id="saveBtn" onclick="saveKey()">
            Save API Key
        </button>

        <div class="divider">After saving</div>

        <div class="alert alert-info">
            After saving, restart the agent, then add this server in your Hivedeck dashboard using the API key above.
        </div>
    </div>

    <script>
        const apiKeyInput = document.getElementById('apiKey');
        const alertDiv = document.getElementById('alert');
        const saveBtn = document.getElementById('saveBtn');

        function showAlert(message, type) {
            alertDiv.textContent = message;
            alertDiv.className = 'alert alert-' + type;
        }

        async function generateKey() {
            try {
                const res = await fetch('/setup/generate', { method: 'POST' });
                const data = await res.json();
                if (data.api_key) {
                    apiKeyInput.value = data.api_key;
                    showAlert('API key generated! Remember to copy it.', 'success');
                } else {
                    showAlert(data.error || 'Failed to generate key', 'error');
                }
            } catch (err) {
                showAlert('Failed to generate key: ' + err.message, 'error');
            }
        }

        function copyKey() {
            if (apiKeyInput.value) {
                navigator.clipboard.writeText(apiKeyInput.value);
                showAlert('API key copied to clipboard!', 'success');
            } else {
                showAlert('No API key to copy', 'error');
            }
        }

        async function saveKey() {
            const apiKey = apiKeyInput.value.trim();
            if (!apiKey) {
                showAlert('Please enter or generate an API key', 'error');
                return;
            }
            if (apiKey.length < 32) {
                showAlert('API key must be at least 32 characters', 'error');
                return;
            }

            saveBtn.disabled = true;
            saveBtn.textContent = 'Saving...';

            try {
                const res = await fetch('/setup/save', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ api_key: apiKey })
                });
                const data = await res.json();
                if (res.ok) {
                    showAlert('API key saved! Restart the agent to apply.', 'success');
                    saveBtn.textContent = 'Saved!';
                    setTimeout(() => {
                        saveBtn.disabled = false;
                        saveBtn.textContent = 'Save API Key';
                    }, 3000);
                } else {
                    showAlert(data.error || 'Failed to save', 'error');
                    saveBtn.disabled = false;
                    saveBtn.textContent = 'Save API Key';
                }
            } catch (err) {
                showAlert('Failed to save: ' + err.message, 'error');
                saveBtn.disabled = false;
                saveBtn.textContent = 'Save API Key';
            }
        }
    </script>
</body>
</html>`

const settingsPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hivedeck Agent Settings</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f3f4f6;
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
        }
        .header {
            display: flex;
            align-items: center;
            gap: 12px;
            margin-bottom: 30px;
        }
        .header svg {
            width: 40px;
            height: 40px;
            color: #3b82f6;
        }
        h1 {
            color: #1f2937;
            font-size: 28px;
        }
        .card {
            background: white;
            border-radius: 12px;
            box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
            padding: 24px;
            margin-bottom: 20px;
        }
        .card h2 {
            color: #1f2937;
            font-size: 18px;
            margin-bottom: 16px;
            padding-bottom: 12px;
            border-bottom: 1px solid #e5e7eb;
        }
        .form-group {
            margin-bottom: 16px;
        }
        label {
            display: block;
            color: #374151;
            font-size: 14px;
            font-weight: 500;
            margin-bottom: 6px;
        }
        input[type="text"], textarea {
            width: 100%;
            padding: 10px 14px;
            border: 1px solid #d1d5db;
            border-radius: 6px;
            font-size: 14px;
            transition: border-color 0.2s;
        }
        input[type="text"]:focus, textarea:focus {
            outline: none;
            border-color: #3b82f6;
            box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
        }
        textarea {
            font-family: 'Monaco', 'Menlo', monospace;
            resize: vertical;
            min-height: 100px;
        }
        .hint {
            font-size: 12px;
            color: #6b7280;
            margin-top: 4px;
        }
        .btn-row {
            display: flex;
            gap: 12px;
            margin-top: 16px;
        }
        button {
            padding: 10px 20px;
            border: none;
            border-radius: 6px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
        }
        .btn-primary {
            background: #3b82f6;
            color: white;
        }
        .btn-primary:hover {
            background: #2563eb;
        }
        .btn-secondary {
            background: #f3f4f6;
            color: #374151;
            border: 1px solid #d1d5db;
        }
        .btn-secondary:hover {
            background: #e5e7eb;
        }
        .btn-danger {
            background: #ef4444;
            color: white;
        }
        .btn-danger:hover {
            background: #dc2626;
        }
        .status-item {
            display: flex;
            justify-content: space-between;
            padding: 10px 0;
            border-bottom: 1px solid #f3f4f6;
        }
        .status-item:last-child {
            border-bottom: none;
        }
        .status-label {
            color: #6b7280;
            font-size: 14px;
        }
        .status-value {
            color: #1f2937;
            font-size: 14px;
            font-weight: 500;
        }
        .alert {
            padding: 12px 16px;
            border-radius: 8px;
            margin-bottom: 20px;
            font-size: 14px;
        }
        .alert-success {
            background: #d1fae5;
            color: #065f46;
        }
        .alert-error {
            background: #fee2e2;
            color: #991b1b;
        }
        .hidden { display: none; }
        .api-key-display {
            display: flex;
            gap: 8px;
        }
        .api-key-display input {
            flex: 1;
            font-family: 'Monaco', 'Menlo', monospace;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
            </svg>
            <h1>Agent Settings</h1>
        </div>

        <div id="alert" class="alert hidden"></div>

        <div class="card">
            <h2>Agent Status</h2>
            <div id="status">Loading...</div>
        </div>

        <div class="card">
            <h2>API Key</h2>
            <div class="form-group">
                <label>Current API Key</label>
                <div class="api-key-display">
                    <input type="text" id="newApiKey" placeholder="Generate or enter new API key">
                    <button class="btn-secondary" onclick="generateKey()">Generate</button>
                    <button class="btn-secondary" onclick="copyKey()">Copy</button>
                </div>
                <p class="hint">Generate a new key and save to update. Requires agent restart.</p>
            </div>
            <div class="btn-row">
                <button class="btn-primary" onclick="saveApiKey()">Save API Key</button>
            </div>
        </div>

        <div class="card">
            <h2>Allowed Paths</h2>
            <div class="form-group">
                <label>File Browser Paths</label>
                <textarea id="allowedPaths" placeholder="/var/log&#10;/etc&#10;/home"></textarea>
                <p class="hint">One path per line. These directories will be accessible via the file browser.</p>
            </div>
            <div class="btn-row">
                <button class="btn-primary" onclick="savePaths()">Save Paths</button>
            </div>
        </div>

        <div class="card">
            <h2>Allowed Services</h2>
            <div class="form-group">
                <label>Manageable Systemd Services</label>
                <textarea id="allowedServices" placeholder="docker&#10;nginx&#10;ssh"></textarea>
                <p class="hint">One service per line. These services can be started/stopped/restarted.</p>
            </div>
            <div class="btn-row">
                <button class="btn-primary" onclick="saveServices()">Save Services</button>
            </div>
        </div>
    </div>

    <script>
        const alertDiv = document.getElementById('alert');
        const API_KEY = new URLSearchParams(window.location.search).get('key') || '';

        function showAlert(message, type) {
            alertDiv.textContent = message;
            alertDiv.className = 'alert alert-' + type;
            setTimeout(() => alertDiv.className = 'alert hidden', 5000);
        }

        async function fetchWithAuth(url, options = {}) {
            options.headers = options.headers || {};
            options.headers['Authorization'] = 'Bearer ' + API_KEY;
            return fetch(url, options);
        }

        async function loadSettings() {
            try {
                const res = await fetchWithAuth('/api/settings');
                if (!res.ok) {
                    showAlert('Failed to load settings. Check API key.', 'error');
                    return;
                }
                const data = await res.json();

                // Update status
                document.getElementById('status').innerHTML =
                    '<div class="status-item"><span class="status-label">Host</span><span class="status-value">' + data.host + ':' + data.port + '</span></div>' +
                    '<div class="status-item"><span class="status-label">API Key</span><span class="status-value">' + (data.api_key_configured ? 'Configured' : 'Not Set') + '</span></div>' +
                    '<div class="status-item"><span class="status-label">Docker</span><span class="status-value">' + (data.docker_enabled ? 'Enabled' : 'Disabled') + '</span></div>' +
                    '<div class="status-item"><span class="status-label">Log Level</span><span class="status-value">' + data.log_level + '</span></div>' +
                    '<div class="status-item"><span class="status-label">Config File</span><span class="status-value">' + data.env_file + '</span></div>';

                // Update form fields
                document.getElementById('allowedPaths').value = (data.allowed_paths || []).join('\n');
                document.getElementById('allowedServices').value = (data.allowed_services || []).join('\n');
            } catch (err) {
                showAlert('Error loading settings: ' + err.message, 'error');
            }
        }

        async function generateKey() {
            try {
                const res = await fetchWithAuth('/api/settings/generate-key', { method: 'POST' });
                const data = await res.json();
                if (data.api_key) {
                    document.getElementById('newApiKey').value = data.api_key;
                    showAlert('New API key generated. Save to apply.', 'success');
                }
            } catch (err) {
                showAlert('Failed to generate key', 'error');
            }
        }

        function copyKey() {
            const input = document.getElementById('newApiKey');
            if (input.value) {
                navigator.clipboard.writeText(input.value);
                showAlert('API key copied!', 'success');
            }
        }

        async function saveApiKey() {
            const apiKey = document.getElementById('newApiKey').value.trim();
            if (!apiKey || apiKey.length < 32) {
                showAlert('API key must be at least 32 characters', 'error');
                return;
            }
            try {
                const res = await fetchWithAuth('/api/settings/api-key', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ api_key: apiKey })
                });
                const data = await res.json();
                if (res.ok) {
                    showAlert('API key saved! Restart agent to apply.', 'success');
                } else {
                    showAlert(data.error || 'Failed to save', 'error');
                }
            } catch (err) {
                showAlert('Error: ' + err.message, 'error');
            }
        }

        async function savePaths() {
            const paths = document.getElementById('allowedPaths').value.split('\n').map(p => p.trim()).filter(p => p);
            try {
                const res = await fetchWithAuth('/api/settings', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ allowed_paths: paths })
                });
                if (res.ok) {
                    showAlert('Paths saved!', 'success');
                } else {
                    showAlert('Failed to save paths', 'error');
                }
            } catch (err) {
                showAlert('Error: ' + err.message, 'error');
            }
        }

        async function saveServices() {
            const services = document.getElementById('allowedServices').value.split('\n').map(s => s.trim()).filter(s => s);
            try {
                const res = await fetchWithAuth('/api/settings', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ allowed_services: services })
                });
                if (res.ok) {
                    showAlert('Services saved!', 'success');
                } else {
                    showAlert('Failed to save services', 'error');
                }
            } catch (err) {
                showAlert('Error: ' + err.message, 'error');
            }
        }

        // Load settings on page load
        if (API_KEY) {
            loadSettings();
        } else {
            showAlert('API key required. Add ?key=YOUR_API_KEY to URL', 'error');
        }
    </script>
</body>
</html>`
