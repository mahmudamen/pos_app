# Go POS SaaS — Backend Specification
## Version: v1.0 | Target: Ubuntu 22.04/24.04 LTS + PostgreSQL 16

---

## 1. Architecture Overview

### 1.1 Stack
| Layer | Technology |
|-------|-----------|
| Language | Go 1.23+ |
| Web Framework | Gin v1.10+ (or Echo v4.12+) |
| ORM/Query | sqlx + scany (recommended) OR GORM v2 |
| Database | PostgreSQL 16 |
| Cache | Redis 7 (sessions, rate limiting, sync tokens) |
| Auth | JWT (golang-jwt/v5) + bcrypt |
| Validation | go-playground/validator/v10 |
| Config | Viper |
| Logging | slog (stdlib) + structured JSON |
| Metrics | Prometheus client + Grafana |
| Deployment | systemd + Caddy (or Docker Compose) |

### 1.2 Project Structure (Standard Go Layout)
```
pos-backend/
├── cmd/
│   └── api/
│       └── main.go                 # Entry point
├── internal/
│   ├── config/
│   │   └── config.go               # Viper: env vars, .env, defaults
│   ├── domain/
│   │   ├── models/
│   │   │   ├── user.go
│   │   │   ├── product.go
│   │   │   ├── sale.go
│   │   │   ├── tenant.go
│   │   │   └── token.go
│   │   └── repositories/
│   │       ├── user_repository.go      # Interface
│   │       ├── product_repository.go   # Interface
│   │       └── sale_repository.go      # Interface
│   ├── infrastructure/
│   │   ├── database/
│   │   │   ├── postgres.go         # Connection pool + migrations
│   │   │   └── migrations/
│   │   │       ├── 001_init.up.sql
│   │   │       ├── 001_init.down.sql
│   │   │       └── ...
│   │   ├── cache/
│   │   │   └── redis.go            # Redis client wrapper
│   │   ├── http/
│   │   │   ├── server.go           # Gin engine setup
│   │   │   ├── router.go           # Route definitions + middleware chain
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go         # JWT validation + tenant extraction
│   │   │   │   ├── cors.go
│   │   │   │   ├── rate_limit.go   # Per-tenant + per-device limits
│   │   │   │   ├── tenant.go       # X-Tenant-ID validation
│   │   │   │   ├── request_id.go   # X-Request-ID propagation
│   │   │   │   ├── logger.go       # Structured access logs
│   │   │   │   └── security.go     # Security headers + fingerprint check
│   │   │   └── handlers/
│   │   │       ├── auth_handler.go
│   │   │       ├── product_handler.go
│   │   │       ├── sale_handler.go
│   │   │       └── sync_handler.go
│   │   └── security/
│   │       ├── password.go         # bcrypt wrapper
│   │       ├── jwt.go              # Token generation/validation
│   │       ├── session.go          # Session management
│   │       └── crypto.go           # AES encryption for sensitive data
│   ├── usecases/
│   │   ├── auth_usecase.go
│   │   ├── product_usecase.go
│   │   ├── sale_usecase.go
│   │   └── sync_usecase.go
│   └── dto/
│       ├── auth_dto.go
│       ├── product_dto.go
│       ├── sale_dto.go
│       └── sync_dto.go
├── pkg/
│   ├── errors/
│   │   └── app_errors.go           # Domain error types
│   ├── validators/
│   │   └── custom_validators.go    # go-playground extensions
│   ├── pagination/
│   │   └── pagination.go
│   └── response/
│       └── json_response.go        # Standard API response envelope
├── scripts/
│   ├── migrate.sh
│   └── deploy.sh
├── deployments/
│   ├── systemd/
│   │   └── pos-api.service
│   ├── caddy/
│   │   └── Caddyfile
│   └── docker/
│       ├── Dockerfile
│       └── docker-compose.yml
├── .env.example
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 2. Authentication & Session Management

### 2.1 Auth Flow (OAuth2 + JWT)

```
┌─────────────┐     POST /auth/login      ┌─────────────┐
│   Flutter   │ ────────────────────────> │   Go API    │
│   Client    │  {email, password,        │   Ubuntu    │
│             │   tenant_id, device_id}   │  PostgreSQL │
└─────────────┘                           └─────────────┘
     ^                                          │
     │     200 OK {access_token, refresh_token, │
     │     expires_in, user, tenant}            │
     └──────────────────────────────────────────┘
```

### 2.2 JWT Token Design

```go
// internal/infrastructure/security/jwt.go
package security

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

const (
    AccessTokenExpiry  = 15 * time.Minute
    RefreshTokenExpiry = 7 * 24 * time.Hour  // 7 days
    TokenIssuer        = "pos-saas-api"
)

