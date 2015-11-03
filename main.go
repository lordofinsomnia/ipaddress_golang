package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type jsonConfig struct {
	Title string `json:"title"`
	From  struct {
		From  string `json:"from"`
		Email string `json:"email"`
	} `json:"from"`
	MailingList []struct {
		Email string `json:"email"`
		To    string `json:"to"`
	} `json:"mailing list"`
}

type servicesType struct {
	Name            string
	Link            string
	LinkDescription string
	Port            int
}
type page struct {
	IPAddress string
	Services  []servicesType
	/*Services []struct {
		Name string
		Link string
		Port int
	}*/
}

var config jsonConfig
var tos map[string]mail.Address
var from mail.Address

var smtpServer string
var senderMailAddress string
var senderPassword string

func encodeRFC2047(String string) string {
	// use mail's rfc2047 to encode any string
	addr := mail.Address{String, ""}
	return strings.Trim(addr.String(), " <>")
}

func sendMail(body string) {
	if smtpServer != "" && senderMailAddress != "" && senderPassword != "" {
		auth := smtp.PlainAuth(
			"",
			senderMailAddress,
			senderPassword,
			smtpServer,
		)

		//firstMail := findFirst(tos)

		header := make(map[string]string)
		header["From"] = from.String()
		header["To"] = ""
		for _, v := range tos {
			header["To"] += v.String() + ", "
		}

		header["Subject"] = encodeRFC2047(config.Title)
		header["MIME-Version"] = "1.0"

		header["Content-Type"] = "text/html; charset=\"utf-8\""
		header["Content-Transfer-Encoding"] = "base64"

		message := ""
		for k, v := range header {
			message += fmt.Sprintf("%s: %s\r\n", k, v)
		}

		message += "\r\n" + base64.StdEncoding.EncodeToString([]byte(body))

		fmt.Println(message)
		//tos[0].Address

		// Connect to the server, authenticate, set the sender and recipient,
		// and send the email all in one step.
		err := smtp.SendMail(
			smtpServer+":25",
			auth,
			from.Address,
			[]string{from.Address},
			[]byte(message),
			//[]byte("This is the email body."),
		)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("configuration missing!")
		fmt.Println("IP_ADDRESS_DEAMON_SMTP_SERVER=" + smtpServer)
		fmt.Println("IP_ADDRESS_DEAMON_USERNAME=" + senderMailAddress)
		fmt.Println("IP_ADDRESS_DEAMON_PASSWORD=" + senderPassword)
		os.Exit(1)

	}
}

func findFirst(m map[string]mail.Address) mail.Address {
	fmt.Println("find first start")
	var someValue mail.Address
	for _, someValue := range m {
		//fmt.Println("key: " + key + ", address: " + someValue.Address + ", from: " + someValue.Name)
		fmt.Println("find first end someValue found! Returning " + someValue.String())
		return someValue
	}
	fmt.Println("find first end someValue not found! Returning nil")
	return someValue
}

func main() {

	flag.StringVar(&smtpServer, "IP_ADDRESS_DEAMON_SMTP_SERVER", "smtp.server.address", "a string")
	flag.StringVar(&senderMailAddress, "IP_ADDRESS_DEAMON_USERNAME", "username", "a string")
	flag.StringVar(&senderPassword, "IP_ADDRESS_DEAMON_PASSWORD", "password", "a string")
	flag.Parse()

	resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.WriteString("\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	readConfig()
	initConfig()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))

	sendIt := true

	var ipConfigFilePath string
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println("error parsing filepath")
		log.Fatal(err)
	}
	ipConfigFilePath = dir + "/ipconfig.conf"

	fmt.Println("deamon path: " + dir)
	fmt.Println("ip config file path: " + ipConfigFilePath)

	_, err = os.Open(ipConfigFilePath)
	if err != nil {
		ioutil.WriteFile(ipConfigFilePath, body, 0644)

	} else {
		var readBody []byte
		readBody, _ = ioutil.ReadFile(ipConfigFilePath)
		if string(readBody) == string(body) {
			fmt.Println("ip has not been changed")

			sendIt = false
		} else {
			fmt.Println("ip has been changed")
			fmt.Println("old ip:" + string(readBody))
			fmt.Println("new ip:" + string(body))
			ioutil.WriteFile(ipConfigFilePath, body, 0644)
		}
	}

	if sendIt {
		ipaddress := strings.Replace(string(body), "\n", "", -1)
		services := make(map[string]string)
		services["gitlab"] = packServiceLink(ipaddress, "gitlab", 10080)
		services["ssh"] = packServiceLink(ipaddress, "ssh", 443)
		services["torrent"] = packServiceLink(ipaddress, "torrent", 18080)

		pageServices := []servicesType{servicesType{Name: "ssh",
			Link:            services["ssh"],
			Port:            443,
			LinkDescription: (services["ssh"])},
			servicesType{Name: "gitlab",
				Link:            services["gitlab"],
				Port:            10080,
				LinkDescription: services["gitlab"]},
			servicesType{Name: "torrent",
				Link:            services["torrent"],
				Port:            18080,
				LinkDescription: services["torrent"]}}

		fmt.Println(pageServices)

		htmlpage := &page{IPAddress: ipaddress, Services: pageServices}
		mailBody, err := template.ParseFiles(dir + "/body.html")
		if err != nil {
			fmt.Println("error loading body.html")
			log.Fatal(err)
		}

		var doc bytes.Buffer
		mailBody.Execute(&doc, htmlpage)
		s := doc.String()
		fmt.Println(s)
		ioutil.WriteFile(dir+"/mail.html", []byte(s), 0644)
		sendMail(s)
	}
}

func readConfig() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	confFile, err := ioutil.ReadFile(dir + "/conf.json")
	if err != nil {
		fmt.Println("error loading configuration")
		log.Fatal(err)
	}

	json.Unmarshal(confFile, &config)

	for _, v := range config.MailingList {
		fmt.Printf("to: %+v, email:%+v\n", v.To, v.Email)
	}
}

func initConfig() {
	tos = make(map[string]mail.Address)
	for k, v := range config.MailingList {
		tos[string(k)] = mail.Address{v.To, v.Email}
	}
	from.Address = config.From.Email
	from.Name = config.From.From
}

func packServiceLink(ipaddress string, serviceName string, servicePort int) string {
	return "http://" + ipaddress + ":" + strconv.Itoa(servicePort)
}
