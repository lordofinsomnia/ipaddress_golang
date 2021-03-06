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

type servicesConf struct {
	Services []struct {
		Name string `json:"Name"`
		Port int    `json:"Port"`
	} `json:"Services"`
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
}

var mailConfig jsonConfig
var servicesConfig servicesConf
var tos map[string]mail.Address
var from mail.Address

var smtpServer string
var senderMailAddress string
var senderPassword string
var mailBody string

func encodeRFC2047(String string) string {
	// use mail's rfc2047 to encode any string
	addr := mail.Address{Name: "", Address: ""}
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

		header := make(map[string]string)
		header["From"] = from.String()
		header["To"] = ""
		for _, v := range tos {
			header["To"] += v.String() + ", "
		}
		header["Subject"] = mailConfig.Title
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

func main() {

	flag.StringVar(&smtpServer, "IP_ADDRESS_DEAMON_SMTP_SERVER", "smtp.server.address", "a string")
	flag.StringVar(&senderMailAddress, "IP_ADDRESS_DEAMON_USERNAME", "username", "a string")
	flag.StringVar(&senderPassword, "IP_ADDRESS_DEAMON_PASSWORD", "password", "a string")
	flag.StringVar(&mailBody, "IP_ADDRESS_DEAMON_MAIL_BODY_FILE", "body.html", "filename to mail body")
	flag.Parse()

	resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.WriteString("\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	readMailConfig()
	readServicesConfig()
	initMailConfig()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))

	sendIt := true

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println("error parsing filepath")
		log.Fatal(err)
	}
	ipConfigFilePath := dir + "/ipconfig.conf"
	mailTemplateFilePath := dir + "/" + mailBody

	fmt.Println("deamon path: " + dir)
	fmt.Println("ip config file path: " + ipConfigFilePath)
	fmt.Println("mailTemplate file path: " + mailTemplateFilePath)

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
		pageServices := make([]servicesType, len(servicesConfig.Services))
		for i, v := range servicesConfig.Services {
			pageServices[i] = servicesType{Name: v.Name, Port: v.Port, Link: packServiceLink(ipaddress, v.Name, v.Port), LinkDescription: v.Name}
		}

		fmt.Println(pageServices)

		htmlpage := &page{IPAddress: ipaddress, Services: pageServices}
		mailTemplate, err := template.ParseFiles(mailTemplateFilePath)
		if err != nil {
			fmt.Println("error loading:" + mailTemplateFilePath)
			log.Fatal(err)
		}

		var doc bytes.Buffer
		mailTemplate.Execute(&doc, htmlpage)
		s := doc.String()
		fmt.Println(s)
		ioutil.WriteFile(dir+"/mail.html", []byte(s), 0644)
		sendMail(s)
	}
}

func readMailConfig() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	confFile, err := ioutil.ReadFile(dir + "/conf.json")
	if err != nil {
		fmt.Println("error loading configuration")
		log.Fatal(err)
	}

	json.Unmarshal(confFile, &mailConfig)

	for _, v := range mailConfig.MailingList {
		fmt.Printf("to: %+v, email:%+v\n", v.To, v.Email)
	}
}

func readServicesConfig() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	confFile, err := ioutil.ReadFile(dir + "/services.json")
	if err != nil {
		fmt.Println("error loading configuration")
		log.Fatal(err)
	}

	json.Unmarshal(confFile, &servicesConfig)

	for _, v := range servicesConfig.Services {
		fmt.Printf("name: %+v, port:%+v\n", v.Name, v.Port)
	}
}

func initMailConfig() {
	tos = make(map[string]mail.Address)
	for k, v := range mailConfig.MailingList {
		tos[string(k)] = mail.Address{Name: v.To, Address: v.Email}
	}
	from.Address = mailConfig.From.Email
	from.Name = mailConfig.From.From
}

func packServiceLink(ipaddress string, serviceName string, servicePort int) string {
	return "http://" + ipaddress + ":" + strconv.Itoa(servicePort)
}
