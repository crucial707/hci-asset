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
	// #region agent log
	func() {
		_ = os.MkdirAll("c:/Users/AB/Code/New folder/hci-asset/.cursor", 0755)
		f, err := os.OpenFile("c:/Users/AB/Code/New folder/hci-asset/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			fmt.Fprintln(f, `{"hypothesisId":"H-W1","location":"cmd/web/main.go:main","message":"web main entered","data":{},"timestamp":`+strconv.FormatInt(time.Now().UnixMilli(), 10)+`}`)
			f.Close()
		}
	}()
	// #endregion

	port := getEnv(envWebPort, defaultPort)
	apiBase := getEnv(envAPIURL, defaultAPI)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health (no auth, no templates)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// #region agent log
		func() {
			_ = os.MkdirAll("c:/Users/AB/Code/New folder/hci-asset/.cursor", 0755)
			f, err := os.OpenFile("c:/Users/AB/Code/New folder/hci-asset/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				fmt.Fprintln(f, `{"hypothesisId":"H-W3","location":"cmd/web/main.go:/health","message":"health handler hit","data":{},"timestamp":`+strconv.FormatInt(time.Now().UnixMilli(), 10)+`}`)
				f.Close()
			}
		}()
		// #endregion
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
		r.Get("/assets", assetsList(apiBase))
		r.Get("/assets/{id}", assetDetail(apiBase))
	})

	log.Printf("Web UI running on http://localhost:%s (API: %s)", port, apiBase)
	// #region agent log
	func() {
		_ = os.MkdirAll("c:/Users/AB/Code/New folder/hci-asset/.cursor", 0755)
		f, err := os.OpenFile("c:/Users/AB/Code/New folder/hci-asset/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			fmt.Fprintln(f, `{"hypothesisId":"H-W2","location":"cmd/web/main.go:ListenAndServe","message":"about to listen","data":{"port":"`+port+`"},"timestamp":`+strconv.FormatInt(time.Now().UnixMilli(), 10)+`}`)
			f.Close()
		}
	}()
	// #endregion
	if err := http.ListenAndServe(":"+port, r); err != nil {
		// #region agent log
		func() {
			_ = os.MkdirAll("c:/Users/AB/Code/New folder/hci-asset/.cursor", 0755)
			f, err2 := os.OpenFile("c:/Users/AB/Code/New folder/hci-asset/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err2 == nil {
				b, _ := json.Marshal(err.Error())
				fmt.Fprintln(f, `{"hypothesisId":"H-W2","location":"cmd/web/main.go:ListenAndServe","message":"ListenAndServe failed","data":{"error":`+string(b)+`},"timestamp":`+strconv.FormatInt(time.Now().UnixMilli(), 10)+`}`)
				f.Close()
			}
		}()
		// #endregion
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// requireAuth redirects to /login if cookie is missing or invalid.
func requireAuth(apiBase string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := r.Cookie(cookieName)
			if err != nil || token.Value == "" {
				http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.Path), http.StatusFound)
				return
			}
			// Optional: validate token by calling API (e.g. GET /users/me). For now we just require presence.
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
		if status != http.StatusOK {
			renderTemplate(w, "dashboard.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var assets []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
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
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := r.Cookie(cookieName)
		tok := ""
		if token != nil {
			tok = token.Value
		}

		data, status, err := apiGet(apiBase, "/assets?limit=500", tok)
		if err != nil {
			renderTemplate(w, "assets.html", map[string]interface{}{"Error": err.Error()})
			return
		}
		if status != http.StatusOK {
			renderTemplate(w, "assets.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var assets []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			NetworkName string `json:"network_name"`
			CreatedAt   string `json:"created_at"`
		}
		if err := json.Unmarshal(data, &assets); err != nil {
			renderTemplate(w, "assets.html", map[string]interface{}{"Error": "Invalid assets response"})
			return
		}

		renderTemplate(w, "assets.html", map[string]interface{}{
			"Assets": assets,
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
		if status != http.StatusOK {
			renderTemplate(w, "asset_detail.html", map[string]interface{}{"Error": "API error: " + string(data)})
			return
		}

		var asset struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			NetworkName string `json:"network_name"`
			CreatedAt   string `json:"created_at"`
		}
		if err := json.Unmarshal(data, &asset); err != nil {
			renderTemplate(w, "asset_detail.html", map[string]interface{}{"Error": "Invalid asset response"})
			return
		}

		renderTemplate(w, "asset_detail.html", map[string]interface{}{
			"Asset": asset,
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
