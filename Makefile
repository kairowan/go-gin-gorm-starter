# =========================
# Makefile（跨平台稳健版）
# 用法：
#   make tidy
#   make run-api
#   make run-admin
#   make run APP=admin
#   make test
#   make print         # 打印当前生效的变量，便于排错
# =========================

# 要运行的子应用（api 或 admin）
APP ?= api
# 配置文件路径
CFG ?= ./configs/config.local.yaml

# 关键：把 CONFIG_PATH 作为环境变量导出给所有 recipe（子进程）
export CONFIG_PATH := $(CFG)
# 声明伪目标
.PHONY: tidy run run-api run-admin test print

# 打印当前变量，排查是否取到了你想要的路径
print:
	@echo APP=$(APP)
	@echo CFG=$(CFG)
	@echo CONFIG_PATH=$(CONFIG_PATH)

# 整理依赖
tidy:
	go mod tidy

# 根据 APP 变量运行（cmd/$(APP)）
run:
	@echo ">> starting $(APP) with CONFIG_PATH=$(CONFIG_PATH)"
	go run ./cmd/$(APP)

# 运行用户端 API（cmd/api）
run-api:
	@echo ">> starting api with CONFIG_PATH=$(CONFIG_PATH)"
	go run ./cmd/api

# 运行后台管理端 API（cmd/admin）
run-admin:
	@echo ">> starting admin with CONFIG_PATH=$(CONFIG_PATH)"
	go run ./cmd/admin

# 运行所有单元测试
test:
	go test ./... -v
