package main

import (
	"net"

	"github.com/golang/glog"
)

type Socket struct {
	conn    net.Conn
	Path    string
	Channel chan []byte
}

func (s *Socket) Write(msg []byte) {
	conn, err := net.Dial("unixgram", s.Path)
	if err != nil {
		glog.Errorf("%s", err.Error())
		return
	}
	defer conn.Close()

	s.conn = conn
	s.Channel = make(chan []byte)

	_, err = s.conn.Write(msg)
	if err != nil {
		glog.Errorf("Unable to write to socket: %s", err.Error())
		return
	}
}
