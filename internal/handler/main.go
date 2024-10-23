package handler

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/marvinruder/hetzner-dyndns/internal/logger"
	"go.acim.net/hcdns"
	"golang.org/x/net/idna"
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
		logger.Error("Missing Basic authorization header")
		return &DynDnsError{Code: BadAuth}
	}

	// Decode username (zone) and password (token)
	credentials, err := base64.StdEncoding.DecodeString((strings.Split(r.Header.Get("Authorization"), " ")[1]))
	if err != nil || strings.Count(string(credentials), ":") != 1 {
		logger.Error("Unable to decode Basic authorization header")
		return &DynDnsError{Code: BadAuth}
	}
	zone, token := string(credentials[:strings.IndexByte(string(credentials), ':')]), string(credentials[strings.IndexByte(string(credentials), ':')+1:])
	if os.Getenv("ZONE") != "" && zone != os.Getenv("ZONE") {
		// deepcode ignore ClearTextLogging: zone is not a secret
		logger.Error("Zone " + zone + " not configured")
		return &DynDnsError{Code: BadAgent}
	}

	if os.Getenv("TOKEN") != "" && token != os.Getenv("TOKEN") {
		logger.Error("Wrong token provided")
		return &DynDnsError{Code: BadAuth}
	}

	client := hcdns.NewClient(token)
	ctx := context.Background()

	hcdnsZone, err := client.ZoneByName(ctx, zone)
	if err != nil {
		logger.Error("Error getting zone: " + err.Error())
		return &DynDnsError{Code: NoHost}
	}
	hcdnsRecords, err := hcdnsZone.Records(ctx)
	if err != nil {
		logger.Error("Error getting records: " + err.Error())
		return &DynDnsError{Code: DNSErr}
	}
	logger.Debug("Found zone " + hcdnsZone.Name)

	hostname, ips := r.URL.Query().Get("hostname"), strings.Split(r.URL.Query().Get("myip"), ",")

	// Check hostname parameter
	_, err = idna.Lookup.ToASCII(hostname)
	if err != nil || strings.Count(hostname, ".") < 2 {
		logger.Error("Provided Hostname is not a FQDN")
		return &DynDnsError{Code: NotFQDN}
	}
	if !strings.HasSuffix(hostname, zone) {
		logger.Error("Hostname is not in zone")
		return &DynDnsError{Code: NoHost}
	}
	logger.Debug("Received valid request for hostname " + hostname)

	// Check IP parameter
	ipv4, ipv6 := "", ""
	for _, s := range ips {
		ip, err := netip.ParseAddr(s)
		if err != nil {
			logger.Error("Error parsing IP: " + err.Error())
			return &DynDnsError{Code: BadAgent}
		}
		if ip.Is4() {
			ipv4 = ip.String()
		}
		if ip.Is6() {
			ipv6 = ip.String()
		}
	}
	if ipv4 == "" && ipv6 == "" {
		logger.Error("Missing IP parameter")
		return &DynDnsError{Code: BadAgent}
	}

	ipv4result, ipv6result := "", ""
	recordName := strings.TrimSuffix(hostname, "."+zone)

	for _, record := range hcdnsRecords {
		if record.Name == recordName {
			if record.Type == hcdns.A && ipv4 != "" {
				if record.Value == ipv4 {
					logger.Warn("IPv4 already up to date")
					ipv4result = "nochg"
				} else {
					logger.Info("Updating IPv4 to " + ipv4 + " (was " + record.Value + ")")
					err = record.UpdateValueAndTTL(ctx, ipv4, 60*time.Second)
					if err != nil {
						logger.Error("Error updating record: " + err.Error())
						return &DynDnsError{Code: DNSErr}
					}
					ipv4result = "good"
				}
			}
			if record.Type == hcdns.AAAA && ipv6 != "" {
				if record.Value == ipv6 {
					logger.Warn("IPv6 already up to date")
					ipv6result = "nochg"
				} else {
					logger.Info("Updating IPv6 to " + ipv6 + " (was " + record.Value + ")")
					err = record.UpdateValueAndTTL(ctx, ipv6, 60*time.Second)
					if err != nil {
						logger.Error("Error updating record: " + err.Error())
						return &DynDnsError{Code: DNSErr}
					}
					ipv6result = "good"
				}
			}
		}
	}
	if ipv4result == "" && ipv4 != "" {
		logger.Info("Creating new IPv4 record " + ipv4)
		_, err = hcdnsZone.CreateRecordWithTTL(ctx, hcdns.A, recordName, ipv4, 60*time.Second)
		if err != nil {
			logger.Error("Error creating record: " + err.Error())
			return &DynDnsError{Code: DNSErr}
		}
		ipv4result = "good"
	}
	if ipv6result == "" && ipv6 != "" {
		logger.Info("Creating new IPv6 record " + ipv6)
		_, err = hcdnsZone.CreateRecordWithTTL(ctx, hcdns.AAAA, recordName, ipv6, 60*time.Second)
		if err != nil {
			logger.Error("Error creating record: " + err.Error())
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
