package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	host := flag.String("host", "127.0.0.1", "api host")
	port := flag.Int("port", 8888, "api port")
	flag.Parse()

	client := newAPIClient(strings.TrimSpace(*host), *port)
	printBanner()
	fmt.Printf("connected target: %s\n", client.baseURL)
	fmt.Println("type h or help to show commands")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("%s> ", promptDollar("$"))
		if !scanner.Scan() {
			break
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		shouldExit, err := client.execute(text)
		if err != nil {
			fmt.Println(errorText("error: " + err.Error()))
		}
		if shouldExit {
			break
		}
	}
}
