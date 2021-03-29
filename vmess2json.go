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
					},
					"wsSettings": {
						"connectionReuse": true,
						"path": "{{.Path}}"{{if .Host}},
						"headers": {
							"Host": "{{.Host}}"
						}
						{{end}}
					}
				},
				"mux": {
					"enabled": true
				},
				"tag": "{{.Address}}"
			}`))

	var ret []string
	var tags []string
	scanner := bufio.NewScanner(strings.NewReader(urls))
	for scanner.Scan() {
		vmess := scanner.Text()
		link := vmess[8:]
		node, err := base64.StdEncoding.DecodeString(link)
		fmt.Println(string(node))

		var result vmessJSON
		err = json.Unmarshal(node, &result)
		if err != nil {
			fmt.Println("err:", err.Error(), "link:", string(node))
			continue
		}

		tags = append(tags, result.Address)

		buf := new(bytes.Buffer)
		text.Execute(buf, result)

		ret = append(ret, buf.String())
	}

	size := len(ret)
	for i := 0; i < size; i++ {
		fmt.Printf(ret[i])
		if i + 1 != size {
			fmt.Printf(",")
		} else {
			fmt.Println("")
		}
	}

	fmt.Println(`
  "routing": {
    "domainStrategy": "IPOnDemand",
    "balancers": [
      {
        "tag": "balancer",
        "selector": [
	`)

        for i := 0; i < size; i++ {
		fmt.Printf("          \"")
                fmt.Printf(tags[i])
		fmt.Printf("\"")
                if i + 1 != size {
                        fmt.Printf(",\n")
                } else {
                        fmt.Println("")
                }
        }

	fmt.Println(`
        ]
      }
    ],
    "rules": [
      {
        "type": "field",
        "network": "tcp,udp",
        "balancerTag": "balancer"
      }
    ]
  }
`)

}

