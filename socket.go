package main

import (
	"net"

	"github.com/golang/glog"
)

// Socket Structure reprensenting an unix socket
type Socket struct {
	conn    net.Conn
	Path    string
	Channel chan []byte
}

// Write Write data to a socket so it can be picked up by another software (python, shell, other..)
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
