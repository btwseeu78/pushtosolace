package main

import (
	"fmt"
	"os"
	"os/signal"
	"solace.dev/go/messaging"
	"solace.dev/go/messaging/pkg/solace/config"
	"solace.dev/go/messaging/pkg/solace/resource"
	"strconv"
	"time"
)

func getEnv(key, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func main() {
	brokerConfig := config.ServicePropertyMap{
		config.TransportLayerPropertyHost:                getEnv("SOLACE_HOST", "tcp://localhost:55555,tcp://localhost:55554"),
		config.ServicePropertyVPNName:                    getEnv("SOLACE_VPN", "default"),
		config.AuthenticationPropertySchemeBasicPassword: getEnv("SOLACE_PASSWORD", "default"),
		config.AuthenticationPropertySchemeBasicUserName: getEnv("SOLACE_USERNAME", "default"),
	}
	messagingService, err := messaging.NewMessagingServiceBuilder().
		FromConfigurationProvider(brokerConfig).
		WithTransportSecurityStrategy(config.NewTransportSecurityStrategy().WithCertificateValidation(false, true, "path_to_trusted_stor_dir", "")).
		Build()

	if err != nil {
		panic(err)
	}

	// Connect to the messaging service
	if err := messagingService.Connect(); err != nil {
		panic(err)
	}
	fmt.Println("Connected to the broker? ", messagingService.IsConnected())
	directPublisher, buildError := messagingService.CreateDirectMessagePublisherBuilder().Build()
	if buildError != nil {
		panic(err)
	}
	starErr := directPublisher.Start()
	if starErr != nil {
		panic(err)
	}
	fmt.Println("Direct Publisher running? ", directPublisher.IsRunning())
	fmt.Println("\n===Interrupt (CTR+C) to stop publishing===\n")
	messageBody := "Hello from Go Direct Publisher Sample"
	messageBuilder := messagingService.MessageBuilder().
		WithProperty("application", "samples").
		WithProperty("language", "go")
	msgSeqNum := 0
	go func() {
		for directPublisher.IsReady() {
			msgSeqNum++
			message, err := messageBuilder.BuildWithStringPayload(messageBody + "-->" + strconv.Itoa(msgSeqNum))
			if err != nil {
				panic(err)
			}
			topic := resource.TopicOf("test-keda-topic")
			// Publish on dynamic topic with dynamic body
			publishErr := directPublisher.Publish(message, topic)
			if publishErr != nil {
				panic(publishErr)
			}
			fmt.Println("Message Topic: ", topic.GetName())
			// fmt.Printf("Published message: %s\n", message)
			time.Sleep(1 * time.Second)

		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until an OS interrupt signal is received.
	<-c

	// Terminate the Direct Receiver
	directPublisher.Terminate(1 * time.Second)
	fmt.Println("\nDirect Publisher Terminated? ", directPublisher.IsTerminated())
	// Disconnect the Message Service
	messagingService.Disconnect()
	fmt.Println("Messaging Service Disconnected? ", !messagingService.IsConnected())

}
