#!/bin/bash

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)
            REMOTE_HOST="$2"
            shift 2
            ;;
        -u|--user)
            REMOTE_USER="$2"
            shift 2
            ;;
        *)
            APP_NAME="$1"
            shift
            ;;
    esac
done

# 设置默认值
REMOTE_USER="${REMOTE_USER:-houzzkit}"
REMOTE_HOST="${REMOTE_HOST:-$HA_URL}"  # 参数优先，其次环境变量
REMOTE_DIR="/var/hass/config/"

# 验证
if [ -z "$REMOTE_HOST" ]; then
    echo "用法: $0 [--host 服务器地址] [--user 用户名] 文件名"
    echo "或设置环境变量: export HA_URL=your_server"
    exit 1
fi
scp $LOCAL_JAR $REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/$APP_NAME.tmp
echo "部署 $APP_NAME 到 $REMOTE_USER@$REMOTE_HOST"