type TokenClaims struct {
    jwt.RegisteredClaims
    UserID    string   `json:"user_id"`
    Email     string   `json:"email"`
    TenantID  string   `json:"tenant_id"`
    DeviceID  string   `json:"device_id"`
    SessionID string   `json:"session_id"`
    Role      string   `json:"role"`
    Permissions []string `json:"permissions"`
}

func GenerateTokenPair(user *domain.User, deviceID string, sessionID string) (*domain.TokenPair, error) {
    now := time.Now().UTC()

    // Access Token
    accessClaims := TokenClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   user.ID,
            Issuer:    TokenIssuer,
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenExpiry)),
            ID:        uuid.New().String(),
        },
        UserID:      user.ID,
        Email:       user.Email,
        TenantID:    user.TenantID,
        DeviceID:    deviceID,
        SessionID:   sessionID,
        Role:        user.Role,
        Permissions: user.Permissions,
    }
    accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    accessString, err := accessToken.SignedString([]byte(config.JWTAccessSecret))
    if err != nil {
        return nil, err
    }

    // Refresh Token
    refreshClaims := jwt.RegisteredClaims{
        Subject:   sessionID,
        Issuer:    TokenIssuer,
        IssuedAt:  jwt.NewNumericDate(now),
        ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenExpiry)),
        ID:        uuid.New().String(),
    }
    refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    refreshString, err := refreshToken.SignedString([]byte(config.JWTRefreshSecret))
    if err != nil {
        return nil, err
    }

    return &domain.TokenPair{
        AccessToken:  accessString,
        RefreshToken: refreshString,
        ExpiresIn:    int(AccessTokenExpiry.Seconds()),
    }, nil
}
```

### 2.3 Session Management (Redis-Backed)

```go
// internal/infrastructure/security/session.go
package security

import (
    "context"
    "fmt"
    "time"
    "github.com/redis/go-redis/v9"
)

type SessionManager struct {
    redis *redis.Client
}

const (
    MaxSessionsPerUser = 3
    SessionPrefix      = "session:"
    UserSessionsPrefix = "user_sessions:"
)

func (sm *SessionManager) CreateSession(ctx context.Context, session *domain.Session) error {
    // Check session limit
    userKey := fmt.Sprintf("%s%s", UserSessionsPrefix, session.UserID)
    currentSessions, err := sm.redis.SCard(ctx, userKey).Result()
    if err != nil {
        return err
    }

    // If limit reached, invalidate oldest session
    if currentSessions >= MaxSessionsPerUser {
        oldest, err := sm.redis.SPop(ctx, userKey).Result()
        if err == nil {
            sm.redis.Del(ctx, fmt.Sprintf("%s%s", SessionPrefix, oldest))
        }
    }

    // Store session
    sessionKey := fmt.Sprintf("%s%s", SessionPrefix, session.ID)
    pipe := sm.redis.Pipeline()
    pipe.HSet(ctx, sessionKey, map[string]interface{}{
        "user_id":    session.UserID,
        "tenant_id":  session.TenantID,
        "device_id":  session.DeviceID,
        "device_name": session.DeviceName,
        "ip_address": session.IPAddress,
        "fingerprint": session.Fingerprint,
        "created_at": session.CreatedAt.Format(time.RFC3339),
        "last_active": time.Now().UTC().Format(time.RFC3339),
    })
    pipe.Expire(ctx, sessionKey, RefreshTokenExpiry)
    pipe.SAdd(ctx, userKey, session.ID)
    pipe.Expire(ctx, userKey, RefreshTokenExpiry)
    _, err = pipe.Exec(ctx)

    return err
}

func (sm *SessionManager) ValidateSession(ctx context.Context, sessionID, deviceID, fingerprint string) bool {
    sessionKey := fmt.Sprintf("%s%s", SessionPrefix, sessionID)
    data, err := sm.redis.HGetAll(ctx, sessionKey).Result()
    if err != nil || len(data) == 0 {
        return false
    }

    // Verify device binding
    if data["device_id"] != deviceID || data["fingerprint"] != fingerprint {
        return false
    }

    // Update last active
    sm.redis.HSet(ctx, sessionKey, "last_active", time.Now().UTC().Format(time.RFC3339))
    return true
}

func (sm *SessionManager) RevokeSession(ctx context.Context, sessionID string) error {
    sessionKey := fmt.Sprintf("%s%s", SessionPrefix, sessionID)
    data, err := sm.redis.HGetAll(ctx, sessionKey).Result()
    if err != nil {
        return err
    }

    userKey := fmt.Sprintf("%s%s", UserSessionsPrefix, data["user_id"])
    pipe := sm.redis.Pipeline()
    pipe.Del(ctx, sessionKey)
    pipe.SRem(ctx, userKey, sessionID)
    _, err = pipe.Exec(ctx)

    return err
}

