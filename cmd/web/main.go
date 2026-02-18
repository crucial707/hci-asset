package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed templates
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

const (
	cookieName   = "hci_asset_token"
	defaultPort  = "3000"
	defaultAPI   = "http://localhost:8080/v1"
	envWebPort   = "HCI_WEB_PORT"
	envAPIURL    = "HCI_ASSET_API_URL"
)

func main() {
	port := getEnv(envWebPort, defaultPort)
	apiBase := getEnv(envAPIURL, defaultAPI)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health (no auth, no templates)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Static assets (logo, etc.)
	staticRoot, _ := fs.Sub(staticFS, "static")
	r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.FS(staticRoot))))

	// Public
	r.Get("/login", loginForm)
	r.Post("/login", loginSubmit(apiBase))
	r.Get("/logout", logout)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(requireAuth(apiBase))
		r.Get("/", redirectDashboard)
		r.Get("/dashboard", dashboard(apiBase))
		r.Get("/assets/new", assetCreateForm(apiBase))
		r.Post("/assets", assetCreate(apiBase))
		r.Get("/assets", assetsList(apiBase))
		r.Get("/assets/{id}", assetDetail(apiBase))
		r.Post("/assets/{id}/heartbeat", assetHeartbeat(apiBase))
		r.Get("/assets/{id}/edit", assetEditForm(apiBase))
		r.Post("/assets/{id}/edit", assetUpdate(apiBase))
		r.Get("/assets/{id}/delete", assetDeleteConfirm(apiBase))
		r.Post("/assets/{id}/delete", assetDelete(apiBase))
		r.Get("/users", usersList(apiBase))
		r.Get("/users/new", userCreateForm(apiBase))
		r.Post("/users", userCreate(apiBase))
		r.Get("/users/{id}/edit", userEditForm(apiBase))
		r.Post("/users/{id}/edit", userUpdate(apiBase))
		r.Get("/users/{id}/change-password", userChangePasswordForm(apiBase))
		r.Post("/users/{id}/change-password", userChangePassword(apiBase))
		r.Get("/users/{id}/delete", userDeleteConfirm(apiBase))
		r.Post("/users/{id}/delete", userDelete(apiBase))
		r.Get("/scans", scanPage(apiBase))
		r.Post("/scans", startScan(apiBase))
		r.Post("/scans/clear", clearScans(apiBase))
		r.Get("/scans/{id}", scanDetail(apiBase))
		r.Post("/scans/{id}/cancel", cancelScan(apiBase))
		r.Get("/saved-scans", savedScansList(apiBase))
		r.Get("/saved-scans/new", savedScanNewForm(apiBase))
		r.Post("/saved-scans", savedScanCreate(apiBase))
		r.Get("/saved-scans/{id}/edit", savedScanEditForm(apiBase))
		r.Post("/saved-scans/{id}/edit", savedScanUpdate(apiBase))
		r.Post("/saved-scans/{id}/run", savedScanRun(apiBase))
		r.Get("/saved-scans/{id}/delete", savedScanDeleteConfirm(apiBase))
		r.Post("/saved-scans/{id}/delete", savedScanDelete(apiBase))
		r.Get("/schedules", schedulesList(apiBase))
		r.Get("/schedules/new", scheduleCreateForm(apiBase))
		r.Post("/schedules", scheduleCreate(apiBase))
		r.Get("/schedules/{id}/edit", scheduleEditForm(apiBase))
		r.Post("/schedules/{id}/edit", scheduleUpdate(apiBase))
		r.Get("/schedules/{id}/delete", scheduleDeleteConfirm(apiBase))
		r.Post("/schedules/{id}/delete", scheduleDelete(apiBase))
		r.Get("/audit", auditList(apiBase))
		r.Get("/network", networkPage(apiBase))
	})

	log.Printf("Web UI running on http://localhost:%s (API: %s)", port, apiBase)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// formatDuration returns a human-readable duration (e.g. "1m30s", "45s").
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Second {
		return "0s"
	}
	return d.String()
}

// contextKey type for request context keys.
type contextKey int

const userContextKey contextKey = 0

// currentUser is stored in context for session display in the layout.
type currentUser struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

