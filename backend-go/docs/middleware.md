# Middleware

`gateway-api` middleware stack:

1. RequestID / TraceID
   - implemented in `common/middleware.RequestID`
   - returns `X-Trace-ID`
   - stores trace ID in context

2. AccessLog
   - implemented in `common/middleware.AccessLog`
   - logs method, path, latency, ip, user agent and trace ID

3. Recover
   - implemented in `common/middleware.Recover`
   - converts panic to unified error response

4. CORS
   - implemented in `common/middleware.CORS`
   - supports browser frontend calls

5. JWT Auth
   - implemented in `common/middleware.JWTAuth`
   - validates Bearer token with `common/jwt`

6. RBAC
   - first-round placeholder in `common/middleware.RBAC`
   - will be wired to `permissions` and `role_permissions`

7. RateLimit
   - first-round placeholder in `common/middleware.RateLimit`
   - intended for login, Agent, RAG and market APIs

8. Idempotency
   - first-round placeholder in `common/middleware.Idempotency`
   - paper order APIs will use `Idempotency-Key`

9. AuditLog
   - database table exists as `audit_logs`
   - writing audit records will be added with user-rpc and paper-rpc flows

All failed responses must follow:

```json
{"code":10001,"message":"error message","data":null,"trace_id":"..."}
```

