package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	help      bool
	subscribe string
)

type vmessJSON struct {
	Version string `json:"v"`
	Title   string `json:"ps"`
	Address string `json:"add"`
	Port    uint16 `json:"port"`
	ID      string `json:"id"`
	Aid     string `json:"aid"`
	Net     string `json:"net"`
	Type    string `json:"type"`
	Host    string `json:"host"`
	Path    string `json:"path"`
	TLS     string `json:"tls"`
}

func init() {
	flag.BoolVar(&help, "help", false, "help message")
	flag.StringVar(&subscribe, "subscribe", "", "v2ray subscribe url")
}

func main() {
	flag.Parse()
	if help || len(os.Args) == 1 {
		flag.Usage()
		return
	}

	request, err := http.NewRequest("GET", subscribe, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	v2rayHTTPClient := &http.Client{}
	response, err := v2rayHTTPClient.Do(request)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	body, err = base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	urls := string(body)
	// 输出所有vmess URL.
	fmt.Println(urls)

	// 循环输出 outbounds 数组...
	text := template.New("test")
	text = template.Must(text.Parse(`
			{
				"protocol": "vmess",
				"settings": {
					"vnext": [
					{
						"address": "{{.Address}}",
						"port": {{.Port}},
						"users": [
						{
							"email": "user@v2ray.com",
							"id": "{{.ID}}",
							"alterId": {{.Aid}},
							"security": "auto"
						}
						]
					}
					]
				},
				"streamSettings": {
					"network": "{{.Net}}",
					"security": "{{.TLS}}",
					"tlsSettings": {
					"allowInsecure": true
					}
				},
				"mux": {
					"enabled": true
				},
				"tag": "{{.Address}}"
			},`))

	scanner := bufio.NewScanner(strings.NewReader(urls))
	for scanner.Scan() {
		vmess := scanner.Text()
		url, err := url.Parse(vmess)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		if url.Host == "" {
			return
		}

		node, err := base64.StdEncoding.DecodeString(url.Host)

		var result vmessJSON
		err = json.Unmarshal(node, &result)
		if err != nil {
			continue
		}

		buf := new(bytes.Buffer)
		text.Execute(buf, result)

		fmt.Println(buf.String())
	}

}