// requireAuth redirects to /login if cookie is missing or if the API returns 401 (invalid/expired token).
// On success, fetches current user from GET /me and stores in context for layout (Logged in as ...).
func requireAuth(apiBase string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := r.Cookie(cookieName)
			if err != nil || token.Value == "" {
				http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.Path), http.StatusFound)
				return
			}
			data, status, _ := apiGet(apiBase, "/me", token.Value)
			if status == http.StatusUnauthorized {
				clearAuthAndRedirectToLogin(w, r, "Session expired. Please sign in again.")
				return
			}
			if status == http.StatusOK {
				var u struct {
					Username string `json:"username"`
					Role     string `json:"role"`
				}
				if json.Unmarshal(data, &u) == nil {
					r = r.WithContext(context.WithValue(r.Context(), userContextKey, &currentUser{Username: u.Username, Role: u.Role}))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func redirectDashboard(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func loginForm(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(cookieName); err == nil {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	data := map[string]interface{}{}
	if msg := r.URL.Query().Get("msg"); msg != "" {
		data["Message"] = msg
	}
	renderTemplate(w, r, "login.html", data)
}

func loginSubmit(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		username := strings.TrimSpace(r.FormValue("username"))
		if username == "" {
			renderTemplate(w, r, "login.html", map[string]string{"Error": "Username is required"})
			return
		}
		password := r.FormValue("password")

		var body string
		if password != "" {
			body = fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
		} else {
			body = fmt.Sprintf(`{"username":%q}`, username)
		}
		req, _ := http.NewRequest("POST", apiBase+"/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			renderTemplate(w, r, "login.html", map[string]string{"Error": "Cannot reach API: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		data, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			var errResp struct{ Error string }
			_ = json.Unmarshal(data, &errResp)
			msg := errResp.Error
			if msg == "" {
				msg = string(data)
			}
			renderTemplate(w, r, "login.html", map[string]string{"Error": msg})
			return
		}

		var out struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(data, &out); err != nil || out.Token == "" {
			renderTemplate(w, r, "login.html", map[string]string{"Error": "Invalid login response"})
			return
		}

		next := r.URL.Query().Get("next")
		if next == "" {
			next = "/dashboard"
		}
		// Append #main so layout can focus main content for accessibility after login
		if !strings.Contains(next, "#") {
			next = next + "#main"
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    out.Token,
			Path:     "/",
			MaxAge:   24 * 3600,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, next, http.StatusFound)
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusFound)
}

// clearAuthAndRedirectToLogin clears the token cookie and redirects to login with next=current path.
// If msg is non-empty, adds msg= to the login URL so the login page can show it (e.g. "Session expired").
func clearAuthAndRedirectToLogin(w http.ResponseWriter, r *http.Request, msg string) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
	next := r.URL.Path
	if r.URL.RawQuery != "" {
		next = next + "?" + r.URL.RawQuery
	}
	loc := "/login?next=" + url.QueryEscape(next)
	if msg != "" {
		loc += "&msg=" + url.QueryEscape(msg)
	}
	http.Redirect(w, r, loc, http.StatusFound)
}

// apiGet performs GET to API with token from request cookie.
func apiGet(apiBase, path, token string) ([]byte, int, error) {
	req, _ := http.NewRequest("GET", apiBase+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

// apiPost performs POST to API with token and JSON body.
func apiPost(apiBase, path, token string, body []byte) ([]byte, int, error) {
	req, _ := http.NewRequest("POST", apiBase+path, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

// apiPut performs PUT to API with token and JSON body.
func apiPut(apiBase, path, token string, body []byte) ([]byte, int, error) {
	req, _ := http.NewRequest("PUT", apiBase+path, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

// apiDelete performs DELETE to API with token.
func apiDelete(apiBase, path, token string) ([]byte, int, error) {
	req, _ := http.NewRequest("DELETE", apiBase+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

func dashboard(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/assets?limit=1000", tok)
		if err != nil {
			renderTemplate(w, r, "dashboard.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "dashboard.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var listResp struct {
			Items []struct {
				ID          int     `json:"id"`
				Name        string  `json:"name"`
				Description string  `json:"description"`
				LastSeen    *string `json:"last_seen"`
			} `json:"items"`
			Total int `json:"total"`
		}
		if err := json.Unmarshal(data, &listResp); err != nil {
			renderTemplate(w, r, "dashboard.html", map[string]interface{}{"Error": "Invalid assets response"})
			return
		}

		renderTemplate(w, r, "dashboard.html", map[string]interface{}{
			"AssetCount": listResp.Total,
			"Assets":    listResp.Items,
		})
	}
}

func assetsList(apiBase string) http.HandlerFunc {
	const pageSize = 20
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		search := strings.TrimSpace(r.URL.Query().Get("search"))
		tagFilter := strings.TrimSpace(r.URL.Query().Get("tag"))
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				page = n
			}
		}
		offset := (page - 1) * pageSize

		path := fmt.Sprintf("/assets?limit=%d&offset=%d", pageSize, offset)
		if search != "" {
			path += "&search=" + url.QueryEscape(search)
		}
		if tagFilter != "" {
			path += "&tag=" + url.QueryEscape(tagFilter)
		}

		data, status, err := apiGet(apiBase, path, tok)
		if err != nil {
			renderTemplate(w, r, "assets.html", map[string]interface{}{"Error": err.Error(), "SearchQuery": search, "TagFilter": tagFilter, "Page": page})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "assets.html", map[string]interface{}{"Error": "API error: " + string(data), "SearchQuery": search, "TagFilter": tagFilter, "Page": page})
			return
		}

		var listResp struct {
			Items  []struct {
				ID          int      `json:"id"`
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Tags        []string `json:"tags"`
				NetworkName string   `json:"network_name"`
				LastSeen    *string  `json:"last_seen"`
			} `json:"items"`
			Total  int `json:"total"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}
		if err := json.Unmarshal(data, &listResp); err != nil {
			renderTemplate(w, r, "assets.html", map[string]interface{}{"Error": "Invalid assets response", "SearchQuery": search, "TagFilter": tagFilter, "Page": page})
			return
		}
		assets := listResp.Items

		hasNext := listResp.Offset+len(listResp.Items) < listResp.Total
		prevPage := 0
		if page > 1 {
			prevPage = page - 1
		}
		nextPage := 0
		if hasNext {
			nextPage = page + 1
		}
		searchEncoded := url.QueryEscape(search)
		tagEncoded := url.QueryEscape(tagFilter)

		renderTemplate(w, r, "assets.html", map[string]interface{}{
			"Assets":        assets,
			"SearchQuery":   search,
			"SearchEncoded": searchEncoded,
			"TagFilter":     tagFilter,
			"TagEncoded":    tagEncoded,
			"Page":          page,
			"PrevPage":      prevPage,
			"NextPage":      nextPage,
		})
	}
}

func assetDetail(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/assets/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "asset_detail.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "asset_detail.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var asset struct {
			ID          int      `json:"id"`
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
			NetworkName string   `json:"network_name"`
			LastSeen    *string  `json:"last_seen"`
		}
		if err := json.Unmarshal(data, &asset); err != nil {
			renderTemplate(w, r, "asset_detail.html", map[string]interface{}{"Error": "Invalid asset response"})
			return
		}

		heartbeatError := r.URL.Query().Get("heartbeat_error") == "1"
		renderTemplate(w, r, "asset_detail.html", map[string]interface{}{
			"Asset":         asset,
			"HeartbeatError": heartbeatError,
		})
	}
}

func assetHeartbeat(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		_, status, err := apiPost(apiBase, "/assets/"+id+"/heartbeat", tok, []byte("{}"))
		if err != nil {
			renderTemplate(w, r, "asset_detail.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			// Redirect back to detail; page will show error via API on next load, or we could pass a query param
			http.Redirect(w, r, "/assets/"+id+"?heartbeat_error=1", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/assets/"+id, http.StatusFound)
	}
}

// ====== Asset create/edit (Web UI) ======

func assetCreateForm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, r, "asset_form.html", map[string]interface{}{
			"FormAction":  "/assets",
			"SubmitLabel": "Create asset",
		})
	}
}

func parseTagsFromForm(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func assetCreate(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		description := strings.TrimSpace(r.FormValue("description"))
		tags := parseTagsFromForm(r.FormValue("tags"))

		if name == "" || description == "" {
			renderTemplate(w, r, "asset_form.html", map[string]interface{}{
				"Error":       "Name and description are required",
				"FormAction":  "/assets",
				"SubmitLabel": "Create asset",
			})
			return
		}

		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		payload := map[string]interface{}{"name": name, "description": description}
		if len(tags) > 0 {
			payload["tags"] = tags
		}
		body, _ := json.Marshal(payload)
		data, status, err := apiPost(apiBase, "/assets", tok, body)
		if err != nil {
			renderTemplate(w, r, "asset_form.html", map[string]interface{}{
				"Error":       err.Error(),
				"FormAction":  "/assets",
				"SubmitLabel": "Create asset",
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status < http.StatusOK || status >= http.StatusMultipleChoices {
			renderTemplate(w, r, "asset_form.html", map[string]interface{}{
				"Error":       "API error: " + string(data),
				"FormAction":  "/assets",
				"SubmitLabel": "Create asset",
			})
			return
		}

		var asset struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(data, &asset); err != nil || asset.ID == 0 {
			renderTemplate(w, r, "asset_form.html", map[string]interface{}{
				"Error":       "Invalid create asset response",
				"FormAction":  "/assets",
				"SubmitLabel": "Create asset",
			})
			return
		}

		http.Redirect(w, r, "/assets/"+fmt.Sprint(asset.ID), http.StatusFound)
	}
}

func assetEditForm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/assets/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "asset_form.html", map[string]interface{}{
				"Error": err.Error(),
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "asset_form.html", map[string]interface{}{
				"Error": "API error: " + string(data),
			})
			return
		}

		var asset struct {
			ID          int      `json:"id"`
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
			NetworkName string   `json:"network_name"`
			LastSeen    *string  `json:"last_seen"`
		}
		if err := json.Unmarshal(data, &asset); err != nil {
			renderTemplate(w, r, "asset_form.html", map[string]interface{}{
				"Error": "Invalid asset response",
			})
			return
		}

		renderTemplate(w, r, "asset_form.html", map[string]interface{}{
			"Asset":       asset,
			"FormAction":  "/assets/" + id + "/edit",
			"SubmitLabel": "Save changes",
		})
	}
}

func assetUpdate(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		description := strings.TrimSpace(r.FormValue("description"))
		tags := parseTagsFromForm(r.FormValue("tags"))

		if name == "" || description == "" {
			renderTemplate(w, r, "asset_form.html", map[string]interface{}{
				"Error":       "Name and description are required",
				"FormAction":  "/assets/" + id + "/edit",
				"SubmitLabel": "Save changes",
			})
			return
		}

		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		payload := map[string]interface{}{"name": name, "description": description, "tags": tags}
		body, _ := json.Marshal(payload)
		data, status, err := apiPut(apiBase, "/assets/"+id, tok, body)
		if err != nil {
			renderTemplate(w, r, "asset_form.html", map[string]interface{}{
				"Error":       err.Error(),
				"FormAction":  "/assets/" + id + "/edit",
				"SubmitLabel": "Save changes",
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status < http.StatusOK || status >= http.StatusMultipleChoices {
			renderTemplate(w, r, "asset_form.html", map[string]interface{}{
				"Error":       "API error: " + string(data),
				"FormAction":  "/assets/" + id + "/edit",
				"SubmitLabel": "Save changes",
			})
			return
		}

		var asset struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(data, &asset); err != nil || asset.ID == 0 {
			renderTemplate(w, r, "asset_form.html", map[string]interface{}{
				"Error":       "Invalid update asset response",
				"FormAction":  "/assets/" + id + "/edit",
				"SubmitLabel": "Save changes",
			})
			return
		}

		http.Redirect(w, r, "/assets/"+fmt.Sprint(asset.ID), http.StatusFound)
	}
}

func assetDeleteConfirm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/assets/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "asset_delete_confirm.html", map[string]interface{}{"Error": err.Error(), "AssetID": id})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "asset_delete_confirm.html", map[string]interface{}{"Error": "Asset not found or API error", "AssetID": id})
			return
		}

		var asset struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(data, &asset); err != nil {
			renderTemplate(w, r, "asset_delete_confirm.html", map[string]interface{}{"Error": "Invalid asset response", "AssetID": id})
			return
		}

		renderTemplate(w, r, "asset_delete_confirm.html", map[string]interface{}{
			"Asset": asset,
		})
	}
}

func assetDelete(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		body, status, err := apiDelete(apiBase, "/assets/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "asset_delete_confirm.html", map[string]interface{}{
				"Error":   err.Error(),
				"AssetID": id,
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status == http.StatusNoContent {
			http.Redirect(w, r, "/assets", http.StatusFound)
			return
		}

		msg := string(body)
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		renderTemplate(w, r, "asset_delete_confirm.html", map[string]interface{}{
			"Error":   "Delete failed: " + msg,
			"AssetID": id,
		})
	}
}

func scanPage(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/scans", tok)
		if err != nil {
			renderTemplate(w, r, "scan.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "scan.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var listResp struct {
			Items []struct {
				ID        int    `json:"id"`
				Target    string `json:"target"`
				Status    string `json:"status"`
				StartedAt string `json:"started_at"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &listResp); err != nil {
			renderTemplate(w, r, "scan.html", map[string]interface{}{})
			return
		}

		renderTemplate(w, r, "scan.html", map[string]interface{}{
			"Scans": listResp.Items,
		})
	}
}

func startScan(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		target := strings.TrimSpace(r.FormValue("target"))
		if target == "" {
			renderTemplate(w, r, "scan.html", map[string]interface{}{"Error": "Target is required (e.g. 192.168.1.0/24 or hostname)"})
			return
		}

		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		body := []byte(fmt.Sprintf(`{"target":%q}`, target))
		data, status, err := apiPost(apiBase, "/scans", tok, body)
		if err != nil {
			renderTemplate(w, r, "scan.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			var errResp struct{ Error string }
			_ = json.Unmarshal(data, &errResp)
			msg := errResp.Error
			if msg == "" {
				msg = string(data)
			}
			renderTemplate(w, r, "scan.html", map[string]interface{}{"Error": "API: " + msg})
			return
		}

		var out struct {
			JobID  string `json:"job_id"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &out); err != nil || out.JobID == "" {
			renderTemplate(w, r, "scan.html", map[string]interface{}{"Error": "Invalid scan response"})
			return
		}

		http.Redirect(w, r, "/scans/"+out.JobID, http.StatusFound)
	}
}

func scanDetail(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/scans/"+jobID, tok)
		if err != nil {
			renderTemplate(w, r, "scan_detail.html", map[string]interface{}{"Error": err.Error(), "JobID": jobID})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status == http.StatusNotFound {
			renderTemplate(w, r, "scan_detail.html", map[string]interface{}{"Error": "Scan job not found", "JobID": jobID})
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "scan_detail.html", map[string]interface{}{"Error": "API error: " + string(data), "JobID": jobID})
			return
		}

		var job struct {
			Target      string     `json:"target"`
			Status      string     `json:"status"`
			StartedAt   time.Time  `json:"started_at"`
			CompletedAt *time.Time `json:"completed_at"`
			Error       string     `json:"error"`
			Assets      []struct {
				ID          int    `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				NetworkName string `json:"network_name"`
			} `json:"assets"`
		}
		if err := json.Unmarshal(data, &job); err != nil {
			renderTemplate(w, r, "scan_detail.html", map[string]interface{}{"Error": "Invalid scan response", "JobID": jobID})
			return
		}

		var elapsed, duration string
		if !job.StartedAt.IsZero() {
			end := time.Now()
			if job.CompletedAt != nil {
				end = *job.CompletedAt
			}
			d := end.Sub(job.StartedAt).Round(time.Second)
			if job.Status == "running" {
				elapsed = formatDuration(d)
			} else {
				duration = formatDuration(d)
			}
		}

		payload := map[string]interface{}{
			"JobID":   jobID,
			"Job":     job,
			"Elapsed": elapsed,
			"Duration": duration,
			"AutoRefresh": job.Status == "running",
		}
		renderTemplate(w, r, "scan_detail.html", payload)
	}
}

func cancelScan(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		_, status, err := apiPost(apiBase, "/scans/"+jobID+"/cancel", tok, []byte("{}"))
		if err != nil {
			http.Redirect(w, r, "/scans/"+jobID+"?error="+url.QueryEscape(err.Error()), http.StatusFound)
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			http.Redirect(w, r, "/scans/"+jobID, http.StatusFound)
			return
		}
		http.Redirect(w, r, "/scans/"+jobID, http.StatusFound)
	}
}

func clearScans(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}
		_, status, err := apiDelete(apiBase, "/scans", tok)
		if err != nil {
			http.Redirect(w, r, "/scans?error="+url.QueryEscape(err.Error()), http.StatusFound)
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusNoContent {
			http.Redirect(w, r, "/scans?error=Failed+to+clear+scans", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/scans", http.StatusFound)
	}
}

// ====== Saved Scans (Web UI) ======

func savedScansList(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}
		data, status, err := apiGet(apiBase, "/saved-scans", tok)
		if err != nil {
			renderTemplate(w, r, "saved_scans.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "saved_scans.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}
		var listResp struct {
			Items []struct {
				ID        int    `json:"id"`
				Name      string `json:"name"`
				Target    string `json:"target"`
				CreatedAt string `json:"created_at"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &listResp); err != nil {
			renderTemplate(w, r, "saved_scans.html", map[string]interface{}{"Error": "Invalid response"})
			return
		}
		renderTemplate(w, r, "saved_scans.html", map[string]interface{}{
			"SavedScans": listResp.Items,
		})
	}
}

func savedScanNewForm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, r, "saved_scan_form.html", map[string]interface{}{
			"FormAction":  "/saved-scans",
			"SubmitLabel": "Save scan",
		})
	}
}

func savedScanCreate(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		target := strings.TrimSpace(r.FormValue("target"))
		if name == "" || target == "" {
			renderTemplate(w, r, "saved_scan_form.html", map[string]interface{}{
				"Error":       "Name and target are required",
				"FormAction":  "/saved-scans",
				"SubmitLabel": "Save scan",
				"Name":        name,
				"Target":      target,
			})
			return
		}
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}
		body, _ := json.Marshal(map[string]string{"name": name, "target": target})
		data, status, err := apiPost(apiBase, "/saved-scans", tok, body)
		if err != nil {
			renderTemplate(w, r, "saved_scan_form.html", map[string]interface{}{
				"Error": err.Error(), "FormAction": "/saved-scans", "SubmitLabel": "Save scan", "Name": name, "Target": target,
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status < 200 || status >= 300 {
			var errResp struct{ Error string }
			_ = json.Unmarshal(data, &errResp)
			msg := errResp.Error
			if msg == "" {
				msg = string(data)
			}
			renderTemplate(w, r, "saved_scan_form.html", map[string]interface{}{
				"Error": "API: " + msg, "FormAction": "/saved-scans", "SubmitLabel": "Save scan", "Name": name, "Target": target,
			})
			return
		}
		http.Redirect(w, r, "/saved-scans", http.StatusFound)
	}
}

func savedScanEditForm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}
		data, status, err := apiGet(apiBase, "/saved-scans/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "saved_scan_form.html", map[string]interface{}{"Error": err.Error(), "FormAction": "/saved-scans/" + id + "/edit", "SubmitLabel": "Save changes"})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "saved_scan_form.html", map[string]interface{}{"Error": "Saved scan not found", "FormAction": "/saved-scans/" + id + "/edit", "SubmitLabel": "Save changes"})
			return
		}
		var saved struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Target string `json:"target"`
		}
		if err := json.Unmarshal(data, &saved); err != nil {
			renderTemplate(w, r, "saved_scan_form.html", map[string]interface{}{"Error": "Invalid response"})
			return
		}
		renderTemplate(w, r, "saved_scan_form.html", map[string]interface{}{
			"Saved":      saved,
			"FormAction": "/saved-scans/" + id + "/edit",
			"SubmitLabel": "Save changes",
		})
	}
}

func savedScanUpdate(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		target := strings.TrimSpace(r.FormValue("target"))
		if name == "" || target == "" {
			renderTemplate(w, r, "saved_scan_form.html", map[string]interface{}{
				"Error": "Name and target are required",
				"Saved": map[string]interface{}{"ID": id, "Name": name, "Target": target},
				"FormAction": "/saved-scans/" + id + "/edit",
				"SubmitLabel": "Save changes",
			})
			return
		}
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}
		body, _ := json.Marshal(map[string]string{"name": name, "target": target})
		data, status, err := apiPut(apiBase, "/saved-scans/"+id, tok, body)
		if err != nil {
			renderTemplate(w, r, "saved_scan_form.html", map[string]interface{}{
				"Error": err.Error(), "Saved": map[string]interface{}{"ID": id, "Name": name, "Target": target},
				"FormAction": "/saved-scans/" + id + "/edit", "SubmitLabel": "Save changes",
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status < 200 || status >= 300 {
			var errResp struct{ Error string }
			_ = json.Unmarshal(data, &errResp)
			msg := errResp.Error
			if msg == "" {
				msg = string(data)
			}
			renderTemplate(w, r, "saved_scan_form.html", map[string]interface{}{
				"Error": "API: " + msg, "Saved": map[string]interface{}{"ID": id, "Name": name, "Target": target},
				"FormAction": "/saved-scans/" + id + "/edit", "SubmitLabel": "Save changes",
			})
			return
		}
		http.Redirect(w, r, "/saved-scans", http.StatusFound)
	}
}

func savedScanRun(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}
		data, status, err := apiPost(apiBase, "/saved-scans/"+id+"/run", tok, []byte("{}"))
		if err != nil {
			http.Redirect(w, r, "/saved-scans?error="+url.QueryEscape(err.Error()), http.StatusFound)
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			http.Redirect(w, r, "/saved-scans?error=Failed+to+start+scan", http.StatusFound)
			return
		}
		var out struct {
			JobID string `json:"job_id"`
		}
		if err := json.Unmarshal(data, &out); err == nil && out.JobID != "" {
			http.Redirect(w, r, "/scans/"+out.JobID, http.StatusFound)
			return
		}
		http.Redirect(w, r, "/scans", http.StatusFound)
	}
}

func savedScanDeleteConfirm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}
		data, status, err := apiGet(apiBase, "/saved-scans/"+id, tok)
		if err != nil || status != http.StatusOK {
			renderTemplate(w, r, "saved_scan_delete_confirm.html", map[string]interface{}{"Error": "Saved scan not found", "Saved": map[string]interface{}{"ID": id, "Name": "", "Target": ""}})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		var saved struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Target string `json:"target"`
		}
		if err := json.Unmarshal(data, &saved); err != nil {
			renderTemplate(w, r, "saved_scan_delete_confirm.html", map[string]interface{}{"Error": "Invalid response"})
			return
		}
		renderTemplate(w, r, "saved_scan_delete_confirm.html", map[string]interface{}{"Saved": saved})
	}
}

func savedScanDelete(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}
		_, status, err := apiDelete(apiBase, "/saved-scans/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "saved_scan_delete_confirm.html", map[string]interface{}{"Error": err.Error(), "Saved": map[string]interface{}{"ID": id}})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status == http.StatusNoContent {
			http.Redirect(w, r, "/saved-scans", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/saved-scans", http.StatusFound)
	}
}

// ====== Schedules (Web UI) ======

func schedulesList(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/schedules?limit=100", tok)
		if err != nil {
			renderTemplate(w, r, "schedules.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "schedules.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var listResp struct {
			Items []struct {
				ID        int       `json:"id"`
				Target    string    `json:"target"`
				CronExpr  string    `json:"cron_expr"`
				Enabled   bool      `json:"enabled"`
				CreatedAt time.Time `json:"created_at"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &listResp); err != nil {
			renderTemplate(w, r, "schedules.html", map[string]interface{}{"Error": "Invalid schedules response"})
			return
		}

		renderTemplate(w, r, "schedules.html", map[string]interface{}{"Schedules": listResp.Items})
	}
}

func scheduleCreateForm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, r, "schedule_form.html", map[string]interface{}{
			"FormAction":  "/schedules",
			"SubmitLabel": "Create schedule",
		})
	}
}

func scheduleCreate(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		target := strings.TrimSpace(r.FormValue("target"))
		cronExpr := strings.TrimSpace(r.FormValue("cron_expr"))
		enabled := r.FormValue("enabled") == "1"

		if target == "" || cronExpr == "" {
			renderTemplate(w, r, "schedule_form.html", map[string]interface{}{
				"Error":       "Target and cron expression are required",
				"FormAction":  "/schedules",
				"SubmitLabel": "Create schedule",
			})
			return
		}

		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		payload := map[string]interface{}{"target": target, "cron_expr": cronExpr, "enabled": enabled}
		body, _ := json.Marshal(payload)
		data, status, err := apiPost(apiBase, "/schedules", tok, body)
		if err != nil {
			renderTemplate(w, r, "schedule_form.html", map[string]interface{}{
				"Error":       err.Error(),
				"FormAction":  "/schedules",
				"SubmitLabel": "Create schedule",
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusCreated && status != http.StatusOK {
			renderTemplate(w, r, "schedule_form.html", map[string]interface{}{
				"Error":       "API error: " + string(data),
				"FormAction":  "/schedules",
				"SubmitLabel": "Create schedule",
			})
			return
		}

		http.Redirect(w, r, "/schedules", http.StatusFound)
	}
}

func scheduleEditForm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/schedules/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "schedule_form.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "schedule_form.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var schedule struct {
			ID        int       `json:"id"`
			Target    string    `json:"target"`
			CronExpr  string    `json:"cron_expr"`
			Enabled   bool      `json:"enabled"`
			CreatedAt time.Time `json:"created_at"`
		}
		if err := json.Unmarshal(data, &schedule); err != nil {
			renderTemplate(w, r, "schedule_form.html", map[string]interface{}{"Error": "Invalid schedule response"})
			return
		}

		renderTemplate(w, r, "schedule_form.html", map[string]interface{}{
			"Schedule":    schedule,
			"FormAction":  "/schedules/" + id + "/edit",
			"SubmitLabel": "Save changes",
		})
	}
}

func scheduleUpdate(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		target := strings.TrimSpace(r.FormValue("target"))
		cronExpr := strings.TrimSpace(r.FormValue("cron_expr"))
		enabled := r.FormValue("enabled") == "1"

		if target == "" || cronExpr == "" {
			renderTemplate(w, r, "schedule_form.html", map[string]interface{}{
				"Error":       "Target and cron expression are required",
				"FormAction":  "/schedules/" + id + "/edit",
				"SubmitLabel": "Save changes",
			})
			return
		}

		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		payload := map[string]interface{}{"target": target, "cron_expr": cronExpr, "enabled": enabled}
		body, _ := json.Marshal(payload)
		_, status, err := apiPut(apiBase, "/schedules/"+id, tok, body)
		if err != nil {
			renderTemplate(w, r, "schedule_form.html", map[string]interface{}{
				"Error":       err.Error(),
				"FormAction":  "/schedules/" + id + "/edit",
				"SubmitLabel": "Save changes",
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			http.Redirect(w, r, "/schedules/"+id+"/edit", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/schedules", http.StatusFound)
	}
}

func scheduleDeleteConfirm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/schedules/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "schedule_delete_confirm.html", map[string]interface{}{"Error": err.Error(), "ScheduleID": id})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "schedule_delete_confirm.html", map[string]interface{}{"Error": "Schedule not found or API error", "ScheduleID": id})
			return
		}

		var schedule struct {
			ID       int    `json:"id"`
			Target   string `json:"target"`
			CronExpr string `json:"cron_expr"`
		}
		if err := json.Unmarshal(data, &schedule); err != nil {
			renderTemplate(w, r, "schedule_delete_confirm.html", map[string]interface{}{"Error": "Invalid schedule response", "ScheduleID": id})
			return
		}

		renderTemplate(w, r, "schedule_delete_confirm.html", map[string]interface{}{"Schedule": schedule})
	}
}

func scheduleDelete(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		_, status, err := apiDelete(apiBase, "/schedules/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "schedule_delete_confirm.html", map[string]interface{}{"Error": err.Error(), "ScheduleID": id})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status == http.StatusNoContent || status == http.StatusOK {
			http.Redirect(w, r, "/schedules", http.StatusFound)
			return
		}
		renderTemplate(w, r, "schedule_delete_confirm.html", map[string]interface{}{"Error": "Delete failed", "ScheduleID": id})
	}
}

