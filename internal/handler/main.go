package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	gonet "github.com/THREATINT/go-net"
	"go.acim.net/hcdns"
)

type DynDnsErrorCode string

const (
	BadAuth  DynDnsErrorCode = "badauth"
	NotFQDN  DynDnsErrorCode = "notfqdn"
	NoHost   DynDnsErrorCode = "nohost"
	NumHost  DynDnsErrorCode = "numhost"
	Abuse    DynDnsErrorCode = "abuse"
	BadAgent DynDnsErrorCode = "badagent"
	DNSErr   DynDnsErrorCode = "dnserr"
	Error911 DynDnsErrorCode = "911"
)

type DynDnsError struct {
	Code DynDnsErrorCode
}

func (dynDnsError *DynDnsError) Error() string {
	return string(dynDnsError.Code)
}

type DynDnsHandler struct {
	DynDnsRequest func(w http.ResponseWriter, r *http.Request) error
}

func (handler DynDnsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := handler.DynDnsRequest(w, r)
	if err != nil {
		switch err.Error() {
		case string(BadAuth):
			w.WriteHeader(http.StatusUnauthorized)
		case string(NotFQDN):
			w.WriteHeader(http.StatusBadRequest)
		case string(NoHost):
			w.WriteHeader(http.StatusBadRequest)
		case string(NumHost):
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		case string(Abuse):
			w.WriteHeader(http.StatusForbidden)
		case string(BadAgent):
			w.WriteHeader(http.StatusBadRequest)
		case string(DNSErr):
			w.WriteHeader(http.StatusBadGateway)
		case string(Error911):
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(err.Error()))
	}
}

func DynDnsRequest(w http.ResponseWriter, r *http.Request) error {
	// Check for Basic authorization header
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
		fmt.Println("Missing Basic authorization header")
		return &DynDnsError{Code: BadAuth}
	}

	// Decode username (zone) and password (token)
	credentials, err := base64.StdEncoding.DecodeString((strings.Split(r.Header.Get("Authorization"), " ")[1]))
	if err != nil || strings.Count(string(credentials), ":") != 1 {
		fmt.Println("Unable to decode Basic authorization header")
		return &DynDnsError{Code: BadAuth}
	}
	zone, token := string(credentials[:strings.IndexByte(string(credentials), ':')]), string(credentials[strings.IndexByte(string(credentials), ':')+1:])
	if os.Getenv("ZONE") != "" && zone != os.Getenv("ZONE") {
		// deepcode ignore ClearTextLogging: zone is not a secret
		fmt.Println("Zone " + zone + " not configured")
		return &DynDnsError{Code: BadAgent}
	}

	if os.Getenv("TOKEN") != "" && token != os.Getenv("TOKEN") {
		fmt.Println("Wrong token provided")
		return &DynDnsError{Code: BadAuth}
	}

	client := hcdns.NewClient(token)
	ctx := context.Background()

	hcdnsZone, err := client.ZoneByName(ctx, zone)
	if err != nil {
		fmt.Println("Error getting zone: " + err.Error())
		return &DynDnsError{Code: NoHost}
	}
	hcdnsRecords, err := hcdnsZone.Records(ctx)
	if err != nil {
		fmt.Println("Error getting records: " + err.Error())
		return &DynDnsError{Code: DNSErr}
	}
	fmt.Println("Found zone " + hcdnsZone.Name)

	// Check for hostname and IP parameters
	hostname, ips := r.URL.Query().Get("hostname"), strings.Split(r.URL.Query().Get("myip"), ",")
	ipv4, ipv6 := "", ""
	for _, s := range ips {
		ip := net.ParseIP(s)
		if ip != nil {
			if gonet.IsIPv4(ip) {
				ipv4 = ip.String()
			}
			if gonet.IsIPv6(ip) {
				ipv6 = ip.String()
			}
		}
	}
	if gonet.IsFQDN(hostname) == false {
		fmt.Println("Provided Hostname is not a FQDN")
		return &DynDnsError{Code: NotFQDN}
	}
	if !strings.HasSuffix(hostname, zone) || (ipv4 == "" && ipv6 == "") {
		fmt.Println("Missing or wrong hostname or IP parameter")
		return &DynDnsError{Code: BadAgent}
	}
	fmt.Println("Received valid request for hostname " + hostname + ":")

	ipv4result, ipv6result := "", ""
	recordName := strings.TrimSuffix(hostname, "."+zone)

	for _, record := range hcdnsRecords {
		if record.Name == recordName {
			if record.Type == hcdns.A && ipv4 != "" {
				if record.Value == ipv4 {
					fmt.Println("\tIPv4 already up to date")
					ipv4result = "nochg"
				} else {
					fmt.Println("\tUpdating IPv4 to " + ipv4 + " (was " + record.Value + ")")
					err = record.UpdateValueAndTTL(ctx, ipv4, 60*time.Second)
					if err != nil {
						fmt.Println("Error updating record: " + err.Error())
						return &DynDnsError{Code: DNSErr}
					}
					ipv4result = "good"
				}
			}
			if record.Type == hcdns.AAAA && ipv6 != "" {
				if record.Value == ipv6 {
					fmt.Println("\tIPv6 already up to date")
					ipv6result = "nochg"
				} else {
					fmt.Println("\tUpdating IPv6 to " + ipv6 + " (was " + record.Value + ")")
					err = record.UpdateValueAndTTL(ctx, ipv6, 60*time.Second)
					if err != nil {
						fmt.Println("Error updating record: " + err.Error())
						return &DynDnsError{Code: DNSErr}
					}
					ipv6result = "good"
				}
			}
		}
	}
	if ipv4result == "" && ipv4 != "" {
		fmt.Println("\tCreating new IPv4 record " + ipv4)
		_, err = hcdnsZone.CreateRecordWithTTL(ctx, hcdns.A, recordName, ipv4, 60*time.Second)
		if err != nil {
			fmt.Println("Error creating record: " + err.Error())
			return &DynDnsError{Code: DNSErr}
		}
		ipv4result = "good"
	}
	if ipv6result == "" && ipv6 != "" {
		fmt.Println("\tCreating new IPv6 record " + ipv6)
		_, err = hcdnsZone.CreateRecordWithTTL(ctx, hcdns.AAAA, recordName, ipv6, 60*time.Second)
		if err != nil {
			fmt.Println("Error creating record: " + err.Error())
			return &DynDnsError{Code: DNSErr}
		}
		ipv6result = "good"
	}

	w.WriteHeader(http.StatusOK)
	newIPs := ""
	if ipv4 != "" && ipv6 != "" {
		newIPs = ipv4 + ", " + ipv6
	} else if ipv4 != "" {
		newIPs = ipv4
	} else if ipv6 != "" {
		newIPs = ipv6
	}
	if ipv4result == "good" || ipv6result == "good" {
		// deepcode ignore XSS: validation against IP pattern is performed
		w.Write([]byte("good " + newIPs))
	} else {
		// deepcode ignore XSS: validation against IP pattern is performed
		w.Write([]byte("nochg " + newIPs))
	}

	return nil
}
