package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
)

var (
	spiPort = flag.String("port", "", "SPI port to use")
)

func main() {
	flag.Parse()

	// Initialize periph.io library
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Open a SPI port.
	p, err := spireg.Open(*spiPort)
	if err != nil {
		log.Fatal(err)
	}
	defer p.Close()

	// Set maximum speed to 1MHz as per MCP3008 datasheet
	if err := p.LimitSpeed(1 * physic.MegaHertz); err != nil {
		log.Fatal(err)
	}

	// Convert the port into a connection to communicate with the MCP3008
	c, err := p.Connect(1*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		log.Fatal(err)
	}

	// Make a channel to receive an interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Start a ticker to read from the MCP3008 every second
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ticker.C:
			values, err := readAllChannels(c)
			if err != nil {
				log.Fatal(err)
			} else {
				fmt.Println("ADC readings:")
				for channel, data := range values {
					fmt.Printf("Channel %d: %d\n", channel, data)
				}
			}
		case <-interrupt:
			fmt.Println("Received interrupt signal, exiting...")
			return
		}
	}

}

func readMCP3008(c spi.Conn, channel int) (int, error) {
	// Send the read command for the specified channel
	tx := []byte{1, byte((8 + channel) << 4), 0}
	rx := make([]byte, 3)
	if err := c.Tx(tx, rx); err != nil {
		return -1, err
	}

	// Parse the received data
	// data := int((rx[1]&3)<<8) + int(rx[2])
	data := (int(rx[1])<<8 | int(rx[2])) & 0x3FF
	return data, nil
}

func readAllChannels(c spi.Conn) ([]int, error) {
	values := make([]int, 8)
	for channel := 0; channel < 8; channel++ {
		data, err := readMCP3008(c, channel)
		if err != nil {
			return nil, err
		}
		values[channel] = data
	}
	return values, nil
}
