#!/bin/bash
# Coding Plan Mask 控制脚本

set -e

APP_NAME="coding-plan-mask"
INSTALL_DIR="/opt/project/${APP_NAME}"
CONFIG_DIR="${INSTALL_DIR}/config"
CONFIG_FILE="${CONFIG_DIR}/config.toml"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_installed() {
    if [ ! -f "${INSTALL_DIR}/bin/${APP_NAME}" ]; then
        log_error "${APP_NAME} 未安装，请先运行: make install"
        exit 1
    fi
}

start() {
    check_installed
    log_info "启动 ${APP_NAME}..."
    systemctl start ${APP_NAME}
    sleep 1
    status
}

stop() {
    check_installed
    log_info "停止 ${APP_NAME}..."
    systemctl stop ${APP_NAME}
    log_info "已停止"
}

restart() {
    check_installed
    log_info "重启 ${APP_NAME}..."
    systemctl restart ${APP_NAME}
    sleep 1
    status
}

status() {
    check_installed
    echo -e "\n${BLUE}服务状态:${NC}"
    systemctl status ${APP_NAME} --no-pager || true

    echo -e "\n${BLUE}配置信息:${NC}"
    if [ -f "${CONFIG_FILE}" ]; then
        # 显示配置（隐藏敏感信息) - 使用 toml 格式解析
        python3 -c "
import sys
try:
    # 磀单解析 TOML
    with open('${CONFIG_FILE}', 'r') as f:
        content = f.read()

    lines = content.split('\n')
    in_section = False
    current_section = ''

    for line in lines:
        stripped = line.strip()
        if not stripped or stripped.startswith('#'):
            print(line)
            continue

        if stripped.startswith('[') and not stripped.startswith('[['):
            in_section = True
            current_section = stripped.strip('[]')
            print(line)
        elif '=' in stripped:
            key, value = stripped.split('=', 1)
            key = key.strip()
            value = value.strip()

            # 飰藏敏感信息
            if key in ['api_key', 'local_api_key'] and value and value != '\"\"':
                if len(value) > 8:
                    masked = value[:4] + '****' + value[-4:]
                    print(f'{key} = {masked}')
                else:
                    print(f'{key} = ****')
            else:
                print(f'{key} = {value}')
        else:
            print(line)
except Exception as e:
    print(f'读取配置失败: {e}')
" 2>/dev/null || cat "${CONFIG_FILE}"
    else
        log_warn "配置文件不存在"
    fi
}

logs() {
    check_installed
    journalctl -u ${APP_NAME} -f --no-pager
}

config() {
    check_installed
    if [ -z "$1" ]; then
        # 显示配置
        if [ -f "${CONFIG_FILE}" ]; then
            cat "${CONFIG_FILE}"
        else
            log_warn "配置文件不存在"
        fi
    elif [ -n "$1" ] && [ -n "$2" ]; then
        # 设置配置项 - 更新 TOML 文件
        log_info "设置 $1 = $2"

        # 更新 TOML 文件
        python3 -c "
import re

with open('${CONFIG_FILE}', 'r') as f:
    content = f.read()

# 查找并更新配置项
key = '${1}'
value = '${2}'
pattern = r'^' + re.escape(key) + r'\s*=\s*[^\n]*'
if value in ['true', 'false'] or value.isdigit():
    replacement = key + ' = ' + value
else:
    replacement = key + ' = \"' + value + '\"'
new_content = re.sub(pattern, replacement, content, flags=re.MULTILINE)

with open('${CONFIG_FILE}', 'w') as f:
    f.write(new_content)

print('已更新配置')
"
        log_info "配置已更新，请重启服务: $0 restart"
    else
        log_error "用法: $0 config <key> <value>"
    fi
}

