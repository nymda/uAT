package main

import (
	"bufio"
	"bytes"
	"fmt"
	"time"

	"go.bug.st/serial"
)

func processSentence(sentence string) {
	// Placeholder for processing NMEA sentences
	// You can parse the sentence and extract relevant information here
	fmt.Printf("Processing GPS sentence: %s\n", sentence)
}

func gpsRead(portName string, baud int) {
	mode := &serial.Mode{BaudRate: baud}
	port, err := serial.Open(portName, mode)
	if err != nil {
		fmt.Printf("Error opening GPS port %s: %s\n", portName, err)
		return
	}
	defer port.Close()

	//nmea sentences should be sent every second
	port.SetReadTimeout(2000 * time.Millisecond)

	reader := bufio.NewReader(port)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Printf("Error reading from GPS port %s: %s\n", portName, err)
			return
		}

		// Trim CR/LF and surrounding whitespace
		sentence := bytes.TrimSpace(line)
		if len(sentence) == 0 {
			continue
		}

		// filter out anything invalid that doesn't start with '$'
		if sentence[0] != '$' {
			continue
		}

		// print for debugging
		//fmt.Printf("Received GPS sentence: %s\n", sentence)
		processSentence(string(sentence))
	}
}
