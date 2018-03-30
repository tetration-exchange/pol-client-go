package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/Shopify/sarama"
	"github.com/golang/glog"

	"github.com/remiphilippe/go-policy/TetrationNetworkPolicyProto"

	"github.com/golang/protobuf/proto"
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

	k.kafkaConfig.ClientID = "quarantine-client"
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

func consumerLoop(cons sarama.Consumer, topic string, part int32) {
	fmt.Printf("Consuming Topic %s Partition %d \n", topic, part)
	partitionConsumer, err := cons.ConsumePartition(topic, part, sarama.OffsetOldest)
	if err != nil {
		panic(err)
	}

	for {
		select {
		case msg := <-partitionConsumer.Messages():

			myTest := TetrationNetworkPolicyProto.KafkaUpdate{}
			err = proto.Unmarshal(msg.Value, &myTest)

			switch myTest.GetType() {
			case TetrationNetworkPolicyProto.KafkaUpdate_UPDATE_START:
				glog.Infoln("Update Started")
				tenantName := myTest.GetTenantNetworkPolicy().GetTenantName()
				networkPolicy := myTest.GetTenantNetworkPolicy().GetNetworkPolicy()

				fmt.Printf("Tenant: %s \n\n", tenantName)

				for _, p := range networkPolicy {
					fmt.Println("Policy")
					filters := p.GetInventoryFilters()
					for _, f := range filters {
						fmt.Printf("Query: %s\n", f.GetQuery())
						fmt.Println("Members")
						members := f.GetInventoryItems()
						for _, m := range members {
							start := m.GetAddressRange().GetStartIpAddr()
							end := m.GetAddressRange().GetEndIpAddr()
							if len(start) == 4 && len(end) == 4 {
								fmt.Printf("range %s - %s\n", net.IPv4(start[0], start[1], start[2], start[3]).String(), net.IPv4(end[0], end[1], end[2], end[3]).String())
							}
						}
					}

					intents := p.GetIntents()
					for _, i := range intents {
						fmt.Printf("Intent %+v\n", i)
					}
				}

			case TetrationNetworkPolicyProto.KafkaUpdate_UPDATE_END:
				glog.Infoln("Update End")
			case TetrationNetworkPolicyProto.KafkaUpdate_UPDATE:
				glog.Infoln("Update")
			}
			//spew.Dump(myTest)
			if err != nil {
				log.Fatal("unmarshaling error: ", err)
			}
			fmt.Printf("Consumed message offset %d on partition %d\n", msg.Offset, part)
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

	// Message can arrive on any partition so we need to consume all partitions
	for _, part := range partitions {
		cons, err := sarama.NewConsumerFromClient(kafkaHandle.kafkaClient)
		if err != nil {
			panic(err)
		}
		go consumerLoop(cons, kafkaConfig.Topic, part)
	}

	// This is for a demo, so keep the program running until some hits enter
	// Note that we use go routine above, if you kill the program no more messages
	fmt.Scanln()
	fmt.Println("done")
}
