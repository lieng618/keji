package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed all:web/dist
var webAssets embed.FS

func main() {
	addr := flag.String("addr", envOr("ADDR", ":8080"), "HTTP listen address")
	dbPath := flag.String("db", envOr("DB_PATH", "data/student-manager.db"), "SQLite database path")
	flag.Parse()

	app, err := NewApp(*dbPath)
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}
	defer app.Close()

	assets, err := fs.Sub(webAssets, "web/dist")
	if err != nil {
		log.Fatalf("读取前端资源失败: %v", err)
	}

	handler := securityHeaders(requestLogger(app.Routes(assets)))
	server := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	log.Printf("课迹已启动: http://localhost%s", displayAddr(*addr))
	log.Printf("数据库: %s", *dbPath)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP 服务退出: %v", err)
	}
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func displayAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return addr
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return strings.TrimPrefix(addr, "0.0.0.0")
	}
	return addr
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; connect-src 'self' ws: wss:; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; img-src 'self' data: blob:; media-src 'self' blob:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
