package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"reflect"

	"github.com/Shopify/sarama"
	"github.com/golang/glog"

	"github.com/tetration-exchange/pol-client-go/TetrationNetworkPolicyProto"

	"github.com/golang/protobuf/proto"
	"github.com/ttacon/chalk"
)

// KafkaConfig Kafka configuration struct
type KafkaConfig struct {
	HostArray         []string
	Topic             string
	SecureKafkaEnable bool
}

// KafkaHandle Kafka Client
type KafkaHandle struct {
	config        *KafkaConfig
	kafkaClient   sarama.Client
	kafkaConsumer sarama.Consumer
	kafkaConfig   *sarama.Config
	controlChan   chan bool
}

// NewKafkaHandle method to initialize the ClientHandle with sane defaults
func NewKafkaHandle(config *KafkaConfig) *KafkaHandle {
	return &KafkaHandle{config: config}
}

//EnableSecureKafka Enable Secure Kafka config
func (k *KafkaHandle) EnableSecureKafka(ClientCertificateFile string, ClientPrivateKeyFile string, KafkaCAFile string) error {
	k.kafkaConfig.Net.TLS.Enable = true
	k.kafkaConfig.Net.TLS.Config = &tls.Config{}
	// TLS version needs to be 1.2
	k.kafkaConfig.Net.TLS.Config.MinVersion = tls.VersionTLS12
	k.kafkaConfig.Net.TLS.Config.MaxVersion = tls.VersionTLS12
	k.kafkaConfig.Net.TLS.Config.PreferServerCipherSuites = true
	k.kafkaConfig.Net.TLS.Config.InsecureSkipVerify = true

	// Handle client cert and also client key
	//TODO add support for password protected key?
	if ClientCertificateFile != "" && ClientPrivateKeyFile != "" {
		glog.V(1).Infof("Client Cert file: %s, Client Cert Key: %s\n", ClientCertificateFile, ClientPrivateKeyFile)
		cert, err := tls.LoadX509KeyPair(ClientCertificateFile, ClientPrivateKeyFile)

		if err != nil {
			glog.Warningf("Error loading certificats: %s", err)
			return err
		}
		glog.V(1).Infoln("Added root Cert in tls config")
		k.kafkaConfig.Net.TLS.Config.Certificates = []tls.Certificate{cert}
	}
	tlsCertPool := x509.NewCertPool()

	// CA file
	if KafkaCAFile != "" {
		glog.V(1).Infof("Kafka CA: %s\n", KafkaCAFile)
		kafkaCaCertFile, err := ioutil.ReadFile(KafkaCAFile)
		if err != nil {
			glog.Warningf("Failed to read kafka Certificate Authority file %s\n", err)
			return err
		}
		if !tlsCertPool.AppendCertsFromPEM(kafkaCaCertFile) {
			glog.Warningln("Failed to append certificates from Kafka Certificate Authority file")
			return nil
		}
		glog.V(1).Infoln("Added root Cert in tls config")
	}
	k.kafkaConfig.Net.TLS.Config.RootCAs = tlsCertPool
	return nil
}

// Initialize the Kafka client, connecting to Kafka and running some sanity checks
func (k *KafkaHandle) Initialize(ClientCertificateFile string, ClientPrivateKeyFile string, KafkaCAFile string, brokersIP []string) error {
	var err error

	if len(brokersIP) == 0 {
		glog.Warningf("Invalid Broker IP")
		return fmt.Errorf("Invalid Broker IP")
	}
	k.kafkaConfig = sarama.NewConfig()

	k.kafkaConfig.ClientID = "policy-client"
	if k.config.SecureKafkaEnable {
		err = k.EnableSecureKafka(ClientCertificateFile, ClientPrivateKeyFile, KafkaCAFile)
		if err != nil {
			glog.Warningf("Failed to enable Secure Kafka err %s\n", err.Error())
			return fmt.Errorf("Failed to enable secure kafka err %s", err.Error())
		}
	}

	// Create a new Kafka client - failure to connect to Kafka is a fatal error.
	k.kafkaClient, err = sarama.NewClient(brokersIP, k.kafkaConfig)
	if err != nil {
		glog.Warningf("Failed to connect to kafka (%s)", err)
		return fmt.Errorf("Failed to connect to kafka (%s)", err)
	}

	k.kafkaConsumer, err = sarama.NewConsumerFromClient(k.kafkaClient)
	if err != nil {
		glog.Errorf("Failed to start consumer (%s)", err)
	}
	defer k.kafkaConsumer.Close()

	return nil
}

