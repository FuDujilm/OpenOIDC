# 开发环境快速启动指南

## 使用一键启动脚本

### Linux / macOS

**启动开发环境：**
```bash
./dev.sh
```

**停止开发环境：**
```bash
./dev-stop.sh
```

### Windows

**启动开发环境：**
```cmd
start.bat
```

---

## 脚本功能

`dev.sh` 会自动：
1. ✅ 检查并创建 `.env` 文件
2. ✅ 检查 Go 和 npm 是否安装
3. ✅ 安装前端依赖（如果需要）
4. ✅ 启动前端开发服务器（port 5173）
5. ✅ 启动后端 API 服务器（port 8080）
6. ✅ 等待服务就绪
7. ✅ 显示访问地址和管理员账号

---

## 访问地址

启动后可以访问：

- **前端开发服务器**: http://localhost:5173
- **后端 API**: http://localhost:8080
- **管理后台**: http://localhost:8080/admin
- **OIDC 配置**: http://localhost:8080/.well-known/openid-configuration

---

## 查看日志

```bash
# 前端日志
tail -f frontend-dev.log

# 后端日志
tail -f backend-dev.log

# 实时查看两个日志
tail -f frontend-dev.log backend-dev.log
```

---

## 手动启动（不使用脚本）

### 前端
```bash
cd frontend
npm install
npm run dev
```

### 后端
```bash
export GOPROXY=https://goproxy.cn,direct
go run ./cmd/server
```

---

## 常见问题

### 端口被占用
```bash
# 查看占用端口的进程
lsof -i :5173
lsof -i :8080

# 杀死进程
kill -9 <PID>
```

### Go 依赖下载超时
```bash
# 使用国内镜像
export GOPROXY=https://goproxy.cn,direct
```

### 前端依赖安装失败
```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

---

## 生产部署

生产环境建议使用 Docker Compose：

```bash
docker compose up -d
```

详见 [README.md](./README.md)
