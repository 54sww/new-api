package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/operation-admin/internal/auth"
	"github.com/QuantumNous/new-api/operation-admin/internal/config"
	"github.com/QuantumNous/new-api/operation-admin/internal/db"
	"github.com/QuantumNous/new-api/operation-admin/internal/handler"
	"github.com/QuantumNous/new-api/operation-admin/internal/middleware"
	"github.com/QuantumNous/new-api/operation-admin/internal/newapi"
	"github.com/QuantumNous/new-api/operation-admin/internal/newapidb"
	"github.com/QuantumNous/new-api/operation-admin/internal/notify"
	"github.com/QuantumNous/new-api/operation-admin/internal/rbac"
	"github.com/QuantumNous/new-api/operation-admin/internal/reconcile"
	syncer "github.com/QuantumNous/new-api/operation-admin/internal/sync"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

//go:embed all:web/dist
var distFS embed.FS

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	if cfg.NewAPIAccessToken == "" {
		log.Println("WARNING: NEW_API_ACCESS_TOKEN is empty; background sync disabled until set")
	}
	if cfg.NewAPILogSQLDSN == "" {
		log.Fatal("NEW_API_SQL_DSN or NEW_API_LOG_SQL_DSN is required (read-only access to new-api logs)")
	}

	gdb, err := db.Open(cfg.SQLDSN)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	if err := rbac.Seed(gdb, cfg.AdminPassword); err != nil {
		log.Fatalf("rbac seed: %v", err)
	}

	newAPIDB, err := newapidb.Open(cfg.NewAPILogSQLDSN)
	if err != nil {
		log.Fatalf("new-api log db open: %v", err)
	}
	log.Printf("new-api consume logs: direct DB read (DSN configured)")

	apiClient := newapi.New(cfg.NewAPIBaseURL, cfg.NewAPIAccessToken)
	rbacSvc := &rbac.Service{DB: gdb}
	authSvc := &auth.Service{
		Secret: cfg.JWTSecret,
		Expire: cfg.JWTExpire,
		Client: apiClient,
		DB:     gdb,
		RBAC:   rbacSvc,
	}
	syncSvc := &syncer.Service{DB: gdb, Client: apiClient, Cfg: cfg, Notify: &notify.Service{DB: gdb}}
	reconSvc := &reconcile.Service{DB: gdb, NewAPIDB: newAPIDB, QuotaPerUnit: cfg.QuotaPerUnit}
	api := &handler.API{DB: gdb, NewAPIDB: newAPIDB, Auth: authSvc, Sync: syncSvc, Reconcile: reconSvc, RBAC: rbacSvc}

	syncSvc.Start()

	r := gin.Default()
	r.Use(middleware.CORS())

	r.GET("/api/health", api.Health)
	r.POST("/api/auth/login", api.Login)
	r.POST("/api/auth/login/2fa", api.Login2FA)

	authed := r.Group("/api")
	authed.Use(middleware.Auth(authSvc))
	{
		authed.GET("/me", api.Me)
		authed.GET("/auth/permissions", api.AuthPermissions)
		authed.GET("/auth/menus", api.AuthMenus)

		authed.GET("/channels", middleware.RequirePerm(rbacSvc, "finance:reconcile:view"), api.ListChannels)
		authed.PUT("/channels/:id/opening-balance", middleware.RequirePerm(rbacSvc, "finance:opening:set"), api.SetOpeningBalance)
		authed.GET("/channels/:id/balance-query", middleware.RequirePerm(rbacSvc, "finance:balance:config"), api.GetBalanceQuery)
		authed.PUT("/channels/:id/balance-query", middleware.RequirePerm(rbacSvc, "finance:balance:config"), api.PutBalanceQuery)
		authed.POST("/channels/:id/balance-query/test", middleware.RequirePerm(rbacSvc, "finance:balance:config"), api.TestBalanceQuery)
		authed.POST("/channels/:id/balance-refresh", middleware.RequirePerm(rbacSvc, "finance:balance:refresh"), api.RefreshChannelBalance)
		authed.GET("/balance-query/presets", middleware.RequirePerm(rbacSvc, "finance:balance:config"), api.BalanceQueryPresets)
		authed.GET("/recharges", middleware.RequirePerm(rbacSvc, "finance:reconcile:view"), api.ListRecharges)
		authed.POST("/recharges", middleware.RequirePerm(rbacSvc, "finance:recharge:create"), api.CreateRecharge)
		authed.DELETE("/recharges/:id", middleware.RequirePerm(rbacSvc, "finance:recharge:delete"), api.DeleteRecharge)
		authed.GET("/reconcile/live", middleware.RequirePerm(rbacSvc, "finance:reconcile:view"), api.LiveReconcile)
		authed.GET("/sync/status", middleware.RequirePerm(rbacSvc, "finance:reconcile:view"), api.SyncStatus)
		authed.POST("/sync/:kind", middleware.RequirePerm(rbacSvc, "finance:reconcile:sync"), api.TriggerSync)
		authed.GET("/settings/balance-sync", middleware.RequirePerm(rbacSvc, "finance:sync-settings:view", "finance:sync-settings:edit"), api.GetBalanceSyncSettings)
		authed.PUT("/settings/balance-sync", middleware.RequirePerm(rbacSvc, "finance:sync-settings:edit"), api.PutBalanceSyncSettings)
		authed.GET("/reports/daily", middleware.RequirePerm(rbacSvc, "finance:daily:view"), api.DailyReport)
		authed.GET("/reports/daily/snapshots", middleware.RequirePerm(rbacSvc, "finance:daily:view"), api.DailyReportSnapshots)
		authed.GET("/logs", middleware.RequirePerm(rbacSvc, "finance:reconcile:view"), api.ListConsumeLogs)
		authed.GET("/external-logs", middleware.RequirePerm(rbacSvc, "finance:reconcile:view"), api.ListExternalLogs)
		authed.POST("/external-logs", middleware.RequirePerm(rbacSvc, "finance:reconcile:sync"), api.UpsertExternalLogs)

		authed.GET("/system/menus/tree", middleware.RequirePerm(rbacSvc, "system:menu:list", "system:role:assign"), api.ListMenus)
		authed.POST("/system/menus", middleware.RequirePerm(rbacSvc, "system:menu:create"), api.CreateMenu)
		authed.PUT("/system/menus/:id", middleware.RequirePerm(rbacSvc, "system:menu:edit"), api.UpdateMenu)
		authed.DELETE("/system/menus/:id", middleware.RequirePerm(rbacSvc, "system:menu:delete"), api.DeleteMenu)

		authed.GET("/system/roles", middleware.RequirePerm(rbacSvc, "system:role:list"), api.ListRoles)
		authed.GET("/system/roles/:id", middleware.RequirePerm(rbacSvc, "system:role:list"), api.GetRole)
		authed.POST("/system/roles", middleware.RequirePerm(rbacSvc, "system:role:create"), api.CreateRole)
		authed.PUT("/system/roles/:id", middleware.RequirePerm(rbacSvc, "system:role:edit"), api.UpdateRole)
		authed.DELETE("/system/roles/:id", middleware.RequirePerm(rbacSvc, "system:role:delete"), api.DeleteRole)
		authed.PUT("/system/roles/:id/menus", middleware.RequirePerm(rbacSvc, "system:role:assign"), api.AssignRoleMenus)

		authed.GET("/system/users", middleware.RequirePerm(rbacSvc, "system:user:list"), api.ListUsers)
		authed.POST("/system/users", middleware.RequirePerm(rbacSvc, "system:user:create"), api.CreateUser)
		authed.PUT("/system/users/:id", middleware.RequirePerm(rbacSvc, "system:user:edit"), api.UpdateUser)
		authed.DELETE("/system/users/:id", middleware.RequirePerm(rbacSvc, "system:user:delete"), api.DeleteUser)

		authed.GET("/system/notify-channels", middleware.RequirePerm(rbacSvc, "system:notify-channel:list", "system:notify-group:list", "system:notify-group:edit", "system:notify-group:create"), api.ListNotifyChannels)
		authed.GET("/system/notify-channels/:id", middleware.RequirePerm(rbacSvc, "system:notify-channel:list"), api.GetNotifyChannel)
		authed.POST("/system/notify-channels", middleware.RequirePerm(rbacSvc, "system:notify-channel:create"), api.CreateNotifyChannel)
		authed.PUT("/system/notify-channels/:id", middleware.RequirePerm(rbacSvc, "system:notify-channel:edit"), api.UpdateNotifyChannel)
		authed.DELETE("/system/notify-channels/:id", middleware.RequirePerm(rbacSvc, "system:notify-channel:delete"), api.DeleteNotifyChannel)

		authed.GET("/system/notify-groups", middleware.RequirePerm(rbacSvc, "system:notify-group:list", "system:notify-rule:list", "system:notify-rule:create", "system:notify-rule:edit"), api.ListNotifyGroups)
		authed.GET("/system/notify-groups/:id", middleware.RequirePerm(rbacSvc, "system:notify-group:list"), api.GetNotifyGroup)
		authed.POST("/system/notify-groups", middleware.RequirePerm(rbacSvc, "system:notify-group:create"), api.CreateNotifyGroup)
		authed.PUT("/system/notify-groups/:id", middleware.RequirePerm(rbacSvc, "system:notify-group:edit"), api.UpdateNotifyGroup)
		authed.DELETE("/system/notify-groups/:id", middleware.RequirePerm(rbacSvc, "system:notify-group:delete"), api.DeleteNotifyGroup)

		authed.GET("/system/notify-rules", middleware.RequirePerm(rbacSvc, "system:notify-rule:list"), api.ListNotifyRules)
		authed.POST("/system/notify-rules", middleware.RequirePerm(rbacSvc, "system:notify-rule:create"), api.CreateNotifyRule)
		authed.PUT("/system/notify-rules/:id", middleware.RequirePerm(rbacSvc, "system:notify-rule:edit"), api.UpdateNotifyRule)
		authed.DELETE("/system/notify-rules/:id", middleware.RequirePerm(rbacSvc, "system:notify-rule:delete"), api.DeleteNotifyRule)
	}

	webRoot, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		log.Fatalf("web dist: %v", err)
	}
	fileServer := http.FileServer(http.FS(webRoot))
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "not found"})
			return
		}
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		f, err := webRoot.Open(path)
		if err != nil {
			index, err2 := webRoot.Open("index.html")
			if err2 != nil {
				c.String(http.StatusInternalServerError, "frontend not built: cd web && npm run build")
				return
			}
			defer index.Close()
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Status(http.StatusOK)
			_, _ = io.Copy(c.Writer, index)
			return
		}
		_ = f.Close()
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	addr := ":" + cfg.Port
	log.Printf("operation-admin listening on %s (new-api=%s)", addr, cfg.NewAPIBaseURL)
	if err := r.Run(addr); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