// ====== Audit log (Web UI) ======

func auditList(apiBase string) http.HandlerFunc {
	const defaultLimit = 50
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		limit := defaultLimit
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		offset := 0
		if o := r.URL.Query().Get("offset"); o != "" {
			if n, err := strconv.Atoi(o); err == nil && n >= 0 {
				offset = n
			}
		}

		path := fmt.Sprintf("/audit?limit=%d&offset=%d", limit, offset)
		data, status, err := apiGet(apiBase, path, tok)
		if err != nil {
			renderTemplate(w, r, "audit.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "audit.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var listResp struct {
			Items  []struct {
				ID           int       `json:"id"`
				UserID       int       `json:"user_id"`
				Action       string    `json:"action"`
				ResourceType string    `json:"resource_type"`
				ResourceID   int       `json:"resource_id"`
				Details      string    `json:"details"`
				CreatedAt    time.Time `json:"created_at"`
			} `json:"items"`
			Total  int `json:"total"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}
		if err := json.Unmarshal(data, &listResp); err != nil {
			renderTemplate(w, r, "audit.html", map[string]interface{}{"Error": "Invalid audit response"})
			return
		}
		entries := listResp.Items

		hasNext := listResp.Offset+len(listResp.Items) < listResp.Total
		prevOffset := 0
		if offset > 0 {
			prevOffset = offset - limit
			if prevOffset < 0 {
				prevOffset = 0
			}
		}
		nextOffset := offset + limit

		renderTemplate(w, r, "audit.html", map[string]interface{}{
			"Entries":    entries,
			"Limit":      limit,
			"Offset":     offset,
			"PrevOffset": prevOffset,
			"NextOffset": nextOffset,
			"HasPrev":    offset > 0,
			"HasNext":    hasNext,
		})
	}
}

// ====== Network map (Web UI) ======

func networkPage(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/network/graph", tok)
		if err != nil {
			renderTemplate(w, r, "network.html", map[string]interface{}{"Error": err.Error(), "BodyClass": "network-viz", "GraphJSON": template.JS("{}")})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "network.html", map[string]interface{}{"Error": "API error: " + string(data), "BodyClass": "network-viz", "GraphJSON": template.JS("{}")})
			return
		}

		renderTemplate(w, r, "network.html", map[string]interface{}{
			"BodyClass": "network-viz",
			"GraphJSON": template.JS(data),
		})
	}
}

// ====== Users (Web UI) ======

func usersList(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/users", tok)
		if err != nil {
			renderTemplate(w, r, "users.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "users.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var listResp struct {
			Items []struct {
				ID       int    `json:"id"`
				Username string `json:"username"`
				Role     string `json:"role"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &listResp); err != nil {
			renderTemplate(w, r, "users.html", map[string]interface{}{"Error": "Invalid users response"})
			return
		}

		renderTemplate(w, r, "users.html", map[string]interface{}{
			"Users": listResp.Items,
		})
	}
}

func userCreateForm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, r, "user_form.html", map[string]interface{}{
			"FormAction":  "/users",
			"SubmitLabel": "Create user",
			"IsCreate":    true,
		})
	}
}

