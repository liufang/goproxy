//支持http和https
//https://tools.ietf.org/html/draft-luotonen-web-proxy-tunneling-01

package main

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/hsyan2008/go-logger/logger"
)

func startHttp(config Config) {
	if config.Addr == "" {
		logger.Warn("no addr")
		return
	}
	lister, err := net.Listen("tcp", config.Addr)
	if err != nil {
		logger.Warn("http/https listen error:", err)
	}
	logger.Info("start http/https listen ", config.Addr, "overssh", config.Overssh)

	for {
		conn, err := lister.Accept()
		if err != nil {
			continue
		}
		logger.Debug("accept connect")
		go handHttp(conn, config.Overssh)
	}
}

func handHttp(conn net.Conn, overssh bool) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(err)

			buf := make([]byte, 1<<20)
			num := runtime.Stack(buf, false)
			logger.Warn(num, string(buf))

			_ = conn.Close()
		}
	}()

	r := bufio.NewReader(conn)

	req, err := http.ReadRequest(r)
	if err != nil {
		logger.Error("http ReadRequest error:", err)
		return
	}

	req.Header.Del("Proxy-Connection")
	//否则远程连接不会关闭，导致Copy卡住
	req.Header.Set("Connection", "close")

	if req.Method == "CONNECT" {
		con, err := dial(req.Host, overssh)
		if err != nil {
			logger.Warn(err)
			return
		}
		logger.Info(req.Host, "连接建立成功")

		_, _ = io.WriteString(conn, "HTTP/1.0 200 Connection Established\r\n\r\n")

		go copyNet(conn, con)
		go copyNet(con, conn)
	} else {
		logger.Info("no connect")
		hosts := strings.Split(req.Host, ":")
		if len(hosts) == 1 {
			hosts = append(hosts, "80")
		}
		con, err := dial(strings.Join(hosts, ":"), overssh)
		if err != nil {
			logger.Warn(req.Host, err)
			return
		}
		logger.Info(req.Host, "连接建立成功")
		err = req.Write(con)
		if err != nil {
			logger.Warn(err)
			return
		}
		go copyNet(conn, con)
	}
}