func processInventoryItems(inventoryFilters []*TetrationNetworkPolicyProto.InventoryGroup, prettyPrint bool) *map[string]string {
	resp := make(map[string]string)
	// Define some color
	lime := chalk.Green.NewStyle().WithBackground(chalk.Black).WithTextStyle(chalk.Bold)

	for _, f := range inventoryFilters {
		if prettyPrint {
			fmt.Printf("%sQuery: %s\n%s\n", lime, chalk.Reset, f.GetQuery())
			fmt.Printf("%sMembers: %s\n", lime, chalk.Reset)
		}

		resp[f.GetId()] = f.GetQuery()

		members := f.GetInventoryItems()
		if len(members) > 0 {
			for _, m := range members {
				switch reflect.TypeOf(m.Address) {
				case reflect.TypeOf(&TetrationNetworkPolicyProto.InventoryItem_AddressRange{}):
					// This is an address range
					a := m.GetAddressRange()

					start := a.GetStartIpAddr()
					end := a.GetEndIpAddr()

					if a.GetAddrFamily() == TetrationNetworkPolicyProto.IPAddressFamily_IPv4 {
						// This is IPv4
						if prettyPrint {
							fmt.Printf("range %s - %s\n", net.IPv4(start[0], start[1], start[2], start[3]).String(), net.IPv4(end[0], end[1], end[2], end[3]).String())
						}
					}

				case reflect.TypeOf(&TetrationNetworkPolicyProto.InventoryItem_IpAddress{}):
					// This is an IP Prefix
					a := m.GetIpAddress()
					ipAddr := a.GetIpAddr()
					if a.GetAddrFamily() == TetrationNetworkPolicyProto.IPAddressFamily_IPv4 {
						// This is IPv4
						if prettyPrint {
							fmt.Printf("prefix %s/%d\n", net.IPv4(ipAddr[0], ipAddr[1], ipAddr[2], ipAddr[3]).String(), a.GetPrefixLength())
						}
					}
				}

			}
		} else {
			if prettyPrint {
				fmt.Println("No Members")
			}
		}
		if prettyPrint {
			fmt.Printf("----\n\n")
		}
	}

	return &resp
}

func processIntents(intents []*TetrationNetworkPolicyProto.Intent, filters map[string]string, prettyPrint bool) map[string]map[string]string {
	// Define some color
	lime := chalk.Green.NewStyle().WithBackground(chalk.Black).WithTextStyle(chalk.Bold)

	var resp = map[string]map[string]string{}

	if len(intents) > 0 {
		for _, i := range intents {
			if prettyPrint {
				fmt.Printf("%sIntent: %s\n", lime, chalk.Reset)
			}

			flowFilter := i.GetFlowFilter()
			resp[i.GetId()] = make(map[string]string)
			resp[i.GetId()]["provider"] = filters[flowFilter.GetProviderFilterId()]
			resp[i.GetId()]["consumer"] = filters[flowFilter.GetConsumerFilterId()]
			resp[i.GetId()]["action"] = i.GetAction().String()

			fmt.Printf("Consumer: %s\n", filters[flowFilter.GetConsumerFilterId()])
			fmt.Printf("Provider: %s\n", filters[flowFilter.GetProviderFilterId()])
			fmt.Printf("Action: %s\n", i.GetAction().String())
			fmt.Printf("Ports / Protocol: %+v\n", flowFilter.GetProtocolAndPorts())
		}
	}

	return resp
}

func processUpdate(kafkaUpdate TetrationNetworkPolicyProto.KafkaUpdate) {
	tenantName := kafkaUpdate.GetTenantNetworkPolicy().GetTenantName()
	glog.Infof("Processing an update for seq %d and tenant %s...\n", kafkaUpdate.GetSequenceNum(), tenantName)

	networkPolicy := kafkaUpdate.GetTenantNetworkPolicy().GetNetworkPolicy()
	blueOnWhite := chalk.Blue.NewStyle().WithBackground(chalk.White)

	for _, p := range networkPolicy {
		fmt.Println(blueOnWhite, "Here comes a Network Policy Update", chalk.Reset)
		fmt.Println(chalk.Blue, "Network Policy Default Action: ", p.GetCatchAll().GetAction().String(), chalk.Reset)

		// First process all InventoryFilters, Intents will reference InventoryFilters
		fmt.Println(chalk.Red, "Start Inventory Filters", chalk.Reset)
		filters := p.GetInventoryFilters()
		filterMap := processInventoryItems(filters, true)
		fmt.Println(chalk.Red, "Done with Inventory Filters", chalk.Reset)
		fmt.Printf("--------------------\n\n")

		// Handle intents now
		fmt.Println(chalk.Red, "Start Intents", chalk.Reset)
		intents := p.GetIntents()
		processIntents(intents, *filterMap, true)
		//spew.Dump(intentMap)
		fmt.Println(chalk.Red, "Done with Intents", chalk.Reset)
	}
}

