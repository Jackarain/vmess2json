package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
)

var (
	help         bool
	subscribe    string
	file         string
	templatefile string
)

type Aid string
type vmessJSON struct {
	Version string `json:"v"`
	Title   string `json:"ps"`
	Address string `json:"add"`
	Port    uint16 `json:"port"`
	ID      string `json:"id"`
	Aid     Aid    `json:"aid"`
	Net     string `json:"net"`
	Type    string `json:"type"`
	Host    string `json:"host"`
	Path    string `json:"path"`
	TLS     string `json:"tls"`
}

func (r *Aid) UnmarshalJSON(input []byte) error {
	var a string
	var i int
	switch input[0] {
	case '"':
		if err := json.Unmarshal(input, &a); err != nil {
			return err
		}
		*r = Aid(a)
	default:
		if err := json.Unmarshal(input, &i); err != nil {
			return err
		}
		*r = Aid(strconv.Itoa(i))
	}

	return nil
}

type templateData struct {
	Outbounds string
	Routing   string
}

func init() {
	flag.BoolVar(&help, "help", false, "help message")
	flag.StringVar(&subscribe, "subscribe", "", "v2ray subscribe url")
	flag.StringVar(&file, "file", "", "v2ray nodes, base64 encode file")
	flag.StringVar(&templatefile, "templatefile", "", "templatefile, v2ray config.json template file")
}

func main() {
	flag.Parse()
	if help || len(os.Args) == 1 {
		flag.Usage()
		return
	}

	var urls string

	if subscribe != "" {
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
		urls = string(body)
	} else if file != "" {
		body, _ := ioutil.ReadFile(file)
		body, err := base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		urls = string(body)
	} else {
		flag.Usage()
		return
	}

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
		if err != nil {
			continue
		}

		var result vmessJSON
		err = json.Unmarshal(node, &result)
		if err != nil {
			continue
		}

		tags = append(tags, result.Address)

		buf := new(bytes.Buffer)
		text.Execute(buf, result)

		ret = append(ret, buf.String())
	}

	var outbounds string
	size := len(ret)
	for i := 0; i < size; i++ {
		outbounds = outbounds + ret[i]
		if i+1 != size {
			outbounds = outbounds + ","
		} else {
			outbounds = outbounds + "\n"
		}
	}

	if templatefile == "" {
		fmt.Println(outbounds)
	}

	routing := `
	"routing": {
		"domainStrategy": "IPOnDemand",
		"balancers": [
		  {
			"tag": "balancer",
			"selector": [
	`

	for i := 0; i < size; i++ {
		routing = routing + "          \""
		routing = routing + tags[i]
		routing = routing + "\""
		if i+1 != size {
			routing = routing + ",\n"
		} else {
			routing = routing + "\n"
		}
	}

	routing = routing +
		`        ]
      }
    ],
    "rules": [
      {
        "type": "field",
        "network": "tcp,udp",
        "balancerTag": "balancer"
      }
    ]
  }`

	if templatefile == "" {
		fmt.Println(routing)
	}

	if templatefile != "" {
		body, _ := ioutil.ReadFile(templatefile)
		tmpl := template.New("template")
		tmpl = template.Must(tmpl.Parse(string(body)))
		buf := new(bytes.Buffer)
		var data templateData
		data.Outbounds = outbounds
		data.Routing = routing
		tmpl.Execute(buf, data)

		fmt.Println(buf.String())
	}
}