func (sm *SessionManager) RevokeAllUserSessions(ctx context.Context, userID string, exceptSessionID string) error {
    userKey := fmt.Sprintf("%s%s", UserSessionsPrefix, userID)
    sessions, err := sm.redis.SMembers(ctx, userKey).Result()
    if err != nil {
        return err
    }

    pipe := sm.redis.Pipeline()
    for _, sid := range sessions {
        if sid != exceptSessionID {
            pipe.Del(ctx, fmt.Sprintf("%s%s", SessionPrefix, sid))
            pipe.SRem(ctx, userKey, sid)
        }
    }
    _, err = pipe.Exec(ctx)
    return err
}
```

### 2.4 Password Security
```go
// internal/infrastructure/security/password.go
package security

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12  // OWASP recommended minimum

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    return string(bytes), err
}

func VerifyPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

---

## 3. API Endpoints (Aligned with Flutter)

### 3.1 Router Configuration
```go
// internal/infrastructure/http/router.go
package http

import (
    "github.com/gin-gonic/gin"
    "pos-backend/internal/infrastructure/http/handlers"
    "pos-backend/internal/infrastructure/http/middleware"
)

func SetupRouter(
    authHandler *handlers.AuthHandler,
    productHandler *handlers.ProductHandler,
    saleHandler *handlers.SaleHandler,
    syncHandler *handlers.SyncHandler,
    authMiddleware *middleware.AuthMiddleware,
    tenantMiddleware *middleware.TenantMiddleware,
    rateLimitMiddleware *middleware.RateLimitMiddleware,
    securityMiddleware *middleware.SecurityMiddleware,
) *gin.Engine {
    r := gin.New()

    // Global middleware
    r.Use(gin.Recovery())
    r.Use(middleware.RequestID())
    r.Use(middleware.Logger())
    r.Use(middleware.CORS())
    r.Use(securityMiddleware.SecurityHeaders())

    // Health check (no auth)
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok", "version": "1.0.0"})
    })

    // API v1
    v1 := r.Group("/v1")

    // Auth routes (public, rate limited)
    auth := v1.Group("/auth")
    auth.Use(rateLimitMiddleware.LoginLimit())
    {
        auth.POST("/login", authHandler.Login)
        auth.POST("/refresh", authHandler.Refresh)
        auth.POST("/logout", authMiddleware.RequireAuth(), authHandler.Logout)
        auth.POST("/logout-all", authMiddleware.RequireAuth(), authHandler.LogoutAll)
    }

    // Protected routes
    protected := v1.Group("")
    protected.Use(authMiddleware.RequireAuth())
    protected.Use(tenantMiddleware.ExtractTenant())
    protected.Use(rateLimitMiddleware.APILimit())
    protected.Use(securityMiddleware.ValidateFingerprint())
    {
        // Products
        protected.GET("/products", productHandler.List)
        protected.GET("/products/:sku", productHandler.GetBySKU)

        // Sales
        protected.POST("/sales", saleHandler.Create)
        protected.GET("/sales", saleHandler.List)
        protected.GET("/sales/:id", saleHandler.GetByID)

        // Sync
        protected.POST("/sync/push", syncHandler.Push)
        protected.GET("/sync/pull", syncHandler.Pull)
    }

    return r
}
```

### 3.2 Auth Handler

#### POST `/v1/auth/login`
```go
// internal/infrastructure/http/handlers/auth_handler.go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "pos-backend/internal/dto"
    "pos-backend/internal/usecases"
    "pos-backend/pkg/response"
)

type AuthHandler struct {
    authUC *usecases.AuthUseCase
}

func (h *AuthHandler) Login(c *gin.Context) {
    var req dto.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.ValidationError(c, err)
        return
    }

    // Extract client metadata
    clientIP := c.ClientIP()
    userAgent := c.GetHeader("User-Agent")
    fingerprint := c.GetHeader("X-Session-Fingerprint")

    result, err := h.authUC.Login(c.Request.Context(), usecases.LoginInput{
        Email:       req.Email,
        Password:    req.Password,
        TenantID:    req.TenantID,
        DeviceID:    req.DeviceID,
        DeviceName:  req.DeviceName,
        FCMToken:    req.FCMToken,
        IPAddress:   clientIP,
        UserAgent:   userAgent,
        Fingerprint: fingerprint,
    })

    if err != nil {
        switch err {
        case usecases.ErrInvalidCredentials:
            response.Error(c, http.StatusUnauthorized, "invalid_credentials", "Email or password incorrect")
        case usecases.ErrTenantInactive:
            response.Error(c, http.StatusForbidden, "tenant_inactive", "Your subscription has expired")
        case usecases.ErrDeviceBlocked:
            response.Error(c, http.StatusForbidden, "device_blocked", "This device is not authorized")
        default:
            response.Error(c, http.StatusInternalServerError, "server_error", "An unexpected error occurred")
        }
        return
    }

    response.Success(c, http.StatusOK, dto.LoginResponse{
        AccessToken:  result.AccessToken,
        RefreshToken: result.RefreshToken,
        ExpiresIn:    result.ExpiresIn,
        User:         dto.MapUserToDTO(result.User),
        Tenant:       dto.MapTenantToDTO(result.Tenant),
        Permissions:  result.Permissions,
    })
}
```

