package main

import (
	"context"
	"fmt"
	"github.com/go-yaml/yaml"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

// Config 结构定义了配置文件的结构
type Config struct {
	PortForwards []PortForward `yaml:"port_forwards"`
}

// PortForward 结构定义了单个端口转发的配置
type PortForward struct {
	LocalPort    int    `yaml:"local_port"`
	RemoteAddr   string `yaml:"remote_addr"`
	RemotePort   int    `yaml:"remote_port"`
	ProtocolType string `yaml:"protocol_type"`
}

func main() {
	// 读取配置文件
	config, err := readConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动端口转发服务
	var wg sync.WaitGroup
	for _, pf := range config.PortForwards {
		wg.Add(1)
		go func(pf PortForward) {
			defer wg.Done()
			startPortForward(ctx, pf, cancel)
		}(pf)
	}
	wg.Wait()
}

// readConfig 从配置文件中读取配置信息
func readConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}

// startPortForward 启动端口转发服务
func startPortForward(ctx context.Context, pf PortForward, cancel context.CancelFunc) {
	sourcePort := fmt.Sprintf(":%v", pf.LocalPort)
	destinationAddress := fmt.Sprintf("%s:%v", pf.RemoteAddr, pf.RemotePort)

	listener, err := net.Listen("tcp", sourcePort)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", sourcePort, err)
	}
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)
	log.Printf("PID %d: Listening on %s and forwarding to %s", os.Getpid(), sourcePort, destinationAddress)

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("PID %d: Received signal: %s. Shutting down.", os.Getpid(), sig)
		cancel() // 取消上下文，停止新的连接接受
		_ = listener.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("PID %d: Context cancelled, shutting down listener on %s", os.Getpid(), sourcePort)
			return
		default:
			clientConn, err := listener.Accept()
			if err != nil {
				if strings.Contains(err.Error(), net.ErrClosed.Error()) {
					log.Printf("PID %d: Temporary accept error: %v", os.Getpid(), err)
					return // 退出循环，避免持续打印错误日志
				}
				log.Printf("PID %d: Failed to accept connection: %v", os.Getpid(), err)
				break
			}
			log.Printf("PID %d: Accepted connection from %s", os.Getpid(), clientConn.RemoteAddr())
			go handleConnection(ctx, clientConn, destinationAddress)
		}
	}
}

// handleConnection 处理连接
func handleConnection(ctx context.Context, clientConn net.Conn, destinationAddress string) {
	defer func(clientConn net.Conn) {
		_ = clientConn.Close()
	}(clientConn)
	log.Printf("PID %d: Handling connection to %s", os.Getpid(), destinationAddress)

	serverConn, err := net.Dial("tcp", destinationAddress)
	if err != nil {
		log.Printf("PID %d: Failed to connect to destination %s: %v", os.Getpid(), destinationAddress, err)
		return
	}
	defer func(serverConn net.Conn) {
		_ = serverConn.Close()
	}(serverConn)

	doneChan := make(chan struct{})

	go copyData(ctx, clientConn, serverConn, doneChan)
	go copyData(ctx, serverConn, clientConn, doneChan)

	select {
	case <-ctx.Done():
		log.Printf("PID %d: Context cancelled, closing connections", os.Getpid())
	case <-doneChan:
		log.Printf("PID %d: Data transfer completed, closing connections", os.Getpid())
	}
}

// copyData 复制数据
func copyData(ctx context.Context, dst net.Conn, src net.Conn, doneChan chan struct{}) {
	log.Printf("PID %d: Starting data copy from %s to %s", os.Getpid(), src.RemoteAddr(), dst.RemoteAddr())
	_, err := io.Copy(dst, src)
	if err != nil {
		select {
		case <-ctx.Done():
			log.Printf("PID %d: Context cancelled, stopping data copy from %s to %s", os.Getpid(), src.RemoteAddr(), dst.RemoteAddr())
			return
		default:
			if strings.Contains(err.Error(), "use of closed network connection") {
				log.Printf("PID %d: Connection closed during data copy from %s to %s", os.Getpid(), src.RemoteAddr(), dst.RemoteAddr())
			} else {
				log.Printf("PID %d: Error copying data from %s to %s: %v", os.Getpid(), src.RemoteAddr(), dst.RemoteAddr(), err)
			}
		}
	}
	log.Printf("PID %d: Completed data copy from %s to %s", os.Getpid(), src.RemoteAddr(), dst.RemoteAddr())
	doneChan <- struct{}{}
}
