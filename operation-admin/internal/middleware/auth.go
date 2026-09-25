package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/operation-admin/internal/auth"
	"github.com/QuantumNous/new-api/operation-admin/internal/rbac"
	"github.com/gin-gonic/gin"
)

func Auth(svc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
			return
		}
		raw := strings.TrimSpace(h)
		if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			raw = strings.TrimSpace(raw[7:])
		}
		claims, err := svc.Parse(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "登录已失效"})
			return
		}
		c.Set("uid", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("auth_source", claims.AuthSource)
		c.Set("super_admin", claims.SuperAdmin)
		c.Set("claims", claims)
		c.Next()
	}
}

// RootAuth kept as alias for Auth during transition.
func RootAuth(svc *auth.Service) gin.HandlerFunc {
	return Auth(svc)
}

func RequirePerm(rbacSvc *rbac.Service, codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uidVal, _ := c.Get("uid")
		srcVal, _ := c.Get("auth_source")
		uid, _ := uidVal.(uint)
		src, _ := srcVal.(string)
		if !rbacSvc.HasPermission(uid, src, codes...) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "没有权限执行此操作"})
			return
		}
		c.Next()
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
