# go-port-forward

一个简单的端口转发程序，当前只支持tcp协议

## 构建

```shell
go mod tidy
go build
```

## 添加端口转发配置

在当前目录下创建一个配置文件 config.yaml，格式如下：

```yaml
port_forwards:
  - local_port: 8080
    remote_addr: "100.88.88.202"
    remote_port: 8080
    protocol_type: "tcp"
  - local_port: 7001
    remote_addr: "100.88.88.202"
    remote_port: 7001
    protocol_type: "tcp"
```

## 启动

```shell
sh monitor start
```