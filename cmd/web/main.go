package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
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

const (
	cookieName   = "hci_asset_token"
	defaultPort  = "3000"
	defaultAPI   = "http://localhost:8080"
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
		r.Get("/users/{id}/delete", userDeleteConfirm(apiBase))
		r.Post("/users/{id}/delete", userDelete(apiBase))
		r.Get("/scans", scanPage(apiBase))
		r.Post("/scans", startScan(apiBase))
		r.Get("/scans/{id}", scanDetail(apiBase))
		r.Post("/scans/{id}/cancel", cancelScan(apiBase))
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

// requireAuth redirects to /login if cookie is missing or if the API returns 401 (invalid/expired token).
func requireAuth(apiBase string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := r.Cookie(cookieName)
			if err != nil || token.Value == "" {
				http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.Path), http.StatusFound)
				return
			}
			// Validate token with API so expired/invalid tokens send user to login before any page loads.
			_, status, _ := apiGet(apiBase, "/assets?limit=1", token.Value)
			if status == http.StatusUnauthorized {
				clearAuthAndRedirectToLogin(w, r)
				return
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
	renderTemplate(w, "login.html", nil)
}

func loginSubmit(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		username := strings.TrimSpace(r.FormValue("username"))
		if username == "" {
			renderTemplate(w, "login.html", map[string]string{"Error": "Username is required"})
			return
		}

		body := fmt.Sprintf(`{"username":%q}`, username)
		req, _ := http.NewRequest("POST", apiBase+"/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			renderTemplate(w, "login.html", map[string]string{"Error": "Cannot reach API: " + err.Error()})
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
			renderTemplate(w, "login.html", map[string]string{"Error": msg})
			return
		}

		var out struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(data, &out); err != nil || out.Token == "" {
			renderTemplate(w, "login.html", map[string]string{"Error": "Invalid login response"})
			return
		}

		next := r.URL.Query().Get("next")
		if next == "" {
			next = "/dashboard"
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
// Call when the API returns 401 (expired or invalid token) so the user can sign in again.
func clearAuthAndRedirectToLogin(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
	next := r.URL.Path
	if r.URL.RawQuery != "" {
		next += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, "/login?next="+url.QueryEscape(next), http.StatusFound)
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
			renderTemplate(w, "dashboard.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, "dashboard.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var assets []struct {
			ID          int     `json:"id"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			LastSeen    *string `json:"last_seen"`
		}
		if err := json.Unmarshal(data, &assets); err != nil {
			renderTemplate(w, "dashboard.html", map[string]interface{}{"Error": "Invalid assets response"})
			return
		}

		renderTemplate(w, "dashboard.html", map[string]interface{}{
			"AssetCount": len(assets),
			"Assets":    assets,
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

		data, status, err := apiGet(apiBase, path, tok)
		if err != nil {
			renderTemplate(w, "assets.html", map[string]interface{}{"Error": err.Error(), "SearchQuery": search, "Page": page})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, "assets.html", map[string]interface{}{"Error": "API error: " + string(data), "SearchQuery": search, "Page": page})
			return
		}

		var assets []struct {
			ID          int     `json:"id"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			NetworkName string  `json:"network_name"`
			LastSeen    *string `json:"last_seen"`
		}
		if err := json.Unmarshal(data, &assets); err != nil {
			renderTemplate(w, "assets.html", map[string]interface{}{"Error": "Invalid assets response", "SearchQuery": search, "Page": page})
			return
		}

		hasNext := len(assets) == pageSize
		prevPage := 0
		if page > 1 {
			prevPage = page - 1
		}
		nextPage := 0
		if hasNext {
			nextPage = page + 1
		}
		searchEncoded := url.QueryEscape(search)

		renderTemplate(w, "assets.html", map[string]interface{}{
			"Assets":          assets,
			"SearchQuery":     search,
			"SearchEncoded":   searchEncoded,
			"Page":            page,
			"PrevPage":        prevPage,
			"NextPage":        nextPage,
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
			renderTemplate(w, "asset_detail.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, "asset_detail.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var asset struct {
			ID          int     `json:"id"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			NetworkName string  `json:"network_name"`
			LastSeen    *string `json:"last_seen"`
		}
		if err := json.Unmarshal(data, &asset); err != nil {
			renderTemplate(w, "asset_detail.html", map[string]interface{}{"Error": "Invalid asset response"})
			return
		}

		heartbeatError := r.URL.Query().Get("heartbeat_error") == "1"
		renderTemplate(w, "asset_detail.html", map[string]interface{}{
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
			renderTemplate(w, "asset_detail.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
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
		renderTemplate(w, "asset_form.html", map[string]interface{}{
			"FormAction":  "/assets",
			"SubmitLabel": "Create asset",
		})
	}
}

func assetCreate(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		description := strings.TrimSpace(r.FormValue("description"))

		if name == "" || description == "" {
			renderTemplate(w, "asset_form.html", map[string]interface{}{
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

		body := []byte(fmt.Sprintf(`{"name":%q,"description":%q}`, name, description))
		data, status, err := apiPost(apiBase, "/assets", tok, body)
		if err != nil {
			renderTemplate(w, "asset_form.html", map[string]interface{}{
				"Error":       err.Error(),
				"FormAction":  "/assets",
				"SubmitLabel": "Create asset",
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status < http.StatusOK || status >= http.StatusMultipleChoices {
			renderTemplate(w, "asset_form.html", map[string]interface{}{
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
			renderTemplate(w, "asset_form.html", map[string]interface{}{
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
			renderTemplate(w, "asset_form.html", map[string]interface{}{
				"Error": err.Error(),
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, "asset_form.html", map[string]interface{}{
				"Error": "API error: " + string(data),
			})
			return
		}

		var asset struct {
			ID          int     `json:"id"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			NetworkName string  `json:"network_name"`
			LastSeen    *string `json:"last_seen"`
		}
		if err := json.Unmarshal(data, &asset); err != nil {
			renderTemplate(w, "asset_form.html", map[string]interface{}{
				"Error": "Invalid asset response",
			})
			return
		}

		renderTemplate(w, "asset_form.html", map[string]interface{}{
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

		if name == "" || description == "" {
			renderTemplate(w, "asset_form.html", map[string]interface{}{
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

		body := []byte(fmt.Sprintf(`{"name":%q,"description":%q}`, name, description))
		data, status, err := apiPut(apiBase, "/assets/"+id, tok, body)
		if err != nil {
			renderTemplate(w, "asset_form.html", map[string]interface{}{
				"Error":       err.Error(),
				"FormAction":  "/assets/" + id + "/edit",
				"SubmitLabel": "Save changes",
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status < http.StatusOK || status >= http.StatusMultipleChoices {
			renderTemplate(w, "asset_form.html", map[string]interface{}{
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
			renderTemplate(w, "asset_form.html", map[string]interface{}{
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
			renderTemplate(w, "asset_delete_confirm.html", map[string]interface{}{"Error": err.Error(), "AssetID": id})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, "asset_delete_confirm.html", map[string]interface{}{"Error": "Asset not found or API error", "AssetID": id})
			return
		}

		var asset struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(data, &asset); err != nil {
			renderTemplate(w, "asset_delete_confirm.html", map[string]interface{}{"Error": "Invalid asset response", "AssetID": id})
			return
		}

		renderTemplate(w, "asset_delete_confirm.html", map[string]interface{}{
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
			renderTemplate(w, "asset_delete_confirm.html", map[string]interface{}{
				"Error":   err.Error(),
				"AssetID": id,
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
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
		renderTemplate(w, "asset_delete_confirm.html", map[string]interface{}{
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
			renderTemplate(w, "scan.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, "scan.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var scans []struct {
			ID        string `json:"id"`
			Target    string `json:"target"`
			Status    string `json:"status"`
			StartedAt string `json:"started_at"`
		}
		if err := json.Unmarshal(data, &scans); err != nil {
			renderTemplate(w, "scan.html", map[string]interface{}{})
			return
		}

		renderTemplate(w, "scan.html", map[string]interface{}{
			"Scans": scans,
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
			renderTemplate(w, "scan.html", map[string]interface{}{"Error": "Target is required (e.g. 192.168.1.0/24 or hostname)"})
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
			renderTemplate(w, "scan.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status != http.StatusOK {
			var errResp struct{ Error string }
			_ = json.Unmarshal(data, &errResp)
			msg := errResp.Error
			if msg == "" {
				msg = string(data)
			}
			renderTemplate(w, "scan.html", map[string]interface{}{"Error": "API: " + msg})
			return
		}

		var out struct {
			JobID  string `json:"job_id"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &out); err != nil || out.JobID == "" {
			renderTemplate(w, "scan.html", map[string]interface{}{"Error": "Invalid scan response"})
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
			renderTemplate(w, "scan_detail.html", map[string]interface{}{"Error": err.Error(), "JobID": jobID})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status == http.StatusNotFound {
			renderTemplate(w, "scan_detail.html", map[string]interface{}{"Error": "Scan job not found", "JobID": jobID})
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, "scan_detail.html", map[string]interface{}{"Error": "API error: " + string(data), "JobID": jobID})
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
			renderTemplate(w, "scan_detail.html", map[string]interface{}{"Error": "Invalid scan response", "JobID": jobID})
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
		renderTemplate(w, "scan_detail.html", payload)
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
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status != http.StatusOK {
			http.Redirect(w, r, "/scans/"+jobID, http.StatusFound)
			return
		}
		http.Redirect(w, r, "/scans/"+jobID, http.StatusFound)
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
			renderTemplate(w, "users.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, "users.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var users []struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal(data, &users); err != nil {
			renderTemplate(w, "users.html", map[string]interface{}{"Error": "Invalid users response"})
			return
		}

		renderTemplate(w, "users.html", map[string]interface{}{
			"Users": users,
		})
	}
}

func userCreateForm(apiBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "user_form.html", map[string]interface{}{
			"FormAction":  "/users",
			"SubmitLabel": "Create user",
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
		if username == "" {
			renderTemplate(w, "user_form.html", map[string]interface{}{
				"Error":       "Username is required",
				"FormAction":  "/users",
				"SubmitLabel": "Create user",
			})
			return
		}

		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		body := []byte(fmt.Sprintf(`{"username":%q}`, username))
		data, status, err := apiPost(apiBase, "/users", tok, body)
		if err != nil {
			renderTemplate(w, "user_form.html", map[string]interface{}{
				"Error":       err.Error(),
				"FormAction":  "/users",
				"SubmitLabel": "Create user",
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status < http.StatusOK || status >= http.StatusMultipleChoices {
			var errResp struct{ Error string }
			_ = json.Unmarshal(data, &errResp)
			msg := errResp.Error
			if msg == "" {
				msg = string(data)
			}
			renderTemplate(w, "user_form.html", map[string]interface{}{
				"Error":       "API: " + msg,
				"FormAction":  "/users",
				"SubmitLabel": "Create user",
			})
			return
		}

		var user struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(data, &user); err != nil || user.ID == 0 {
			renderTemplate(w, "user_form.html", map[string]interface{}{
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
			renderTemplate(w, "user_form.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, "user_form.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var user struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal(data, &user); err != nil {
			renderTemplate(w, "user_form.html", map[string]interface{}{"Error": "Invalid user response"})
			return
		}

		renderTemplate(w, "user_form.html", map[string]interface{}{
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
		editPayload := func(errMsg string) map[string]interface{} {
			return map[string]interface{}{
				"Error":       errMsg,
				"User":       map[string]interface{}{"Username": username},
				"FormAction":  "/users/" + id + "/edit",
				"SubmitLabel": "Save changes",
			}
		}
		if username == "" {
			renderTemplate(w, "user_form.html", editPayload("Username is required"))
			return
		}

		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		body := []byte(fmt.Sprintf(`{"username":%q}`, username))
		data, status, err := apiPut(apiBase, "/users/"+id, tok, body)
		if err != nil {
			renderTemplate(w, "user_form.html", editPayload(err.Error()))
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status < http.StatusOK || status >= http.StatusMultipleChoices {
			var errResp struct{ Error string }
			_ = json.Unmarshal(data, &errResp)
			msg := errResp.Error
			if msg == "" {
				msg = string(data)
			}
			renderTemplate(w, "user_form.html", editPayload("API: "+msg))
			return
		}

		http.Redirect(w, r, "/users", http.StatusFound)
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
			renderTemplate(w, "user_delete_confirm.html", map[string]interface{}{"Error": err.Error(), "UserID": id})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, "user_delete_confirm.html", map[string]interface{}{"Error": "User not found or API error", "UserID": id})
			return
		}

		var user struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal(data, &user); err != nil {
			renderTemplate(w, "user_delete_confirm.html", map[string]interface{}{"Error": "Invalid user response", "UserID": id})
			return
		}

		renderTemplate(w, "user_delete_confirm.html", map[string]interface{}{
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
			renderTemplate(w, "user_delete_confirm.html", map[string]interface{}{
				"Error":  err.Error(),
				"UserID": id,
			})
			return
		}
		if status == http.StatusUnauthorized {
			clearAuthAndRedirectToLogin(w, r)
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
		renderTemplate(w, "user_delete_confirm.html", map[string]interface{}{
			"Error":  "Delete failed: " + msg,
			"UserID": id,
		})
	}
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	funcs := template.FuncMap{"eq": func(a, b interface{}) bool { return a == b }}
	content, err := templatesFS.ReadFile("templates/" + name)
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
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
