package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/Shopify/sarama"
	"github.com/golang/glog"

	"github.com/remiphilippe/go-policy/TetrationNetworkPolicyProto"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"
)

type KafkaConfig struct {
	HostArray []string
	Topic     string
	// Kafka producer flush interval
	Interval          time.Duration
	SecureKafkaEnable bool
}

// ClientHandle Kafka Client
type ClientHandle struct {
	config        *KafkaConfig
	kafkaClient   sarama.Client
	kafkaConsumer sarama.Consumer
	kafkaConfig   *sarama.Config
	controlChan   chan bool
}

//Factory method to initialize the ClientHandle with sane defaults
func KafkaFactory(config *KafkaConfig) *ClientHandle {
	return &ClientHandle{config: config}
}

//EnableSecureKafka Enable Secure Kafka config
func (k *ClientHandle) EnableSecureKafka(ClientCertificateFile string, ClientPrivateKeyFile string, RootCAFile string, KafkaCAFile string) error {
	k.kafkaConfig.Net.TLS.Enable = true
	k.kafkaConfig.Net.TLS.Config = &tls.Config{}
	// set TLS version to TLSv1.2
	k.kafkaConfig.Net.TLS.Config.MinVersion = tls.VersionTLS12
	k.kafkaConfig.Net.TLS.Config.MaxVersion = tls.VersionTLS12
	k.kafkaConfig.Net.TLS.Config.PreferServerCipherSuites = true
	k.kafkaConfig.Net.TLS.Config.InsecureSkipVerify = true
	glog.V(2).Infof("MinVersion: %d, MaxVersion: %d\n", k.kafkaConfig.Net.TLS.Config.MinVersion, k.kafkaConfig.Net.TLS.Config.MaxVersion)

	if ClientCertificateFile != "" && ClientPrivateKeyFile != "" {
		glog.V(2).Infof("Client Cert file: %s, Client Cert Key: %s\n", ClientCertificateFile, ClientPrivateKeyFile)
		cert, err := tls.LoadX509KeyPair(ClientCertificateFile, ClientPrivateKeyFile)
		if err != nil {
			glog.Warningf("Error loading certificats: %s", err)
			return err
		}
		glog.V(2).Infoln("Added root Cert in tls config")
		k.kafkaConfig.Net.TLS.Config.Certificates = []tls.Certificate{cert}
	}
	tlsCertPool := x509.NewCertPool()
	if RootCAFile != "" {
		glog.V(2).Infof("Root CA: %s\n", RootCAFile)
		caCertFile, err := ioutil.ReadFile(RootCAFile)
		if err != nil {
			glog.Warningln("Failed to read intermediate Certificate Authority file %s", err)
			return err
		}
		if !tlsCertPool.AppendCertsFromPEM(caCertFile) {
			glog.Warningln("Failed to append certificates from intermediate Certificate Authority file")
			return nil
		}
		glog.V(2).Infoln("Added root Cert in tls config")
	}
	if KafkaCAFile != "" {
		glog.V(2).Infof("Kafka CA: %s\n", KafkaCAFile)
		kafkaCaCertFile, err := ioutil.ReadFile(KafkaCAFile)
		if err != nil {
			glog.Warningf("Failed to read kafka Certificate Authority file %s\n", err)
			return err
		}
		if !tlsCertPool.AppendCertsFromPEM(kafkaCaCertFile) {
			glog.Warningln("Failed to append certificates from Kafka Certificate Authority file")
			return nil
		}
		glog.V(2).Infoln("Added root Cert in tls config")
	}
	k.kafkaConfig.Net.TLS.Config.RootCAs = tlsCertPool
	return nil
}

// Initialize the Kafka client, connecting to Kafka and running some sanity checks
func (k *ClientHandle) Initialize(ClientCertificateFile string,
	ClientPrivateKeyFile string, RootCAFile string, KafkaCAFile string,
	brokersIP []string) error {
	var err error

	if len(brokersIP) == 0 {
		glog.Warningf("Invalid Broker IP")
		return fmt.Errorf("Invalid Broker IP")
	}
	k.kafkaConfig = sarama.NewConfig()

	k.kafkaConfig.ClientID = "quarantine-client"
	if k.config.SecureKafkaEnable {
		err = k.EnableSecureKafka(ClientCertificateFile, ClientPrivateKeyFile,
			RootCAFile, KafkaCAFile)
		if err != nil {
			glog.Warningf("Failed to enable Secure Kafka err %s\n", err.Error())
			return fmt.Errorf("Failed to enable secure kafka err %s", err.Error())
		}
	}

	// Create a new Kafka client - failure to connect to Kafka is a fatal error.
	k.kafkaClient, err = sarama.NewClient(brokersIP, k.kafkaConfig)
	if err != nil {
		glog.Warningf("Failed to connect to kafka (%s)", err)
		return fmt.Errorf("Failed to connect to kafka (%s)",
			err)
	}

	k.kafkaConsumer, err = sarama.NewConsumerFromClient(k.kafkaClient)
	if err != nil {
		fmt.Printf("Failed to start consumer (%s)", err)
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
	/*
		pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
		pflag.Parse()

		goflag.CommandLine.Parse([]string{})
	*/

	config := ConfigFactory()

	kafkaConfig := new(KafkaConfig)
	kafkaConfig.Topic = config.KafkaTopic
	kafkaConfig.SecureKafkaEnable = config.KafkaSSL
	kafkaHandle := KafkaFactory(kafkaConfig)

	err := kafkaHandle.Initialize(config.KafkaCert, config.KafkaKey, "", config.KafkaRootCA, config.KafkaBroker)
	if err != nil {
		spew.Dump(err)
		return
	}

	partitions, err := kafkaHandle.kafkaConsumer.Partitions(kafkaConfig.Topic)
	if err != nil {
		spew.Dump(err)
		return
	}
	fmt.Println("-----------------------------------------------------------")
	fmt.Printf("Topic %s has %d partitions\n", kafkaConfig.Topic, len(partitions))

	//TODO do I really need to use offset? Seems like queue is not draining...
	//var startOffset, endOffset int64
	for _, part := range partitions {
		cons, err := sarama.NewConsumerFromClient(kafkaHandle.kafkaClient)
		if err != nil {
			panic(err)
		}
		go consumerLoop(cons, kafkaConfig.Topic, part)
	}
	fmt.Scanln()
	fmt.Println("done")
}
