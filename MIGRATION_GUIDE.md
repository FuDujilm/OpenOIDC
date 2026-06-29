# OpenOIDC 迁移方案

## 当前架构分析

### 现有系统
- **mp-oauth2**: Next.js + NextAuth.js 自建认证系统
  - 支持邮箱注册/登录
  - 支持 Google、GitHub、微信、LinuxDO OAuth
  - PostgreSQL 数据库存储用户数据
  - 提供标准 OAuth 2.0 端点

- **MeowzExam**: Next.js 业余无线电考试系统
  - 通过 NextAuth.js 接入 mp-oauth2
  - OAuth 客户端 ID: `68e2b22b41e23a1c5800d12bfa69a202`
  
- **beacon-api**: Rust 后端服务
  - 环境变量配置: `OAUTH_BASE_URL=https://oauth.mzyd.work`
  
- **beacon-toolkit**: （待分析）

## 迁移目标

将 **OpenOIDC** 替代 mp-oauth2，作为统一身份认证平台：
- ✅ 更完善的置信等级模型（Trust Level）
- ✅ 风控能力（滥用上报、共享风控库）
- ✅ 更多第三方绑定（Discord、Telegram、QQ、微信、Apple 等）
- ✅ 准入策略（按应用设置最低置信等级 + 必须绑定条件）
- ✅ WebAuthn/Passkey 支持
- ✅ 后台管理控制台
- ✅ 审计日志

---

## 迁移方案

### 阶段一：数据迁移（用户数据同步）

#### 1.1 从 mp-oauth2 导出用户数据

创建数据导出脚本：

```sql
-- mp-oauth2 PostgreSQL 数据导出
-- 导出用户表
COPY (
  SELECT 
    id,
    name,
    email,
    "emailVerified",
    image,
    "createdAt",
    "updatedAt"
  FROM "User"
) TO '/tmp/mp_oauth2_users.csv' WITH CSV HEADER;

-- 导出本地账户（密码）
COPY (
  SELECT 
    "userId",
    email,
    password,
    "emailVerified",
    "createdAt"
  FROM "LocalAccount"
) TO '/tmp/mp_oauth2_local_accounts.csv' WITH CSV HEADER;

-- 导出 OAuth 账号绑定
COPY (
  SELECT 
    "userId",
    provider,
    "providerAccountId",
    "createdAt"
  FROM "Account"
) TO '/tmp/mp_oauth2_accounts.csv' WITH CSV HEADER;
```

#### 1.2 导入到 OpenOIDC

OpenOIDC 数据库结构：
- `users` 表：用户基础信息
- `bindings` 表：第三方账号绑定（GitHub、Google 等）
- `clients` 表：接入的业务应用（OAuth 客户端）

创建迁移脚本 `scripts/migrate_from_mp_oauth2.go`:

