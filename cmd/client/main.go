// Simple client used for testing main app
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
)

func main() {
	endpoint := "http://localhost:8080/"
	fmt.Println("Enter URL to be shortened:")

	reader := bufio.NewReader(os.Stdin)
	long, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}
	long = strings.TrimSuffix(long, "\n")

	client := resty.New()
	response, err := client.
		R().
		SetHeader("Content-Type", "text/plain").
		SetBody(long).
		Post(endpoint)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Response status ", response.Status())
	responseString := string(response.Body())
	fmt.Println(responseString)
}