func processMessage(msg *sarama.ConsumerMessage) {
	kafkaUpdate := TetrationNetworkPolicyProto.KafkaUpdate{}
	err := proto.Unmarshal(msg.Value, &kafkaUpdate)

	switch kafkaUpdate.GetType() {
	case TetrationNetworkPolicyProto.KafkaUpdate_UPDATE_START:
		glog.Infoln("Message type: Update Started")
		processUpdate(kafkaUpdate)

	case TetrationNetworkPolicyProto.KafkaUpdate_UPDATE_END:
		glog.Infoln("Message type: Update End")
	case TetrationNetworkPolicyProto.KafkaUpdate_UPDATE:
		// If update is too big it will be split in multiple updates. UPDATE will be the subsequent one until UPDATE_END
		// There could be 0 or more updates
		glog.Infoln("Message type: Update")
		processUpdate(kafkaUpdate)

	default:
		glog.Infoln("Message type: Unknown")
	}
	//spew.Dump(myTest)
	if err != nil {
		glog.Errorf("unmarshaling error: %s", err)
	}
}

func consumerLoop(cons sarama.Consumer, topic string, part int32, socket *Socket) {
	fmt.Printf("Consuming Topic %s Partition %d \n", topic, part)
	partitionConsumer, err := cons.ConsumePartition(topic, part, sarama.OffsetOldest)
	if err != nil {
		panic(err)
	}

	glog.V(1).Infof("high water mark offset %d\n", partitionConsumer.HighWaterMarkOffset())

	for {
		select {
		case msg := <-partitionConsumer.Messages():
			if socket != nil {
				// If sockets are enabled, push the proto to the socket, no further processing
				socket.Write(msg.Value)
			} else {
				// If socket is not enabled, go to the example protobuff processing
				processMessage(msg)
			}
			glog.Infof("Consumed message offset %d on partition %d\n", msg.Offset, part)
		}
	}
}

func main() {
	flag.Parse()
	glog.Infoln("Starting go-policy...")
	config := NewConfig()

	kafkaConfig := new(KafkaConfig)
	// Topic is provided in the topics.txt file
	kafkaConfig.Topic = config.KafkaTopic
	kafkaConfig.SecureKafkaEnable = config.KafkaSSL
	kafkaHandle := NewKafkaHandle(kafkaConfig)

	var socket *Socket

	// KafkaCert / KafkaKey / KafkaRootCA and KafkaBroker are provided by Tetration
	// for more info, see the README
	glog.V(1).Infof("Initializing Kafka...")
	err := kafkaHandle.Initialize(config.KafkaCert, config.KafkaKey, config.KafkaRootCA, config.KafkaBroker)
	if err != nil {
		glog.Errorf("Kafka Initialization failed: %s", err.Error())
		return
	}

	glog.V(1).Infof("Getting Partitions...")
	partitions, err := kafkaHandle.kafkaConsumer.Partitions(kafkaConfig.Topic)
	if err != nil {
		glog.Errorf("Getting Kafka partition failed: %s", err.Error())
		return
	}
	glog.V(2).Infof("Topic %s has %d partitions\n", kafkaConfig.Topic, len(partitions))

	//TODO do I really need to use offset? Seems like queue is not draining...
	//var startOffset, endOffset int64

	if config.SocketEnabled {
		socket = new(Socket)
		socket.Path = config.SocketLocation
	}

	// Message can arrive on any partition so we need to consume all partitions
	for _, part := range partitions {
		cons, err := sarama.NewConsumerFromClient(kafkaHandle.kafkaClient)
		if err != nil {
			panic(err)
		}
		go consumerLoop(cons, kafkaConfig.Topic, part, socket)
	}

	// This is for a demo, so keep the program running until some hits enter
	// Note that we use go routine above, if you kill the program no more messages
	fmt.Scanln()
	fmt.Println("done")
}