func userCreate(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		username := strings.TrimSpace(r.FormValue("username"))
		role := strings.TrimSpace(r.FormValue("role"))
		if role == "" {
			role = "viewer"
		}
		if username == "" {
			renderTemplate(w, r, "user_form.html", map[string]interface{}{
				"Error":       "Username is required",
				"FormAction":  "/users",
				"SubmitLabel": "Create user",
				"IsCreate":    true,
			})
			return
		}

		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		body := []byte(fmt.Sprintf(`{"username":%q,"role":%q}`, username, role))
		data, status, err := apiPost(apiBase, "/users", tok, body)
		if err != nil {
			renderTemplate(w, r, "user_form.html", map[string]interface{}{
				"Error":       err.Error(),
				"FormAction":  "/users",
				"SubmitLabel": "Create user",
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status < http.StatusOK || status >= http.StatusMultipleChoices {
			var errResp struct {
				Error  string            `json:"error"`
				Fields map[string]string `json:"fields"`
			}
			_ = json.Unmarshal(data, &errResp)
			msg := errResp.Error
			if msg == "" {
				msg = string(data)
			}
			payload := map[string]interface{}{
				"Error":       "API: " + msg,
				"FormAction":  "/users",
				"SubmitLabel": "Create user",
				"IsCreate":    true,
			}
			if len(errResp.Fields) > 0 {
				payload["Fields"] = errResp.Fields
			}
			renderTemplate(w, r, "user_form.html", payload)
			return
		}

		var user struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(data, &user); err != nil || user.ID == 0 {
			renderTemplate(w, r, "user_form.html", map[string]interface{}{
				"Error":       "Invalid create user response",
				"FormAction":  "/users",
				"SubmitLabel": "Create user",
			})
			return
		}

		http.Redirect(w, r, "/users", http.StatusFound)
	}
}

func userEditForm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/users/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "user_form.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "user_form.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var user struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		}
		if err := json.Unmarshal(data, &user); err != nil {
			renderTemplate(w, r, "user_form.html", map[string]interface{}{"Error": "Invalid user response"})
			return
		}

		renderTemplate(w, r, "user_form.html", map[string]interface{}{
			"User":        user,
			"FormAction":  "/users/" + id + "/edit",
			"SubmitLabel": "Save changes",
		})
	}
}