```go
package main

import (
    "context"
    "database/sql"
    "encoding/csv"
    "fmt"
    "log"
    "os"
    "time"
    
    _ "github.com/jackc/pgx/v5/stdlib"
)

type User struct {
    ID            string
    Name          string
    Email         string
    EmailVerified bool
    Image         string
    CreatedAt     time.Time
}

type LocalAccount struct {
    UserID        string
    Email         string
    Password      string  // bcrypt 哈希，可以直接迁移
    EmailVerified bool
}

type OAuthAccount struct {
    UserID            string
    Provider          string
    ProviderAccountID string
    CreatedAt         time.Time
}

func main() {
    // 连接 OpenOIDC 数据库
    db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    ctx := context.Background()
    
    // 1. 导入用户
    users, err := loadUsers("/tmp/mp_oauth2_users.csv")
    if err != nil {
        log.Fatal(err)
    }
    
    for _, u := range users {
        // 插入到 OpenOIDC users 表
        _, err := db.ExecContext(ctx, `
            INSERT INTO users (id, email, display_name, email_verified, avatar_url, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            ON CONFLICT (email) DO NOTHING
        `, u.ID, u.Email, u.Name, u.EmailVerified, u.Image, u.CreatedAt, time.Now())
        
        if err != nil {
            log.Printf("导入用户 %s 失败: %v", u.Email, err)
        }
    }
    
    // 2. 导入本地账户（密码）
    localAccounts, err := loadLocalAccounts("/tmp/mp_oauth2_local_accounts.csv")
    if err != nil {
        log.Fatal(err)
    }
    
    for _, acc := range localAccounts {
        // OpenOIDC 中密码存储在 users 表的 password_hash 字段
        _, err := db.ExecContext(ctx, `
            UPDATE users 
            SET password_hash = $1
            WHERE id = $2
        `, acc.Password, acc.UserID)
        
        if err != nil {
            log.Printf("导入密码失败: %v", err)
        }
    }
    
    // 3. 导入 OAuth 绑定
    oauthAccounts, err := loadOAuthAccounts("/tmp/mp_oauth2_accounts.csv")
    if err != nil {
        log.Fatal(err)
    }
    
    // mp-oauth2 provider 名称映射到 OpenOIDC
    providerMap := map[string]string{
        "google":   "google",
        "github":   "github",
        "wechat":   "wechat",
        "linuxdo":  "linuxdo",
    }
    
    for _, acc := range oauthAccounts {
        provider, ok := providerMap[acc.Provider]
        if !ok {
            log.Printf("未知的 provider: %s", acc.Provider)
            continue
        }
        
        // 插入到 OpenOIDC bindings 表
        _, err := db.ExecContext(ctx, `
            INSERT INTO bindings (user_id, provider, provider_user_id, provider_username, bound_at)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (user_id, provider) DO NOTHING
        `, acc.UserID, provider, acc.ProviderAccountID, "", acc.CreatedAt)
        
        if err != nil {
            log.Printf("导入绑定失败: %v", err)
        }
    }
    
    fmt.Println("✅ 数据迁移完成")
}

func loadUsers(path string) ([]User, error) {
    // 读取 CSV 并解析
    // ...省略 CSV 解析代码
    return nil, nil
}

func loadLocalAccounts(path string) ([]LocalAccount, error) {
    // 读取 CSV 并解析
    return nil, nil
}

func loadOAuthAccounts(path string) ([]OAuthAccount, error) {
    // 读取 CSV 并解析
    return nil, nil
}
```

运行迁移：
```bash
cd /data/D/Project/OpenOIDC
export DATABASE_URL="postgresql://user:pass@host:5432/openoidc"
go run scripts/migrate_from_mp_oauth2.go
```

---

### 阶段二：业务系统改造

#### 2.1 MeowzExam 改造

**当前配置**（`MeowzExam/.env`）:
```env
OAUTH_BASE_URL="http://192.168.31.187:3005"  # mp-oauth2
OAUTH_CLIENT_ID="68e2b22b41e23a1c5800d12bfa69a202"
OAUTH_CLIENT_SECRET="032b23848eb021b7b39b5549dffea6e6e91b269a8bd5395e321169864a97d65b"
```

**改造步骤**:

1. **在 OpenOIDC 后台创建应用**
   - 访问 `http://localhost:8080/admin` (管理员登录)
   - 创建新应用："MeowzExam 业余无线电考试系统"
   - 设置 Redirect URI: `http://192.168.31.187:3001/api/auth/callback/custom`
   - 获取新的 Client ID 和 Client Secret

2. **修改 MeowzExam 配置**

更新 `MeowzExam/.env`:
```env
# 改为 OpenOIDC 地址
OAUTH_BASE_URL="http://localhost:8080"
NEXT_PUBLIC_OAUTH_BASE_URL="http://localhost:8080"

# 使用 OpenOIDC 生成的新凭据
OAUTH_CLIENT_ID="<OpenOIDC_生成的_Client_ID>"
OAUTH_CLIENT_SECRET="<OpenOIDC_生成的_Client_Secret>"

# 回调地址保持不变
OAUTH_REDIRECT_URI="http://192.168.31.187:3001/api/auth/callback/custom"
```

3. **修改 auth.ts 配置**

`MeowzExam/auth.ts` 需要适配 OpenOIDC 的 OIDC 标准端点：

```typescript
export const config = {
  adapter: PrismaAdapter(prisma) as Adapter,
  trustHost: true,
  providers: [
    {
      id: "openoidc",
      name: "OpenOIDC",
      type: "oidc",
      // 使用标准 OIDC 发现端点
      wellKnown: `${oauthBaseUrl}/.well-known/openid-configuration`,
      clientId: process.env.OAUTH_CLIENT_ID!,
      clientSecret: process.env.OAUTH_CLIENT_SECRET!,
      authorization: {
        params: {
          scope: "openid profile email",
        },
      },
      profile(profile) {
        console.log('[OpenOIDC] Profile:', profile)
        return {
          id: profile.sub,
          name: profile.name || null,
          email: profile.email || null,
          image: profile.picture || null,
          // ... 其他字段
        }
      },
      checks: ["pkce", "state"],  // OpenOIDC 支持 PKCE
    }
  ],
  // ... 其余配置不变
}
```

4. **配置准入策略**（可选）

在 OpenOIDC 后台为 MeowzExam 设置：
- 最低置信等级：Lv2（邮箱验证 + 任意社交账号）
- 必须绑定：可选
- 附加条件：可选

这样可以确保只有经过验证的用户才能访问考试系统。

#### 2.2 beacon-api 改造

**当前配置**（`beacon-api/.env`）:
```env
OAUTH_BASE_URL=https://oauth.mzyd.work
```

**改造步骤**:

1. **在 OpenOIDC 创建应用**
   - 应用名称："Beacon API"
   - Redirect URI: `https://your-domain.com/auth/callback`（根据实际情况）
   - 获取凭据

2. **修改 beacon-api 配置**

更新 `beacon-api/.env`:
```env
# OpenOIDC 地址
OAUTH_BASE_URL=http://localhost:8080

# OpenOIDC 凭据
OAUTH_CLIENT_ID=<新的_Client_ID>
OAUTH_CLIENT_SECRET=<新的_Client_Secret>
```

3. **修改 Rust 代码**（假设使用 JWT 验证）

`beacon-api/src/extractors.rs` 或类似文件:

```rust
use jsonwebtoken::{decode, decode_header, DecodingKey, Validation, Algorithm};
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
struct Claims {
    sub: String,        // 用户 ID
    email: String,      // 用户邮箱
    name: Option<String>,
    exp: usize,         // 过期时间
}

// 从 OpenOIDC 获取 JWKS 公钥
async fn get_jwks() -> Result<DecodingKey, Error> {
    let oauth_base = env::var("OAUTH_BASE_URL")?;
    let jwks_url = format!("{}/.well-known/jwks.json", oauth_base);
    
    // 获取并缓存 JWKS
    // ...
}

// 验证 JWT token
pub async fn verify_token(token: &str) -> Result<Claims, Error> {
    let key = get_jwks().await?;
    
    let mut validation = Validation::new(Algorithm::RS256);
    validation.set_audience(&[env::var("OAUTH_CLIENT_ID")?]);
    
    let token_data = decode::<Claims>(
        token,
        &key,
        &validation,
    )?;
    
    Ok(token_data.claims)
}
```

#### 2.3 beacon-toolkit 改造

需要先分析 beacon-toolkit 的类型（前端/后端/CLI工具），再提供具体方案。

---

### 阶段三：部署 OpenOIDC

#### 3.1 生产环境配置

修改 OpenOIDC `.env`:

```env
# 数据库（建议使用 PostgreSQL）
OIDC_DATABASE_DRIVER=postgres
OIDC_DATABASE_DSN=postgresql://user:pass@localhost:5432/openoidc

# Redis（用于会话和缓存）
OIDC_REDIS_ADDR=localhost:6379
OIDC_REDIS_PASSWORD=your_redis_password

# 服务器配置
OIDC_SERVER_ISSUER=https://auth.mzyd.work
OIDC_SERVER_PUBLIC_URL=https://auth.mzyd.work
OIDC_SERVER_PORT=8080

# 管理员账户
OIDC_ADMIN_EMAIL=admin@mzyd.work
OIDC_ADMIN_PASSWORD=<强密码>

# 加密密钥（32字节）
OIDC_SECRETS_CLIENT_SECRET_ENCRYPTION_KEY=<随机生成的32字节密钥>
OIDC_SECRETS_SESSION_ENCRYPTION_KEY=<随机生成的32字节密钥>

# 邮件配置（用于邮箱验证）
OIDC_EMAIL_SMTP_HOST=smtp.qq.com
OIDC_EMAIL_SMTP_PORT=587
OIDC_EMAIL_SMTP_USERNAME=your_email@qq.com
OIDC_EMAIL_SMTP_PASSWORD=<授权码>
OIDC_EMAIL_FROM_ADDRESS=noreply@mzyd.work
OIDC_EMAIL_FROM_NAME=MZYD 认证中心

# 第三方 OAuth 配置
OIDC_PROVIDER_GOOGLE_CLIENT_ID=<Google_Client_ID>
OIDC_PROVIDER_GOOGLE_CLIENT_SECRET=<Google_Secret>

OIDC_PROVIDER_GITHUB_CLIENT_ID=<GitHub_Client_ID>
OIDC_PROVIDER_GITHUB_CLIENT_SECRET=<GitHub_Secret>

# 其他支持的：Gitee, GitLab, Discord, Microsoft, Apple, Telegram, QQ, 微信等
```

#### 3.2 Docker Compose 部署

```bash
cd /data/D/Project/OpenOIDC
docker compose up -d
```

默认会启动：
- OpenOIDC 应用（端口 8080）
- PostgreSQL（内部网络）
- Redis（内部网络）

#### 3.3 域名和反向代理

Nginx 配置示例：

```nginx
server {
    listen 443 ssl http2;
    server_name auth.mzyd.work;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

### 阶段四：配置第三方绑定

在 OpenOIDC 后台配置第三方 OAuth：

1. **访问管理后台**: `https://auth.mzyd.work/admin`
2. **进入"社交渠道配置"**
3. **配置各个 Provider**:
   - Google: 填写 Client ID 和 Secret
   - GitHub: 填写 Client ID 和 Secret
   - 微信: 填写 App ID 和 App Secret
   - LinuxDO: 填写凭据

4. **启用/禁用渠道**:
   - 可以单独控制每个渠道的"允许登录"和"允许注册"开关

---

### 阶段五：测试和验证

#### 5.1 功能测试清单

- [ ] 用户邮箱注册/登录
- [ ] 邮箱验证流程
- [ ] 第三方账号绑定（Google、GitHub 等）
- [ ] MeowzExam OAuth 登录
- [ ] beacon-api JWT 验证
- [ ] 管理后台功能
- [ ] 准入策略生效
- [ ] 审计日志记录

#### 5.2 数据完整性验证

```sql
-- 检查用户数量
SELECT COUNT(*) FROM users;

-- 检查绑定数量
SELECT provider, COUNT(*) 
FROM bindings 
GROUP BY provider;

-- 检查客户端（业务应用）
SELECT * FROM clients;
```

---

## 迁移时间线

### 第 1 天：准备
- ✅ 启动 OpenOIDC 开发环境（已完成）
- 导出 mp-oauth2 数据
- 编写数据迁移脚本

### 第 2-3 天：数据迁移
- 运行数据迁移脚本
- 验证数据完整性
- 配置第三方 OAuth

### 第 4 天：业务系统改造
- 在 OpenOIDC 创建应用（MeowzExam、beacon-api）
- 修改业务系统配置
- 本地测试

### 第 5 天：部署上线
- 部署 OpenOIDC 到生产环境
- 配置域名和 SSL
- 灰度切换流量

### 第 6 天：监控和优化
- 监控登录成功率
- 处理用户反馈
- 优化性能

---

## 注意事项

### 1. 用户体验
- **无缝切换**: 迁移后用户使用原有邮箱/密码登录，体验不变
- **OAuth 绑定**: 之前绑定的 Google/GitHub 等账号会自动迁移

### 2. 密码兼容性
- mp-oauth2 使用 bcrypt（12轮）
- OpenOIDC 也使用 bcrypt（12轮）
- 密码哈希可以**直接迁移**，无需重置

### 3. 客户端凭据
- 每个业务系统需要新的 Client ID 和 Secret
- 旧的凭据在切换后失效

### 4. 会话处理
- 切换后，用户需要重新登录
- 建议在低峰期进行切换

### 5. 回滚方案
- 保留 mp-oauth2 数据库备份
- 保留 mp-oauth2 代码（Docker 镜像）
- 如有问题，可以快速回滚

---

## 优势对比

| 功能 | mp-oauth2 | OpenOIDC |
|------|-----------|----------|
| 邮箱注册/登录 | ✅ | ✅ |
| OAuth 第三方登录 | ✅ (4种) | ✅ (12种+) |
| 用户管理后台 | ✅ | ✅ |
| 置信等级模型 | ❌ | ✅ |
| 准入策略 | ❌ | ✅ |
| 风控系统 | ❌ | ✅ |
| 滥用上报 | ❌ | ✅ |
| WebAuthn/Passkey | ❌ | ✅ |
| 审计日志 | 基础 | 完善 |
| 人机验证 | ❌ | ✅ (Turnstile/hCaptcha) |
| 多租户 | ❌ | 规划中 |
| OIDC 标准 | 部分 | 完整 |

---

## 下一步行动

1. **确认迁移时间窗口**
2. **备份 mp-oauth2 数据库**
3. **运行数据迁移脚本**
4. **配置 OpenOIDC 生产环境**
5. **逐个改造业务系统**

需要我帮你开始哪个步骤吗？
