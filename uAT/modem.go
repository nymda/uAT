package main

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"go.bug.st/serial"
)

func modemRead(port serial.Port) (string, error) {
	port.SetReadTimeout(250 * time.Millisecond)
	read := make([]byte, 512)
	for {
		drain := make([]byte, 128)
		n, err := port.Read(drain)
		if err != nil || n == 0 {
			break
		}
		read = append(read, drain[:n]...)
	}

	read = bytes.TrimSpace(read)
	return string(read), nil
}

func modemSms(port serial.Port, number string, message string) string {
	port.Write([]byte("AT+CMGF=1\r"))
	time.Sleep(100 * time.Millisecond)
	port.Write([]byte(fmt.Sprintf("AT+CMGS=\"%s\"\r", number)))
	time.Sleep(100 * time.Millisecond)
	port.Write([]byte(message + "\x1A"))
	read, err := modemRead(port)

	if err != nil {
		return "Modem error: " + err.Error()
	}

	if strings.Contains(read, "OK"){
		return fmt.Sprintf("Sent SMS to [%s]", number)
	} else {
		return fmt.Sprintf("Failed to send SMS: %s", strings.ReplaceAll(strings.ReplaceAll(read, "\r", ""), "\n", ""))
	}
}

func modemCallAndResponse(port serial.Port, command string, stripOK bool) (string, bool) {

	port.SetReadTimeout(250 * time.Millisecond)

	port.Write([]byte(command + "\r"))

	read, err := modemRead(port)
	if err != nil {
		return "", false
	}

	readBytes := []byte(read)
	if(stripOK){
		readBytes = bytes.Replace(readBytes, []byte("OK"), []byte(""), -1) //remove the OK response
	}
	readBytes = bytes.ReplaceAll(readBytes, []byte(command), []byte("")) //remove the command echo
	readBytes = bytes.ReplaceAll(readBytes, []byte("\r"), []byte("")) 
	readBytes = bytes.ReplaceAll(readBytes, []byte("\n"), []byte("")) 
	readBytes = bytes.TrimSpace(readBytes)
	read = string(readBytes)
	return read, true
}

func modemTest(port serial.Port, c chan string) {

	response, ok := modemCallAndResponse(port, "AT", false)
	if ok && bytes.Contains([]byte(response), []byte("OK")) {
		c <- "TEST_OK"
	} else {
		c <- "TEST_FAIL"
	}
}

func modemInfo(port serial.Port, c chan string) {
	if response, ok := modemCallAndResponse(port, "AT+CGMM", true); ok {
		c <- fmt.Sprintf("Model: %s", response)
	} else {
		c <- "Error getting model info"
	}
	if response, ok := modemCallAndResponse(port, "AT+CGMI", true); ok {
		c <- fmt.Sprintf("Manufacturer: %s", response)
	} else {
		c <- "Error getting manufacturer info"
	}
	if response, ok := modemCallAndResponse(port, "AT+CGMR", true); ok {
		c <- fmt.Sprintf("Firmware: %s", response)
	} else {
		c <- "Error getting firmware info"
	}
	if response, ok := modemCallAndResponse(port, "AT+CGSN", true); ok {
		c <- fmt.Sprintf("IMEI: %s", response)
	} else {
		c <- "Error getting IMEI info"
	}
	if response, ok := modemCallAndResponse(port, "AT+CREG?", true); ok {
		c <- fmt.Sprintf("Connected: %s", response)
	} else {
		c <- "Error getting connection status"
	}
	if response, ok := modemCallAndResponse(port, "AT+CSQ", true); ok {
		c <- fmt.Sprintf("Signal Strength: %s", response)
	} else {
		c <- "Error getting signal strength"
	}
}

func modemAwaitCall(port serial.Port, c chan string) {
	port.Write([]byte("AT+CLCC=1\r"))
	c <- "Awaiting call..."
	for {
		time.Sleep(500 * time.Millisecond)
		if response, ok := modemCallAndResponse(port, "AT+CPAS", true); ok {
			c <- response
			if(bytes.Contains([]byte(response), []byte("CPAS: 3"))){
				c <- "Incoming call detected!"
				port.Write([]byte("ATA\r"))
				return;
			}
		} 
	}
}

func modemLoop(portName string, baud int, c chan string) {
	mode := &serial.Mode{BaudRate: baud}
	port, err := serial.Open(portName, mode)
	if err != nil {
		c <- fmt.Sprintf("Error opening modem port %s: %s\n", portName, err)
		return
	}
	defer port.Close()
	
	//nmea sentences should be sent every second
	port.SetReadTimeout(2000 * time.Millisecond)

	for {
		var command string = <-c
		//split by space into subcommands
		subcommands := bytes.Split([]byte(command), []byte(" "))
		command = string(subcommands[0])

		switch command {
			case "TEST":
				modemTest(port, c)

			case "INFO":
				modemInfo(port, c)

			case "SMS":
				if len(subcommands) < 3 {
					c <- "Usage: SMS <number> <message>"
				} else {
					number := string(subcommands[1])
					message := string(bytes.Join(subcommands[2:], []byte(" ")))
					c <- modemSms(port, number, message)
				}

			case "AWAIT_CALL":
				modemAwaitCall(port, c)

			default:
				c <- fmt.Sprintf("Unknown command: %s", command)
		} 
	}
}