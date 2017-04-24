package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

var (
	flag_addr = flag.String("addr", "0.0.0.0:8083", "socket proxy addr")
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	ls, err := net.Listen("tcp", *flag_addr)
	if err != nil {
		log.Panic(err)
	}

	//接受客户端连接
	for {
		conn, err := ls.Accept()
		if err != nil {
			log.Panic(err)
		}
		go socket(conn)
	}
}

func socket(client net.Conn) {
	defer client.Close()
	var b [1024]byte

	n, err := client.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}

	var addr string

	//sock5代理
	if b[0] == 0x05 {
		//回应确认代理
		client.Write([]byte{0x05, 0x00})

		n, err = client.Read(b[:])
		if err != nil {
			log.Println(err)
			return
		}
		switch b[3] {
		case 0x01:
			//解析代理ip
			type sockIP struct {
				A, B, C, D byte
				PORT       uint16
			}
			sip := sockIP{}
			if err := binary.Read(bytes.NewReader(b[4:n]), binary.BigEndian, &sip); err != nil {
				log.Println("请求解析错误")
				return
			}
			addr = fmt.Sprintf("%d.%d.%d.%d:%d", sip.A, sip.B, sip.C, sip.D, sip.PORT)
		case 0x03:
			//解析代理域名
			host := string(b[5 : n-2])
			var port uint16
			err = binary.Read(bytes.NewReader(b[n-2:n]), binary.BigEndian, &port)
			if err != nil {
				log.Println(err)
				return
			}
			addr = fmt.Sprintf("%s:%d", host, port)
		}

		server, err := net.DialTimeout("tcp", addr, time.Second*3)
		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("%s -> %s connected!", client.RemoteAddr().String(), server.RemoteAddr().String())
		defer server.Close()
		//回复确定代理成功
		client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		//转发
		go io.Copy(server, client)
		io.Copy(client, server)
	}
}
