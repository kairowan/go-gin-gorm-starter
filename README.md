<details>
  <summary><strong>目录结构</strong></summary>

<pre><code>go-gin-gorm-starter/
├─ cmd/                                           — 应用入口层（可有多个可执行程序）
│  ├─ api/
│  │  └─ main.go                                  — 用户端 API 入口：加载配置→建日志/DB→装配路由→启动/关闭
│  └─ admin/
│     └─ main.go                                  — 后台端 API 入口：同上，路由为 /admin/v1
├─ configs/                                       — 配置文件目录
│  └─ config.local.yaml                           — 本地实际配置
├─ internal/                                      — 内部实现（不导出）
│  ├─ core/                                       — 核心基础设施（横切关注：配置/DB/日志/HTTP等）
│  │  ├─ auth/jwt.go                              — JWT 工具：签发/解析自定义 Claims
│  │  ├─ cache/                                   — 缓存层（Redis + singleflight）
│  │  │  ├─ redis.go                              — Redis 客户端封装 + GetOrLoad（读穿）
│  │  │  └─ json.go                               — GetOrLoadJSON[T] JSON 序列化助手
│  │  ├─ config/config.go                         — 配置加载：Viper 读取 YAML + 环境变量覆盖
│  │  ├─ database/gorm.go                         — GORM 初始化：驱动选择、连接池、日志等级、Navicat URL→DSN 适配
│  │  ├─ logger/logger.go                         — Zap 日志构建器（控制台/文件切割、ReplaceGlobals、StdLog、Ctx）
│  │  └─ server/router.go                         — Gin 基础 Router 构造 + http.Server 构建（超时/地址拼装）
│  ├─ domain/user.go                              — 领域模型 User + 仓储接口定义（Repository Port）
│  ├─ repo/user_repo.go                           — User 仓储 GORM 实现（Repository Adapter）
│  ├─ service/user_service.go                     — 业务服务：注册、登录、查询、封禁（应用用例）
│  └─ transport/http/                             — 传输层（HTTP）
│     ├─ handler/
│     │  ├─ admin_handler.go                      — 后台端 Handler：用户列表/封禁
│     │  └─ user_handler.go                       — 用户端 Handler：注册/登录/个人资料
│     ├─ middleware/
│     │  ├─ accesslog.go                          — 访问日志（脱敏摘要）
│     │  ├─ ratelimit.go                          — 全局/按 IP 限速中间件
│     │  ├─ concurrency.go                        — 并发闸门（限制同时处理请求数）
│     │  ├─ maxbody.go                            — 限制请求体大小，保护上传/大包
│     │  ├─ metrics.go                            — Prometheus 指标（QPS/延迟/状态码）
│     │  ├─ auth_jwt.go                           — JWT 鉴权中间件（可校验角色：user/admin）
│     │  ├─ requestid.go                          — 请求 ID 中间件：注入/回传 X-Request-ID
│     │  ├─ recovery.go                           — Panic 恢复：500 兜底，防止进程崩溃
│     │  └─ timeout.go                            — 请求超时：超时自动返回 504
│     ├─ response/response.go                     — 统一响应结构：Resp{code,msg,data}
│     └─ router/
│        ├─ api.go                                — 用户端路由装配：/api/v1（健康检查/注册/登录/鉴权后 /me）
│        ├─ admin.go                              — 后台端路由装配：/admin/v1（鉴权要求 admin 角色）
│        └─ registry.go                           — 统一路由注册器（APIModule/AdminModule + 可选 Priority）
├─ migrations/
│  └─ 20250905_init.sql                           — 示例数据库迁移脚本（可放 DDL/补数据）
├─ pkg/utils/                                     — 轻量工具库（与业务无关）
│  ├─ id.go                                       — UUID 生成（用户 ID 等）
│  └─ password.go                                 — 密码哈希/校验（bcrypt）
├─ .env.example                                   — 环境变量示例（可配 CONFIG_PATH 等）
├─ go.mod                                         — Go 模块定义 + 依赖版本
├─ Makefile                                       — 常用命令：tidy/run-api/run-admin/test（跨平台稳健版）
└─ README.md                                      — 项目树形结构
</code></pre>

</details>