#### DTO Definitions
```go
// internal/dto/auth_dto.go
package dto

import "github.com/go-playground/validator/v10"

type LoginRequest struct {
    Email      string `json:"email" binding:"required,email"`
    Password   string `json:"password" binding:"required,min=8"`
    TenantID   string `json:"tenant_id" binding:"required,alphanum,min=3,max=50"`
    DeviceID   string `json:"device_id" binding:"required,uuid"`
    DeviceName string `json:"device_name" binding:"required,max=100"`
    FCMToken   string `json:"fcm_token,omitempty"`
}

type LoginResponse struct {
    AccessToken  string       `json:"access_token"`
    RefreshToken string       `json:"refresh_token"`
    ExpiresIn    int          `json:"expires_in"`
    User         UserDTO      `json:"user"`
    Tenant       TenantDTO    `json:"tenant"`
    Permissions  []string     `json:"permissions"`
}

type RefreshRequest struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
    DeviceID     string `json:"device_id" binding:"required,uuid"`
}

type RefreshResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
}
```

### 3.3 Product Handler

#### GET `/v1/products`
```go
func (h *ProductHandler) List(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    userID := c.GetString("user_id")

    var query dto.ProductListQuery
    if err := c.ShouldBindQuery(&query); err != nil {
        response.ValidationError(c, err)
        return
    }

    // Apply tenant filter + RLS
    products, meta, err := h.productUC.List(c.Request.Context(), usecases.ProductListInput{
        TenantID:     tenantID,
        UserID:       userID,
        Search:       query.Search,
        CategoryID:   query.CategoryID,
        UpdatedAfter: query.UpdatedAfter,
        Page:         query.Page,
        Limit:        query.Limit,
    })

    if err != nil {
        response.Error(c, http.StatusInternalServerError, "server_error", err.Error())
        return
    }

    response.Success(c, http.StatusOK, dto.ProductListResponse{
        Items: dto.MapProductsToDTO(products),
        Meta:  meta,
    })
}

// Query DTO
type ProductListQuery struct {
    Search       string    `form:"search"`
    CategoryID   string    `form:"category_id"`
    UpdatedAfter time.Time `form:"updated_after" time_format:"2006-01-02T15:04:05Z07:00"`
    Page         int       `form:"page,default=1" binding:"min=1"`
    Limit        int       `form:"limit,default=50" binding:"min=1,max=200"`
}
```

#### GET `/v1/products/:sku` (Barcode Lookup)
```go
func (h *ProductHandler) GetBySKU(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    sku := c.Param("sku")

    // Validate SKU format
    if !validators.IsValidSKU(sku) {
        response.Error(c, http.StatusBadRequest, "invalid_sku", "Invalid SKU format")
        return
    }

    product, err := h.productUC.GetBySKU(c.Request.Context(), tenantID, sku)
    if err != nil {
        if errors.Is(err, usecases.ErrProductNotFound) {
            response.Error(c, http.StatusNotFound, "product_not_found", "Product not found")
            return
        }
        response.Error(c, http.StatusInternalServerError, "server_error", err.Error())
        return
    }

    response.Success(c, http.StatusOK, dto.MapProductToDTO(product))
}
```

### 3.4 Sale Handler

