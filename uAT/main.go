package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"time"

	"go.bug.st/serial"
)

func scan(baud int) {
	fmt.Println("Starting serial port probe...")

	ports, err := serial.GetPortsList()
	if err != nil {
		fmt.Printf("Error listing serial ports: %s\n", err)
		return
	}

	for _, s := range ports {
		mode := &serial.Mode{BaudRate: baud}
		port, err := serial.Open(s, mode)
		if err != nil {
			continue
		}

		readCh := make(chan []byte, 1)
		errCh := make(chan error, 1)

		go func() {
			var n int
			var err error
			n, err = port.Write([]byte("AT\r"))

			response := make([]byte, 256)

			port.SetReadTimeout(100 * time.Millisecond)

			for i := 0; i < 5; i++ {
				buf := make([]byte, 256)
				n, err = port.Read(buf)
				if err != nil {
					errCh <- err
					return
				}
				response = append(response, buf[:n]...)
			}

			readCh <- response
		}()

		select {
			case data := <-readCh:
				trimmed := bytes.Trim(data, "\x00")

				if bytes.Contains(trimmed, []byte("OK")) {
					fmt.Printf("Discovered modem on %s\n", s)
				}

				if bytes.Contains(trimmed, []byte("$G")) {
					fmt.Printf("Discovered GPS on %s\n", s)
				}
			case <-time.After(2 * time.Second):
				//fmt.Printf("Read timeout from %s\n", s)
		}

		port.Close()
	}
}

func cmdLoop(c chan string) {
	reader := bufio.NewReader(os.Stdin)
	for{
		txt, err := reader.ReadBytes('\n')
		if err != nil {
            close(c)
            return
        }
		line := bytes.TrimSpace(txt)
		if(len(line) > 0){
			c <- string(line)
		}
	}
}

func main() {
	baud := flag.Int("b", 115200, "Modem baud rate")
	port := flag.String("p", "", "Modem port")
	baudGps := flag.Int("bg", 115200, "GPS baud rate (optional)")
	portGps := flag.String("pg", "", "GPS port (optional)")
	perfScan := flag.Bool("s", false, "Scan all serial ports for modems")
	flag.Parse()

	if *perfScan {
		scan(*baud)
		if *baudGps != *baud {
			scan(*baudGps)
		}
	}

	if *port != "" {
		mod := make(chan string)
		com := make(chan string)
		go modemLoop(*port, *baud, mod)
		go cmdLoop(com)
		mod <- "TEST"
		for{
			select {
				case msg := <- mod:
					fmt.Printf(">MSG: %s\n", msg)
				case cmd := <- com:
					if(cmd == "EXIT"){ return }
					mod <- cmd
			}
		}
	}

	if *portGps != "" {
		gpsRead(*portGps, *baudGps)
		// Handle GPS port logic here
	}

}
