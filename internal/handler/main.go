package handler

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/netip"
	"os"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/marvinruder/hetzner-dyndns/internal/logger"
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
		logger.Error("Zone not configured", "zone", zone)
		return &DynDnsError{Code: BadAgent}
	}

	if os.Getenv("TOKEN") != "" && token != os.Getenv("TOKEN") {
		logger.Error("Wrong token provided")
		return &DynDnsError{Code: BadAuth}
	}

	client := hcloud.NewClient(hcloud.WithToken(token))
	ctx := context.Background()

	hcloudZone, _, err := client.Zone.GetByName(ctx, zone)
	if err != nil || hcloudZone == nil {
		logger.Error("Error getting zone", "err", err)
		return &DynDnsError{Code: NoHost}
	}
	logger.Debug("Found zone", "zone", hcloudZone.Name)

	hostname, ips := r.URL.Query().Get("hostname"), strings.Split(r.URL.Query().Get("myip"), ",")

	// Check hostname parameter
	_, err = idna.Lookup.ToASCII(hostname)
	if err != nil || strings.Count(hostname, ".") < 2 {
		logger.Error("Provided hostname is not a FQDN", "hostname", hostname)
		return &DynDnsError{Code: NotFQDN}
	}
	if !strings.HasSuffix(hostname, zone) {
		logger.Error("Hostname is not in zone", "hostname", hostname, "zone", zone)
		return &DynDnsError{Code: NoHost}
	}
	logger.Debug("Detected hostname", "hostname", hostname)

	// Check IP parameter
	ipv4, ipv6 := "", ""
	for _, s := range ips {
		ip, err := netip.ParseAddr(s)
		if err != nil {
			logger.Error("Error parsing IP", "ip", s, "err", err)
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
	if ipv4 != "" {
		logger.Debug("Detected IPv4", "ip", ipv4)
	}
	if ipv6 != "" {
		logger.Debug("Detected IPv6", "ip", ipv6)
	}

	ipv4result, ipv6result := "", ""
	recordName := strings.TrimSuffix(hostname, "."+zone)

	hcloudRRSets, _, err := client.Zone.ListRRSets(ctx, hcloudZone, hcloud.ZoneRRSetListOpts{
		Name: recordName,
		Type: []hcloud.ZoneRRSetType{hcloud.ZoneRRSetTypeA, hcloud.ZoneRRSetTypeAAAA}},
	)
	if err != nil {
		logger.Error("Error getting records", "err", err)
		return &DynDnsError{Code: DNSErr}
	}

	for _, rrSet := range hcloudRRSets {
		if rrSet.Type == hcloud.ZoneRRSetTypeA && ipv4 != "" {
			if rrSet.Records[0].Value == ipv4 {
				logger.Warn("IPv4 already up to date", "ip", ipv4, "hostname", hostname)
				ipv4result = "nochg"
			} else {
				logger.Info("Updating IPv4", "was", rrSet.Records[0].Value, "ip", ipv4, "hostname", hostname)
				setRecordsOpts := hcloud.ZoneRRSetSetRecordsOpts{Records: rrSet.Records}
				setRecordsOpts.Records[0].Value = ipv4
				_, _, err = client.Zone.SetRRSetRecords(ctx, rrSet, setRecordsOpts)
				if err != nil {
					logger.Error("Error updating IPv4 record", "ip", ipv4, "hostname", hostname, "err", err)
					return &DynDnsError{Code: DNSErr}
				}
				ipv4result = "good"
			}
		}
		if rrSet.Type == hcloud.ZoneRRSetTypeAAAA && ipv6 != "" {
			if rrSet.Records[0].Value == ipv6 {
				logger.Warn("IPv6 already up to date", "ip", ipv6, "hostname", hostname)
				ipv6result = "nochg"
			} else {
				logger.Info("Updating IPv6", "was", rrSet.Records[0].Value, "ip", ipv6, "hostname", hostname)
				setRecordsOpts := hcloud.ZoneRRSetSetRecordsOpts{Records: rrSet.Records}
				setRecordsOpts.Records[0].Value = ipv6
				_, _, err = client.Zone.SetRRSetRecords(ctx, rrSet, setRecordsOpts)
				if err != nil {
					logger.Error("Error updating IPv6 record", "ip", ipv6, "hostname", hostname, "err", err)
					return &DynDnsError{Code: DNSErr}
				}
				ipv6result = "good"
			}
		}
	}

	TTL := 60

	if ipv4result == "" && ipv4 != "" {
		logger.Info("Creating new IPv4 record", "ip", ipv4, "hostname", hostname)
		createRecordOpts := hcloud.ZoneRRSetCreateOpts{
			Name:    recordName,
			Type:    hcloud.ZoneRRSetTypeA,
			TTL:     &TTL,
			Records: []hcloud.ZoneRRSetRecord{{Value: ipv4}},
		}
		_, _, err = client.Zone.CreateRRSet(ctx, hcloudZone, createRecordOpts)
		if err != nil {
			logger.Error("Error creating IPv4 record", "ip", ipv4, "hostname", hostname, "err", err)
			return &DynDnsError{Code: DNSErr}
		}
		ipv4result = "good"
	}
	if ipv6result == "" && ipv6 != "" {
		logger.Info("Creating new IPv6 record", "ip", ipv6, "hostname", hostname)
		createRecordOpts := hcloud.ZoneRRSetCreateOpts{
			Name:    recordName,
			Type:    hcloud.ZoneRRSetTypeAAAA,
			TTL:     &TTL,
			Records: []hcloud.ZoneRRSetRecord{{Value: ipv6}},
		}
		_, _, err = client.Zone.CreateRRSet(ctx, hcloudZone, createRecordOpts)
		if err != nil {
			logger.Error("Error creating IPv6 record", "ip", ipv6, "hostname", hostname, "err", err)
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