#### POST `/v1/sales`
```go
func (h *SaleHandler) Create(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    userID := c.GetString("user_id")
    deviceID := c.GetString("device_id")

    var req dto.SaleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.ValidationError(c, err)
        return
    }

    // Idempotency check
    existing, err := h.saleUC.GetByClientID(c.Request.Context(), tenantID, req.ID)
    if err == nil && existing != nil {
        response.Error(c, http.StatusConflict, "sale_exists", "Sale already processed")
        return
    }

    sale, err := h.saleUC.Create(c.Request.Context(), usecases.CreateSaleInput{
        ClientID:       req.ID,
        TenantID:       tenantID,
        UserID:         userID,
        DeviceID:       deviceID,
        Items:          dto.MapSaleItemsFromDTO(req.Items),
        Subtotal:       req.Subtotal,
        TaxAmount:      req.TaxAmount,
        DiscountAmount: req.DiscountAmount,
        Total:          req.Total,
        Currency:       req.Currency,
        PaymentMethod:  req.PaymentMethod,
        CustomerID:     req.CustomerID,
        Notes:          req.Notes,
        ClientTimestamp: req.CreatedAt,
    })

    if err != nil {
        switch {
        case errors.Is(err, usecases.ErrInvalidSaleTotal):
            response.Error(c, http.StatusUnprocessableEntity, "invalid_total", "Sale total does not match items")
        case errors.Is(err, usecases.ErrInsufficientStock):
            response.Error(c, http.StatusUnprocessableEntity, "insufficient_stock", "One or more items out of stock")
        default:
            response.Error(c, http.StatusInternalServerError, "server_error", err.Error())
        }
        return
    }

    response.Success(c, http.StatusCreated, dto.SaleResponse{
        ID:             sale.ID,
        ReceiptNumber:  sale.ReceiptNumber,
        Status:         sale.Status,
        ServerTimestamp: sale.CreatedAt,
    })
}
```

### 3.5 Sync Handler

#### POST `/v1/sync/push`
```go
func (h *SyncHandler) Push(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    deviceID := c.GetString("device_id")

    var req dto.SyncPushRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.ValidationError(c, err)
        return
    }

    result, err := h.syncUC.Push(c.Request.Context(), usecases.SyncPushInput{
        TenantID:      tenantID,
        DeviceID:      deviceID,
        LastSyncToken: req.LastSyncToken,
        Sales:         dto.MapSyncSalesFromDTO(req.Sales),
    })

    if err != nil {
        response.Error(c, http.StatusInternalServerError, "sync_failed", err.Error())
        return
    }

    response.Success(c, http.StatusOK, dto.SyncPushResponse{
        AcceptedIDs:     result.AcceptedIDs,
        Conflicts:       dto.MapConflictsToDTO(result.Conflicts),
        NewSyncToken:    result.NewSyncToken,
        ServerTimestamp: result.ServerTimestamp,
    })
}
```

#### GET `/v1/sync/pull`
```go
func (h *SyncHandler) Pull(c *gin.Context) {
    tenantID := c.GetString("tenant_id")

    var query dto.SyncPullQuery
    if err := c.ShouldBindQuery(&query); err != nil {
        response.ValidationError(c, err)
        return
    }

    result, err := h.syncUC.Pull(c.Request.Context(), usecases.SyncPullInput{
        TenantID:      tenantID,
        LastSyncToken: query.LastSyncToken,
        EntityTypes:   query.EntityTypes,
    })

    if err != nil {
        response.Error(c, http.StatusInternalServerError, "sync_failed", err.Error())
        return
    }

    response.Success(c, http.StatusOK, dto.SyncPullResponse{
        Products:        dto.MapProductsToDTO(result.Products),
        Prices:          dto.MapPricesToDTO(result.Prices),
        Customers:       dto.MapCustomersToDTO(result.Customers),
        NewSyncToken:    result.NewSyncToken,
        HasMore:         result.HasMore,
        ServerTimestamp: result.ServerTimestamp,
    })
}
```

---

## 4. Database Schema (PostgreSQL)

