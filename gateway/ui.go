package gateway

//go:generate sh -c "rm -rf ui_dist && cp -r ../ui/dist ui_dist"

import (
	"bytes"
	"crypto/subtle"
	"embed"
	"net/http"
	"strings"
	"time"
)

// UI 静态文件嵌入
//
//go:embed ui_dist
var uiDist embed.FS

// UIStaticHandler 创建 UI 静态文件处理器
// 使用 embed.FS 提供打包在程序中的静态文件
func UIStaticHandler() http.Handler {
	// 创建 FileServer，托管 uiDist 中的文件
	fs := http.FileServer(http.FS(uiDist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 处理 /ui 根路径，重定向到 /ui/
		if path == "/ui" {
			http.Redirect(w, r, "/ui/index.html", http.StatusMovedPermanently)
			return
		}

		// 处理 /ui/ 开头的路径
		if len(path) >= 4 && path[:4] == "/ui/" {
			// 去掉 /ui 前缀，让 FileServer 从根目录找文件
			r.URL.Path = path[3:] // 变成 /xx
		} else if path == "/ui/" {
			// /ui/ -> /index.html
			r.URL.Path = "/index.html"
		}
		r.URL.Path = "/ui_dist/" + r.URL.Path
		fs.ServeHTTP(w, r)
	})
}

// TeamConsoleHandler serves only immutable application code. Project data and
// mutations remain behind the short-lived browser session, project RBAC and
// CSRF checks in /auth/session and /rpc.
func TeamConsoleHandler() http.Handler {
	fs := http.FileServer(http.FS(uiDist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self'; "+
				"img-src 'self' data:; connect-src 'self' ws: wss:; "+
				"frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")

		path := r.URL.Path
		switch {
		case path == "/dashboard":
			http.Redirect(w, r, "/dashboard/", http.StatusMovedPermanently)
			return
		case path == "/dashboard/":
			w.Header().Set("Cache-Control", "no-store")
			index, err := uiDist.ReadFile("ui_dist/index.html")
			if err != nil {
				http.Error(w, "team console unavailable", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
			return
		case strings.HasPrefix(path, "/dashboard/"):
			r.URL.Path = strings.TrimPrefix(path, "/dashboard")
		case strings.HasPrefix(path, "/assets/"):
			r.URL.Path = path
		default:
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/" {
			r.URL.Path = "/index.html"
		}
		if r.URL.Path == "/index.html" {
			w.Header().Set("Cache-Control", "no-store")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		r.URL.Path = "/ui_dist" + r.URL.Path
		fs.ServeHTTP(w, r)
	})
}

// DashboardAuthConfig Dashboard 认证配置
type DashboardAuthConfig struct {
	RequireAuth bool   // 是否需要认证
	AuthToken   string // 认证 token
}

// DashboardHandler 创建 Dashboard 静态文件处理器
// 用于 WebSocket 服务器端口，路径为 /dashboard
func DashboardHandler(authConfig *DashboardAuthConfig) http.Handler {
	// 创建 FileServer，托管 uiDist 中的文件
	fs := http.FileServer(http.FS(uiDist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 如果需要认证，检查 token
		if authConfig != nil && authConfig.RequireAuth {
			// 检查客户端是否是本地访问
			if isLocalRequest(r) {
				http.SetCookie(w, &http.Cookie{
					Name:     "dashboard_token",
					Value:    authConfig.AuthToken,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
				})
				// 本地访问，跳过认证
				goto serveDashboard
			}

			// 从查询参数获取 token
			token := r.URL.Query().Get("token")
			if token == "" {
				// 从 Cookie 获取 token
				if cookie, err := r.Cookie("dashboard_token"); err == nil {
					token = cookie.Value
				}
			}
			if token == "" {
				// 从 Authorization header 获取
				auth := r.Header.Get("Authorization")
				if auth != "" {
					// 支持 "Bearer <token>" 格式
					if len(auth) > 7 && auth[:7] == "Bearer " {
						token = auth[7:]
					}
				}
			}

			// 验证 token
			if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(authConfig.AuthToken)) != 1 {
				// 返回 401 未授权
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "unauthorized", "message": "Dashboard access requires valid token. Add ?token=YOUR_TOKEN to the URL."}`))
				return
			}

			// 设置 cookie 以便后续请求无需再次验证
			http.SetCookie(w, &http.Cookie{
				Name:     "dashboard_token",
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				Secure:   r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"),
				SameSite: http.SameSiteStrictMode,
			})
		}

	serveDashboard:
		path := r.URL.Path

		// 处理 /dashboard 根路径，重定向到 /dashboard/
		if path == "/dashboard" {
			http.Redirect(w, r, "/dashboard/", http.StatusMovedPermanently)
			return
		}

		// 处理 /dashboard/ 开头的路径
		if len(path) >= 11 && path[:11] == "/dashboard/" {
			// 去掉 /dashboard 前缀，让 FileServer 从根目录找文件
			r.URL.Path = path[10:] // 变成 /xx
		} else if path == "/dashboard/" {
			// /dashboard/ -> /index.html
			r.URL.Path = "/index.html"
		}

		// 处理 assets 路径
		if len(path) >= 7 && path[:7] == "/assets" {
			r.URL.Path = path
		}

		r.URL.Path = "/ui_dist" + r.URL.Path
		fs.ServeHTTP(w, r)
	})
}

// isLocalRequest 检查请求是否来自本地
func isLocalRequest(r *http.Request) bool {
	// 获取客户端地址
	host := r.RemoteAddr
	// 去除端口号
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	// 去除 IPv6 的方括号
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")

	remoteLocal := host == "127.0.0.1" || host == "::1" || host == "localhost"
	if !remoteLocal {
		return false
	}

	// A local reverse proxy must provide the real client address. Evaluate it
	// before granting the loopback shortcut so a remote caller through Caddy
	// does not inherit the proxy's local address.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// 取第一个 IP
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			firstIP := strings.TrimSpace(ips[0])
			return firstIP == "127.0.0.1" || firstIP == "::1" || firstIP == "localhost"
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri == "127.0.0.1" || xri == "::1" || xri == "localhost"
	}

	return true
}
