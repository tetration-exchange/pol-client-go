# -*- coding: utf-8 -*-
import socket
import os, os.path
import time

if os.path.exists( "/tmp/policy.sock" ):
  os.remove( "/tmp/policy.sock" )

print "Opening socket..."
server = socket.socket( socket.AF_UNIX, socket.SOCK_DGRAM )
server.bind("/tmp/policy.sock")

print "Listening..."
while True:
  datagram = server.recv(8192)
  if not datagram:
    break
  else:
    print "-" * 20
    print datagram
    if "DONE" == datagram:
      break
print "-" * 20
print "Shutting down..."
server.close()
os.remove( "/tmp/policy.sock" )
print "Done"
