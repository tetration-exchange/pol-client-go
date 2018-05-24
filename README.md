# go-policy

This is an example of the policy client for Tetration written in Go. You can find an example in Java here: [](https://github.com/tetration-exchange/pol-client-java). The objective of this example is to show how to use the different policy objects, this code is not designed to be used in production.

## Building

```shell
go build
```

### Configuration file

In order to start go-home you need a configuration file located in the conf directory (same folder as the binary).

There parameters are self explanatory.
```
[kafka]
topic = ""Tnp-XX"
ssl = true
rootca = "cert/KafkaCA.cert"
cert = "cert/KafkaConsumerCA.cert"
key = "cert/KafkaConsumerPrivateKey.key"
brokers = ["1.1.1.1:9093"]

[socket]
enabled = false
location = "/tmp/policy.sock"
```

This will configure a basic setup to integrate go-policy with Tetration. The socket options transports the protobuf from Kafka to the a unix socket, for example for use with python (test.py example)

## Running

```shell
./go-policy -logtostderr=true
```

Note: you can enable verbose logging via `./go-policy -logtostderr=true -v 2`

Example output:
```
 Here comes a Network Policy Update 
 Start Inventory Filters 
Query: 
type=and, filter=[{type=eq, field=vrf_id, value=42},{type=contains, field=os, value=MS},unsafeSubnetValue=null]
Members: 
range 172.31.219.130 - 172.31.219.130
----

Query: 
type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.200.1.0/24, subnet=172.200.1.0/24},unsafeSubnetValue=[172.200.1.0/24]]
Members: 
prefix 172.200.1.0/24
----

Query: 
type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.200.1.0/24, subnet=172.200.1.0/24},{type=eq, field=host_name, value=server-red},unsafeSubnetValue=null]
Members: 
range 172.200.1.250 - 172.200.1.250
----

Query: 
type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.200.1.0/24, subnet=172.200.1.0/24},{type=eq, field=host_name, value=server-blue},unsafeSubnetValue=null]
Members: 
range 172.200.1.202 - 172.200.1.202
----

Query: 
type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.31.0.0/16, subnet=172.31.0.0/16},unsafeSubnetValue=[172.31.0.0/16]]
Members: 
prefix 172.31.0.0/16
----

Query: 
type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.200.1.0/24, subnet=172.200.1.0/24},{type=contains, field=host_name, value=client},unsafeSubnetValue=null]
Members: 
range 172.200.1.100 - 172.200.1.100
range 172.200.1.150 - 172.200.1.150
----

Query: 
type=and, filter=[{type=eq, field=vrf_id, value=42},{type=eq, field=host_name, value=client-blue},unsafeSubnetValue=null]
Members: 
range 172.31.219.27 - 172.31.219.27
range 172.200.1.100 - 172.200.1.100
range 192.168.122.1 - 192.168.122.1
----

Query: 
type=and, filter=[{type=contains, field=host_name, value=Windows},type=and, filter=[{type=eq, field=vrf_id, value=42},{type=contains, field=os, value=MS},{type=eq, field=tags_is_internal, value=true},unsafeSubnetValue=null],unsafeSubnetValue=null]
Members: 
range 172.31.219.130 - 172.31.219.130
----

 Done with Inventory Filters 
--------------------

 Start Intents 
Intent: 
Consumer: 
Provider: type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.31.0.0/16, subnet=172.31.0.0/16},unsafeSubnetValue=[172.31.0.0/16]]
Action: ALLOW
Ports / Protocol: []
Intent: 
Consumer: type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.31.0.0/16, subnet=172.31.0.0/16},unsafeSubnetValue=[172.31.0.0/16]]
Provider: 
Action: ALLOW
Ports / Protocol: []
Intent: 
Consumer: type=and, filter=[{type=contains, field=host_name, value=Windows},type=and, filter=[{type=eq, field=vrf_id, value=42},{type=contains, field=os, value=MS},{type=eq, field=tags_is_internal, value=true},unsafeSubnetValue=null],unsafeSubnetValue=null]
Provider: 
Action: ALLOW
Ports / Protocol: []
Intent: 
Consumer: 
Provider: type=and, filter=[{type=contains, field=host_name, value=Windows},type=and, filter=[{type=eq, field=vrf_id, value=42},{type=contains, field=os, value=MS},{type=eq, field=tags_is_internal, value=true},unsafeSubnetValue=null],unsafeSubnetValue=null]
Action: ALLOW
Ports / Protocol: [protocol:UDP port_ranges:<start_port:3389 end_port:3389 > ]
Intent: 
Consumer: 
Provider: type=and, filter=[{type=contains, field=host_name, value=Windows},type=and, filter=[{type=eq, field=vrf_id, value=42},{type=contains, field=os, value=MS},{type=eq, field=tags_is_internal, value=true},unsafeSubnetValue=null],unsafeSubnetValue=null]
Action: ALLOW
Ports / Protocol: [protocol:TCP port_ranges:<start_port:3389 end_port:3389 > ]
Intent: 
Consumer: type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.200.1.0/24, subnet=172.200.1.0/24},{type=contains, field=host_name, value=client},unsafeSubnetValue=null]
Provider: type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.200.1.0/24, subnet=172.200.1.0/24},{type=eq, field=host_name, value=server-blue},unsafeSubnetValue=null]
Action: ALLOW
Ports / Protocol: [protocol:TCP port_ranges:<start_port:90 end_port:90 > ]
Intent: 
Consumer: type=and, filter=[{type=eq, field=vrf_id, value=42},{type=eq, field=host_name, value=client-blue},unsafeSubnetValue=null]
Provider: type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.200.1.0/24, subnet=172.200.1.0/24},unsafeSubnetValue=[172.200.1.0/24]]
Action: DROP
Ports / Protocol: [protocol:ICMP ]
Intent: 
Consumer: type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.200.1.0/24, subnet=172.200.1.0/24},unsafeSubnetValue=[172.200.1.0/24]]
Provider: type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.200.1.0/24, subnet=172.200.1.0/24},unsafeSubnetValue=[172.200.1.0/24]]
Action: ALLOW
Ports / Protocol: [protocol:ICMP ]
Intent: 
Consumer: type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.200.1.0/24, subnet=172.200.1.0/24},{type=contains, field=host_name, value=client},unsafeSubnetValue=null]
Provider: type=and, filter=[{type=eq, field=vrf_id, value=42},{type=subnet, field=ip, value=172.200.1.0/24, subnet=172.200.1.0/24},{type=eq, field=host_name, value=server-red},unsafeSubnetValue=null]
Action: ALLOW
Ports / Protocol: [protocol:TCP port_ranges:<start_port:80 end_port:80 > ]
 Done with Intents 
```

## License

Please refer to the file *LICENSE.pdf* in same directory of this README file.