### 4.1 Core Tables
```sql
-- migrations/001_init.up.sql

-- Tenants (Multi-tenancy root)
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(50) UNIQUE NOT NULL,           -- "acme-corp"
    name VARCHAR(100) NOT NULL,
    plan VARCHAR(20) NOT NULL DEFAULT 'free',   -- free, basic, pro, enterprise
    is_active BOOLEAN NOT NULL DEFAULT true,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(20) NOT NULL DEFAULT 'cashier',  -- admin, manager, cashier
    permissions TEXT[] DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

-- Devices (Registered POS terminals)
CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    device_id VARCHAR(255) NOT NULL,              -- Client-generated UUID
    name VARCHAR(100) NOT NULL,
    fingerprint VARCHAR(64),                      -- SHA-256 of device info
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, device_id)
);

-- Products
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sku VARCHAR(100) NOT NULL,                   -- Barcode/QR
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category_id UUID,
    base_price DECIMAL(15,4) NOT NULL,
    sale_price DECIMAL(15,4),
    cost_price DECIMAL(15,4),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    stock_quantity DECIMAL(15,4) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, sku)
);

-- Categories
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    parent_id UUID REFERENCES categories(id),
    sort_order INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sales
CREATE TABLE sales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id VARCHAR(36) UNIQUE,                -- Idempotency key from Flutter
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    device_id UUID REFERENCES devices(id),
    receipt_number VARCHAR(50) NOT NULL,
    subtotal DECIMAL(15,4) NOT NULL,
    tax_amount DECIMAL(15,4) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(15,4) NOT NULL DEFAULT 0,
    total DECIMAL(15,4) NOT NULL,
    currency CHAR(3) NOT NULL,
    payment_method VARCHAR(20) NOT NULL,          -- cash, card, mada
    customer_id UUID,
    notes TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'confirmed', -- confirmed, refunded, cancelled
    client_timestamp TIMESTAMPTZ,                -- When sale happened on device
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sale Items
CREATE TABLE sale_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sale_id UUID NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    product_name VARCHAR(255) NOT NULL,           -- Denormalized
    sku VARCHAR(100) NOT NULL,                    -- Denormalized
    quantity DECIMAL(15,4) NOT NULL,
    unit_price DECIMAL(15,4) NOT NULL,
    total_price DECIMAL(15,4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sync Tokens (for delta sync)
CREATE TABLE sync_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    entity_types TEXT[] NOT NULL,
    last_sequence BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '7 days',
    UNIQUE(tenant_id, device_id)
);

-- Audit Log
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID,
    action VARCHAR(50) NOT NULL,                  -- login, sale_created, product_updated
    entity_type VARCHAR(50),                       -- sale, product, user
    entity_id UUID,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_products_tenant_sku ON products(tenant_id, sku);
CREATE INDEX idx_products_tenant_active ON products(tenant_id, is_active);
CREATE INDEX idx_products_tenant_updated ON products(tenant_id, updated_at);
CREATE INDEX idx_sales_tenant_created ON sales(tenant_id, created_at);
CREATE INDEX idx_sales_client_id ON sales(client_id);
CREATE INDEX idx_sale_items_sale ON sale_items(sale_id);
CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id, created_at);
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);

-- Row Level Security (RLS) Policies
ALTER TABLE products ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales ENABLE ROW LEVEL SECURITY;
ALTER TABLE sale_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE devices ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_products ON products
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

CREATE POLICY tenant_isolation_sales ON sales
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

CREATE POLICY tenant_isolation_users ON users
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

-- Updated at trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_sales_updated_at BEFORE UPDATE ON sales
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### 4.2 Multi-Tenancy Strategy
- **Row-Level Security (RLS)**: PostgreSQL native RLS with `app.current_tenant` setting
- **Schema-per-tenant**: Alternative for enterprise tier (isolated schemas)
- **Tenant Context**: Set on every DB connection via `SET app.current_tenant = 'uuid'`

```go
// internal/infrastructure/database/postgres.go
func (db *PostgresDB) WithTenant(ctx context.Context, tenantID string) (context.Context, error) {
    conn, err := db.pool.Acquire(ctx)
    if err != nil {
        return ctx, err
    }
    defer conn.Release()

    _, err = conn.Exec(ctx, "SET app.current_tenant = $1", tenantID)
    if err != nil {
        return ctx, err
    }

    return context.WithValue(ctx, "db_conn", conn), nil
}
```

---

## 5. Middleware & Security

### 5.1 Authentication Middleware
```go
// internal/infrastructure/http/middleware/auth.go
package middleware

import (
    "net/http"
    "strings"
    "github.com/gin-gonic/gin"
    "pos-backend/internal/infrastructure/security"
    "pos-backend/pkg/response"
)

type AuthMiddleware struct {
    jwtValidator *security.JWTValidator
    sessionMgr   *security.SessionManager
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            response.Error(c, http.StatusUnauthorized, "missing_token", "Authorization header required")
            c.Abort()
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
            response.Error(c, http.StatusUnauthorized, "invalid_token_format", "Bearer token required")
            c.Abort()
            return
        }

        tokenString := parts[1]
        claims, err := m.jwtValidator.ValidateAccessToken(tokenString)
        if err != nil {
            response.Error(c, http.StatusUnauthorized, "invalid_token", err.Error())
            c.Abort()
            return
        }

        // Validate session in Redis
        deviceID := c.GetHeader("X-Device-ID")
        fingerprint := c.GetHeader("X-Session-Fingerprint")
        if !m.sessionMgr.ValidateSession(c.Request.Context(), claims.SessionID, deviceID, fingerprint) {
            response.Error(c, http.StatusUnauthorized, "session_invalid", "Session expired or device mismatch")
            c.Abort()
            return
        }

        // Set context values
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("tenant_id", claims.TenantID)
        c.Set("device_id", claims.DeviceID)
        c.Set("session_id", claims.SessionID)
        c.Set("role", claims.Role)
        c.Set("permissions", claims.Permissions)

        c.Next()
    }
}
```

### 5.2 Rate Limiting
```go
// internal/infrastructure/http/middleware/rate_limit.go
package middleware