func userUpdate(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		username := strings.TrimSpace(r.FormValue("username"))
		role := strings.TrimSpace(r.FormValue("role"))
		editPayload := func(errMsg string) map[string]interface{} {
			return map[string]interface{}{
				"Error":       errMsg,
				"User":       map[string]interface{}{"ID": id, "Username": username, "Role": role},
				"FormAction":  "/users/" + id + "/edit",
				"SubmitLabel": "Save changes",
			}
		}
		if username == "" {
			renderTemplate(w, r, "user_form.html", editPayload("Username is required"))
			return
		}

		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		body := []byte(fmt.Sprintf(`{"username":%q,"role":%q}`, username, role))
		data, status, err := apiPut(apiBase, "/users/"+id, tok, body)
		if err != nil {
			renderTemplate(w, r, "user_form.html", editPayload(err.Error()))
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status < http.StatusOK || status >= http.StatusMultipleChoices {
			var errResp struct {
				Error  string            `json:"error"`
				Fields map[string]string `json:"fields"`
			}
			_ = json.Unmarshal(data, &errResp)
			msg := errResp.Error
			if msg == "" {
				msg = string(data)
			}
			payload := editPayload("API: " + msg)
			if len(errResp.Fields) > 0 {
				payload["Fields"] = errResp.Fields
			}
			renderTemplate(w, r, "user_form.html", payload)
			return
		}

		http.Redirect(w, r, "/users", http.StatusFound)
	}
}

