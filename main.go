package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"strings"

	"github.com/thoj/go-ircevent"

	"net/http"
	"strconv"
)

var (
	host     = flag.String("host", "irc.hackint.org", "Hostname of the IRC server")
	port     = flag.Int("sslport", 6697, "SSL capable port of the IRC server")
	channel  = flag.String("channel", "#fleaz", "Target to send notifications to, likely a channel.")
	nickname = flag.String("nickname", "go-prom-irc", "Nickname to assume once connected")
	gecos    = flag.String("gecos", "go-prom-irc", "Realname to assume once connected")
	cafile   = flag.String("cafile", "hackint-rootca.crt", "Path to the ca file that verifies the server certificate.")
)

func CreateFunctionNotifyFunction(bot *irc.Connection) http.HandlerFunc {

	const templateString = "[{{ .ColorStart }}{{ .Status }}{{ .ColorEnd }}:{{ .InstanceCount }}] {{ .Alert.Labels.alertname}} - {{ .Alert.Annotations.description}}"

	notificationTemplate, err := template.New("notification").Parse(templateString)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	return func(wr http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		decoder := json.NewDecoder(req.Body)

		var notification Notification

		if err := decoder.Decode(&notification); err != nil {
			log.Println(err)
			return
		}

		body, err := json.Marshal(&notification)

		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("JSON: %v", string(body))

		resolved_alerts, firing_alerts := SortAlerts(notification.Alerts)
		var resolved_hosts, firing_hosts []string
		var instance string

		// FIRING
		var alertStatus = "firing"
		for _, alert := range firing_alerts {
			instance = alert.Labels["instance"].(string)
			instance = strings.Split(instance, ".")[0]
			firing_hosts = append(firing_hosts, instance)

		}

		context := NotificationContext{
			Alert:         &notification.Alerts[0],
			Notification:  &notification,
			Status:        alertStatus,
			InstanceCount: len(firing_hosts),
			ColorStart:    getColorcode(alertStatus),
			ColorEnd:      "\x03",
		}

		var buf bytes.Buffer
		err = notificationTemplate.Execute(&buf, &context)
		bot.Privmsg(*channel, buf.String())
		bot.Privmsg(*channel, strings.Join(firing_hosts, ","))

		// RESOLVED
		alertStatus = "resolved"
		for _, alert := range resolved_alerts {
			instance = alert.Labels["instance"].(string)
			instance = strings.Split(instance, ".")[0]
			resolved_hosts = append(resolved_hosts, instance)

		}

		context = NotificationContext{
			Alert:         &notification.Alerts[0],
			Notification:  &notification,
			Status:        alertStatus,
			InstanceCount: len(resolved_hosts),
			ColorStart:    getColorcode(alertStatus),
			ColorEnd:      "\x03",
		}

		buf.Reset()
		err = notificationTemplate.Execute(&buf, &context)
		bot.Privmsg(*channel, buf.String())
		bot.Privmsg(*channel, strings.Join(resolved_hosts, ","))

	}

}

func getColorcode(status string) string {
	switch status {
	case "firing":
		return "\x0305"
	case "resolved":
		return "\x0303"
	default:
		return "\x0300"
	}
}

func main() {
	flag.Parse()

	caCertPool := x509.NewCertPool()
	caCert, err := ioutil.ReadFile(*cafile)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}
	irccon := irc.IRC(*nickname, *gecos)

	irccon.Debug = true
	irccon.UseTLS = true
	irccon.TLSConfig = tlsConfig

	RegisterHandlers(irccon)

	var server bytes.Buffer
	server.WriteString(*host)
	server.WriteString(":")
	server.WriteString(strconv.Itoa(*port))

	err = irccon.Connect(server.String())
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		http.HandleFunc("/notify", CreateFunctionNotifyFunction(irccon))
		http.ListenAndServe("127.0.0.1:8083", nil)
	}()

	irccon.Loop()
}

func RegisterHandlers(irccon *irc.Connection) {
	irccon.AddCallback("001", func(e *irc.Event) {
		log.Printf("Joining %v", channel)
		irccon.Join(*channel)
	})
	irccon.AddCallback("366", func(e *irc.Event) {})
}