import (
    "net/http"
    "time"
    "github.com/gin-gonic/gin"
    "github.com/redis/go-redis/v9"
    "pos-backend/pkg/response"
)

type RateLimitMiddleware struct {
    redis *redis.Client
}

func (m *RateLimitMiddleware) LoginLimit() gin.HandlerFunc {
    return m.createLimiter("login", 5, time.Minute*5)  // 5 attempts per 5 min
}

func (m *RateLimitMiddleware) APILimit() gin.HandlerFunc {
    return m.createLimiter("api", 1000, time.Minute)   // 1000 req/min per tenant
}

func (m *RateLimitMiddleware) createLimiter(prefix string, maxRequests int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := prefix + ":" + c.ClientIP()
        if prefix == "api" {
            tenantID := c.GetHeader("X-Tenant-ID")
            deviceID := c.GetHeader("X-Device-ID")
            key = prefix + ":" + tenantID + ":" + deviceID
        }

        ctx := c.Request.Context()
        pipe := m.redis.Pipeline()
        incr := pipe.Incr(ctx, key)
        pipe.Expire(ctx, key, window)
        _, err := pipe.Exec(ctx)

        if err != nil {
            c.Next()
            return
        }

        if incr.Val() > int64(maxRequests) {
            response.Error(c, http.StatusTooManyRequests, "rate_limited", 
                "Too many requests. Please try again later.")
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### 5.3 Security Headers
```go
// internal/infrastructure/http/middleware/security.go
package middleware

import "github.com/gin-gonic/gin"

type SecurityMiddleware struct{}

func (m *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        c.Next()
    }
}

func (m *SecurityMiddleware) ValidateFingerprint() gin.HandlerFunc {
    return func(c *gin.Context) {
        fp := c.GetHeader("X-Session-Fingerprint")
        if fp == "" || len(fp) != 64 {
            response.Error(c, http.StatusForbidden, "invalid_fingerprint", "Device fingerprint required")
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 5.4 CORS Configuration
```go
func CORS() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")  // Restrict in production
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", 
            "Origin, Content-Type, Accept, Authorization, X-Tenant-ID, X-Device-ID, X-Request-ID, X-Session-Fingerprint, X-Timestamp")
        c.Header("Access-Control-Expose-Headers", "X-Request-ID")
        c.Header("Access-Control-Max-Age", "86400")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        c.Next()
    }
}
```

---

## 6. Response Standardization

```go
// pkg/response/json_response.go
package response

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/go-playground/validator/v10"
)

type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *APIError   `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type Meta struct {
    RequestID string `json:"request_id"`
    Timestamp string `json:"timestamp"`
    Page      int    `json:"page,omitempty"`
    Limit     int    `json:"limit,omitempty"`
    Total     int64  `json:"total,omitempty"`
}

func Success(c *gin.Context, status int, data interface{}) {
    c.JSON(status, APIResponse{
        Success: true,
        Data:    data,
        Meta: &Meta{
            RequestID: c.GetString("request_id"),
            Timestamp: time.Now().UTC().Format(time.RFC3339),
        },
    })
}

func Error(c *gin.Context, status int, code, message string) {
    c.JSON(status, APIResponse{
        Success: false,
        Error: &APIError{
            Code:    code,
            Message: message,
        },
        Meta: &Meta{
            RequestID: c.GetString("request_id"),
            Timestamp: time.Now().UTC().Format(time.RFC3339),
        },
    })
}

func ValidationError(c *gin.Context, err error) {
    if validationErrors, ok := err.(validator.ValidationErrors); ok {
        errors := make(map[string]string)
        for _, e := range validationErrors {
            errors[e.Field()] = e.Tag()
        }
        c.JSON(http.StatusBadRequest, APIResponse{
            Success: false,
            Error: &APIError{
                Code:    "validation_failed",
                Message: "Request validation failed",
            },
            Meta: &Meta{
                RequestID: c.GetString("request_id"),
                Timestamp: time.Now().UTC().Format(time.RFC3339),
            },
        })
        return
    }
    Error(c, http.StatusBadRequest, "bad_request", err.Error())
}
```

---

## 7. Deployment (Ubuntu + systemd)

### 7.1 systemd Service
```ini
; deployments/systemd/pos-api.service
[Unit]
Description=POS SaaS API
After=network.target postgresql.service redis-server.service

[Service]
Type=simple
User=pos-api
Group=pos-api
WorkingDirectory=/opt/pos-api
ExecStart=/opt/pos-api/pos-api
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=pos-api

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/pos-api/logs
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

# Environment
Environment="GO_ENV=production"
Environment="PORT=8080"
EnvironmentFile=/opt/pos-api/.env

[Install]
WantedBy=multi-user.target
```