func userChangePasswordForm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/users/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "user_change_password.html", map[string]interface{}{"Error": err.Error(), "User": map[string]interface{}{"ID": id, "Username": ""}})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "user_change_password.html", map[string]interface{}{"Error": "User not found or API error", "User": map[string]interface{}{"ID": id, "Username": ""}})
			return
		}

		var user struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal(data, &user); err != nil {
			renderTemplate(w, r, "user_change_password.html", map[string]interface{}{"Error": "Invalid user response", "User": map[string]interface{}{"ID": id, "Username": ""}})
			return
		}

		renderTemplate(w, r, "user_change_password.html", map[string]interface{}{
			"User": user,
		})
	}
}

func userChangePassword(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		currentPassword := r.FormValue("current_password")
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_password")

		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		payload := func(errMsg string) map[string]interface{} {
			user := map[string]interface{}{"ID": id, "Username": ""}
			if data, st, _ := apiGet(apiBase, "/users/"+id, tok); st == http.StatusOK {
				var u struct { ID int `json:"id"`; Username string `json:"username"` }
				_ = json.Unmarshal(data, &u)
				user["ID"], user["Username"] = u.ID, u.Username
			}
			return map[string]interface{}{"Error": errMsg, "User": user}
		}

		if newPassword == "" {
			renderTemplate(w, r, "user_change_password.html", payload("New password is required"))
			return
		}
		if newPassword != confirmPassword {
			renderTemplate(w, r, "user_change_password.html", payload("New password and confirmation do not match"))
			return
		}

		body, _ := json.Marshal(map[string]string{
			"current_password": currentPassword,
			"new_password":     newPassword,
		})
		data, status, err := apiPut(apiBase, "/users/"+id+"/password", tok, body)
		if err != nil {
			renderTemplate(w, r, "user_change_password.html", payload(err.Error()))
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusNoContent {
			var errResp struct{ Error string }
			_ = json.Unmarshal(data, &errResp)
			msg := errResp.Error
			if msg == "" {
				msg = string(data)
			}
			renderTemplate(w, r, "user_change_password.html", payload("API: "+msg))
			return
		}

		http.Redirect(w, r, "/users/"+id+"/edit", http.StatusFound)
	}
}