info() {
    check_installed

    # 读取配置
    HOST=$(python3 -c "
import re
with open('${CONFIG_FILE}', 'r') as f:
    content = f.read()
    match = re.search(r'^listen_host\s*=\s*\"([^\"]+)\"', content, re.MULTILINE)
    print(match.group(1) if match else '127.0.0.1')
" 2>/dev/null || echo "127.0.0.1")

    PORT=$(python3 -c "
import re
with open('${CONFIG_FILE}', 'r') as f:
    content = f.read()
    match = re.search(r'^listen_port\s*=\s*(\d+)', content, re.MULTILINE)
    print(match.group(1) if match else '8787')
" 2>/dev/null || echo "8787")

    LOCAL_API_KEY=$(python3 -c "
import re
with open('${CONFIG_FILE}', 'r') as f:
    content = f.read()
    match = re.search(r'^local_api_key\s*=\s*\"([^\"]*)\"', content, re.MULTILINE)
    print(match.group(1) if match else '')
" 2>/dev/null || echo "")

    # JSON 格式输出
    if [ "$1" = "--json" ]; then
        python3 -c "
import json
print(json.dumps({
    'base_url': 'http://${HOST}:${PORT}/v1',
    'api_key': '${LOCAL_API_KEY}'
}, indent=2))
"
        return
    fi

    echo -e "\n${BLUE}本地连接信息:${NC}"
    echo "----------------------------------------"
    echo -e "Base URL:    ${GREEN}http://${HOST}:${PORT}/v1${NC}"
    if [ -n "$LOCAL_API_KEY" ]; then
        echo -e "API Key:    ${GREEN}${LOCAL_API_KEY}${NC}"
    else
        echo -e "API Key:    ${YELLOW}(未设置，公开模式)${NC}"
    fi
    echo "----------------------------------------"
}

test_api() {
    check_installed

    if ! systemctl is-active --quiet ${APP_NAME}; then
        log_error "服务未运行"
        exit 1
    fi

    # 读取配置
    LOCAL_API_KEY=$(python3 -c "
import re
with open('${CONFIG_FILE}', 'r') as f:
    content = f.read()
    match = re.search(r'^local_api_key\s*=\s*\"([^\"]*)\"', content, re.MULTILINE)
    print(match.group(1) if match else '')
" 2>/dev/null)

    AUTH_HEADER=""
    if [ -n "$LOCAL_API_KEY" ]; then
        log_info "使用本地 API Key 进行认证"
        AUTH_HEADER="Authorization: Bearer $LOCAL_API_KEY"
    else
        log_warn "未配置本地 API Key，以公开模式测试"
    fi

    echo -e "\n${BLUE}测试模型列表...${NC}"
    if [ -n "$AUTH_HEADER" ]; then
        curl -s -H "$AUTH_HEADER" http://127.0.0.1:8787/v1/models | python3 -m json.tool
    else
        curl -s http://127.0.0.1:8787/v1/models | python3 -m json.tool
    fi

    echo -e "\n${BLUE}测试聊天补全...${NC}"
    if [ -n "$AUTH_HEADER" ]; then
        curl -s http://127.0.0.1:8787/v1/chat/completions \
            -H "Content-Type: application/json" \
            -H "$AUTH_HEADER" \
            -d '{
                "model": "glm-4-flash",
                "messages": [{"role": "user", "content": "说你好"}],
                "max_tokens": 50
            }' | python3 -m json.tool
    else
        curl -s http://127.0.0.1:8787/v1/chat/completions \
            -H "Content-Type: application/json" \
            -d '{
                "model": "glm-4-flash",
                "messages": [{"role": "user", "content": "说你好"}],
                "max_tokens": 50
            }' | python3 -m json.tool
    fi
}

enable() {
    check_installed
    log_info "设置开机自启..."
    systemctl enable ${APP_NAME}
    log_info "已启用开机自启"
}

disable() {
    check_installed
    log_info "取消开机自启..."
    systemctl disable ${APP_NAME}
    log_info "已取消开机自启"
}

edit() {
    check_installed
    if [ -f "${CONFIG_FILE}" ]; then
        ${EDITOR:-vim} "${CONFIG_FILE}"
    else
        log_error "配置文件不存在: ${CONFIG_FILE}"
    fi
}

usage() {
    echo "Coding Plan Mask 控制脚本

用法: $0 <命令> [参数]

命令:
    start       启动服务
    stop        停止服务
    restart     重启服务
    status      查看服务状态
    logs        查看日志 (实时)
    config      查看或修改配置
                $0 config api_key YOUR_KEY        设置 Coding Plan API Key
                $0 config local_api_key YOUR_KEY  设置本地 API Key
                $0 config provider zhipu          设置服务商
                $0 config listen_port 8787        设置端口
    info        显示本地连接信息
                $0 info --json                 JSON 格式输出
    test        测试 API 连接
    edit        编辑配置文件 (使用 \$EDITOR 或 vim)
    enable      设置开机自启
    disable     取消开机自启

配置文件: ${CONFIG_FILE}

支持的服务商:
    zhipu      智谱 GLM (open.bigmodel.cn)
    zhipu_v2   智谱 GLM (api.z.ai)
    aliyun     阿里云百炼
    minimax    MiniMax
    deepseek   DeepSeek
    moonshot   Moonshot (Kimi)

示例:
    # 配置并启动
    $0 config provider zhipu
    $0 config api_key sk-xxxxx
    $0 config local_api_key sk-local-xxx
    $0 start

    # 查看连接信息
    $0 info
"
}

case "${1:-}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    logs)
        logs
        ;;
    config)
        config "${2:-}" "${3:-}"
        ;;
    info)
        info "${2:-}"
        ;;
    test)
        test_api
        ;;
    edit)
        edit
        ;;
    enable)
        enable
        ;;
    disable)
        disable
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        usage
        exit 1
        ;;
esac
