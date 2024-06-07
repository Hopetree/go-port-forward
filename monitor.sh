#!/bin/bash

# 定义go-port-forward可执行文件的路径和日志文件路径
EXECUTABLE="./go-port-forward"
LOG_FILE="./go-port-forward.log"

# 定义PID文件路径
PID_FILE="./go-port-forward.pid"

start() {
    # 检查PID文件是否存在
    if [ -f "$PID_FILE" ]; then
        echo "go-port-forward is already running."
        exit 1
    fi

    # 启动go-port-forward服务，并将日志输出追加到日志文件中
    nohup $EXECUTABLE >> $LOG_FILE 2>&1 &

    # 获取启动的进程ID并写入PID文件
    PID=$!
    echo $PID > $PID_FILE
    echo "go-port-forward started with PID: $PID"
}

stop() {
    # 检查PID文件是否存在
    if [ ! -f "$PID_FILE" ]; then
        echo "go-port-forward is not running."
        exit 1
    fi

    # 从PID文件中读取进程ID并停止进程
    PID=$(cat $PID_FILE)
    kill $PID

    # 删除PID文件
    rm $PID_FILE
    echo "go-port-forward stopped."
}

restart() {
    stop
    sleep 1
    start
}

status() {
    # 检查PID文件是否存在
    if [ -f "$PID_FILE" ]; then
        echo "go-port-forward is running with PID: $(cat $PID_FILE)"
    else
        echo "go-port-forward is not running."
    fi
}

# 根据参数调用相应的操作
case "$1" in
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
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac

exit 0