func userDeleteConfirm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/users/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "user_delete_confirm.html", map[string]interface{}{"Error": err.Error(), "UserID": id})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, r, "user_delete_confirm.html", map[string]interface{}{"Error": "User not found or API error", "UserID": id})
			return
		}

		var user struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal(data, &user); err != nil {
			renderTemplate(w, r, "user_delete_confirm.html", map[string]interface{}{"Error": "Invalid user response", "UserID": id})
			return
		}

		renderTemplate(w, r, "user_delete_confirm.html", map[string]interface{}{
			"User": user,
		})
	}
}

func userDelete(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		body, status, err := apiDelete(apiBase, "/users/"+id, tok)
		if err != nil {
			renderTemplate(w, r, "user_delete_confirm.html", map[string]interface{}{
				"Error":  err.Error(),
				"UserID": id,
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r, "")
			return
		}
		if status == http.StatusNoContent {
			http.Redirect(w, r, "/users", http.StatusFound)
			return
		}

		msg := string(body)
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		renderTemplate(w, r, "user_delete_confirm.html", map[string]interface{}{
			"Error":  "Delete failed: " + msg,
			"UserID": id,
		})
	}
}

func renderTemplate(w http.ResponseWriter, r *http.Request, name string, data interface{}) {
	funcs := template.FuncMap{"eq": func(a, b interface{}) bool { return a == b }}
	content, err := templatesFS.ReadFile("templates/" + name)
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	// Merge current user from context into data for layout (Logged in as ...).
	if r != nil {
		if u := r.Context().Value(userContextKey); u != nil {
			if m, ok := data.(map[string]interface{}); ok {
				m2 := make(map[string]interface{})
				for k, v := range m {
					m2[k] = v
				}
				m2["User"] = u
				data = m2
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if name == "login.html" {
		t := template.Must(template.New("").Funcs(funcs).Parse(string(content)))
		_ = t.ExecuteTemplate(w, "login", data)
		return
	}

	layout, _ := templatesFS.ReadFile("templates/layout.html")
	t := template.Must(template.New("").Funcs(funcs).Parse(string(layout)))
	t = template.Must(t.New("").Parse(string(content)))
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("template execute: %v", err)
	}
}