### 7.2 Caddy Reverse Proxy
```
# deployments/caddy/Caddyfile
api.yourpos.com {
    reverse_proxy localhost:8080

    # Security headers
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        Referrer-Policy "strict-origin-when-cross-origin"
    }

    # Rate limiting at edge
    rate_limit {
        zone static_example {
            key {remote_host}
            events 100
            window 1m
        }
    }

    # Logging
    log {
        output file /var/log/caddy/access.log
        format json
    }

    # TLS (auto via Let's Encrypt)
    tls admin@yourpos.com
}
```

### 7.3 Docker Compose (Development)
```yaml
# deployments/docker/docker-compose.yml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://pos:pos@postgres:5432/pos_db?sslmode=disable
      - REDIS_URL=redis:6379
      - JWT_ACCESS_SECRET=${JWT_ACCESS_SECRET}
      - JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: pos
      POSTGRES_PASSWORD: pos
      POSTGRES_DB: pos_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"

volumes:
  postgres_data:
  redis_data:
```

### 7.4 Makefile
```makefile
# Makefile
.PHONY: build run test migrate migrate-down docker-build deploy

APP_NAME=pos-api
BUILD_DIR=./build
MAIN_FILE=./cmd/api/main.go

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)

run:
	go run $(MAIN_FILE)

test:
	go test -v -race -coverprofile=coverage.out ./...

migrate:
	migrate -path ./internal/infrastructure/database/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./internal/infrastructure/database/migrations -database "$(DATABASE_URL)" down 1

docker-build:
	docker build -t $(APP_NAME):latest -f deployments/docker/Dockerfile .

deploy: build
	sudo systemctl stop $(APP_NAME)
	cp $(BUILD_DIR)/$(APP_NAME) /opt/pos-api/
	sudo systemctl start $(APP_NAME)
	sudo systemctl status $(APP_NAME)
```

---

## 8. Monitoring & Observability

### 8.1 Structured Logging (slog)
```go
// internal/config/logger.go
package config

import (
    "log/slog"
    "os"
)

func SetupLogger(env string) *slog.Logger {
    var handler slog.Handler
    if env == "production" {
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelInfo,
        })
    } else {
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelDebug,
        })
    }
    return slog.New(handler)
}
```

### 8.2 Prometheus Metrics
```go
// internal/infrastructure/http/middleware/metrics.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"
)

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )

    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

func Metrics() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()

        duration := time.Since(start).Seconds()
        status := strconv.Itoa(c.Writer.Status())

        httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
        httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
    }
}
```

---

## 9. Dependencies (go.mod)

```go
module pos-backend

go 1.23

require (
    github.com/gin-gonic/gin v1.10.0
    github.com/golang-jwt/jwt/v5 v5.2.1
    github.com/go-playground/validator/v10 v10.22.0
    github.com/jackc/pgx/v5 v5.6.0
    github.com/jmoiron/sqlx v1.4.0
    github.com/redis/go-redis/v9 v9.6.1
    github.com/spf13/viper v1.19.0
    github.com/google/uuid v1.6.0
    github.com/prometheus/client_golang v1.20.0
    github.com/pressly/goose/v3 v3.21.1
    golang.org/x/crypto v0.26.0
)
```

---

## 10. Environment Configuration (.env.example)

```bash
# Server
PORT=8080
GO_ENV=development

# Database
DATABASE_URL=postgres://pos:pos@localhost:5432/pos_db?sslmode=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# Redis
REDIS_URL=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_ACCESS_SECRET=your-256-bit-secret-key-here-change-in-production
JWT_REFRESH_SECRET=your-different-256-bit-refresh-secret-here

# Security
BCRYPT_COST=12
MAX_SESSIONS_PER_USER=3

# CORS
CORS_ALLOWED_ORIGINS=https://yourpos.com,https://admin.yourpos.com

# Logging
LOG_LEVEL=info

# Sync
SYNC_TOKEN_TTL=168h
SYNC_BATCH_SIZE=500
```

---

## 11. Testing Strategy

| Type | Tool | Target |
|------|------|--------|
| Unit | `testing` + `testify` | Use cases, repositories (mocked) |
| Integration | `testing` + testcontainers-go | DB queries, Redis operations |
| API | `httptest` + `gin` | Handler layer |
| E2E | `cypress` or `k6` | Load testing, critical flows |
| Benchmark | `testing.B` | Barcode lookup (<10ms), sale creation |

---

*Document Version: 1.0 | Last Updated: 2026-09-04*
*Aligned with: `flutter_pos_spec.md` v1.0*
