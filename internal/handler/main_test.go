package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func setupSuite(t *testing.T, token string, zone string) func(t *testing.T) {
	client := hcloud.NewClient(hcloud.WithToken(token))
	ctx := context.Background()
	fmt.Println("Creating test zone")
	hcloudZoneCreateResult, _, err := client.Zone.Create(ctx, hcloud.ZoneCreateOpts{Name: zone, Mode: hcloud.ZoneModePrimary})
	if err != nil {
		t.Fatal(err)
	}

	os.Setenv("ZONE", "")
	os.Setenv("TOKEN", "")

	return func(t *testing.T) {
		fmt.Println("Deleting test zone")
		client.Zone.Delete(ctx, hcloudZoneCreateResult.Zone)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestDynDnsHandler(t *testing.T) {
	zone := os.Getenv("ZONE")
	token := os.Getenv("TOKEN")
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(zone+":"+token))

	tt := []struct {
		name            string
		zoneEnv         string
		tokenEnv        string
		authHeader      string
		urlParamString  string
		statusCode      int
		dynDnsErrorCode DynDnsErrorCode
		body            string
	}{
		{
			name:            "Missing Basic authorization header (no header)",
			statusCode:      http.StatusUnauthorized,
			dynDnsErrorCode: BadAuth,
		},
		{
			name:            "Missing Basic authorization header (wrong header)",
			authHeader:      "Bearer abcdefg",
			statusCode:      http.StatusUnauthorized,
			dynDnsErrorCode: BadAuth,
		},
		{
			name:            "Bad Basic authorization header (no base64)",
			authHeader:      "Basic ...",
			statusCode:      http.StatusUnauthorized,
			dynDnsErrorCode: BadAuth,
		},
		{
			name:            "Bad Basic authorization header (no colon)",
			authHeader:      "Basic " + base64.StdEncoding.EncodeToString([]byte(zone+" "+token)),
			statusCode:      http.StatusUnauthorized,
			dynDnsErrorCode: BadAuth,
		},
		{
			name:            "Zone not matching",
			authHeader:      authHeader,
			zoneEnv:         "example.org",
			statusCode:      http.StatusBadRequest,
			dynDnsErrorCode: BadAgent,
		},
		{
			name:            "Token not matching",
			authHeader:      "Basic " + base64.StdEncoding.EncodeToString([]byte(zone+":wrongtoken")),
			tokenEnv:        token,
			statusCode:      http.StatusUnauthorized,
			dynDnsErrorCode: BadAuth,
		},
		{
			name:            "No such host",
			authHeader:      "Basic " + base64.StdEncoding.EncodeToString([]byte("example.org:"+token)),
			statusCode:      http.StatusBadRequest,
			dynDnsErrorCode: NoHost,
		},
		{
			name:            "Not FQDN (empty hostname)",
			authHeader:      authHeader,
			urlParamString:  "?hostname=",
			statusCode:      http.StatusBadRequest,
			dynDnsErrorCode: NotFQDN,
		},
		{
			name:            "Not FQDN (no domain part)",
			authHeader:      authHeader,
			urlParamString:  "?hostname=example",
			statusCode:      http.StatusBadRequest,
			dynDnsErrorCode: NotFQDN,
		},
		{
			name:            "Not FQDN (no hostname part)",
			authHeader:      authHeader,
			urlParamString:  "?hostname=" + zone,
			statusCode:      http.StatusBadRequest,
			dynDnsErrorCode: NotFQDN,
		},
		{
			name:            "Hostname not part of zone",
			authHeader:      authHeader,
			urlParamString:  "?hostname=dyndns.example.org",
			statusCode:      http.StatusBadRequest,
			dynDnsErrorCode: NoHost,
		},
		{
			name:            "No IP address",
			authHeader:      authHeader,
			urlParamString:  "?hostname=dyndns." + zone,
			statusCode:      http.StatusBadRequest,
			dynDnsErrorCode: BadAgent,
		},
		{
			name:            "Wrong IP address",
			authHeader:      authHeader,
			urlParamString:  "?hostname=dyndns." + zone + "&myip=hello",
			statusCode:      http.StatusBadRequest,
			dynDnsErrorCode: BadAgent,
		},
		{
			name:            "Create IPv4 record",
			zoneEnv:         zone,
			tokenEnv:        token,
			authHeader:      authHeader,
			urlParamString:  "?hostname=dyndns." + zone + "&myip=127.0.0.1",
			statusCode:      http.StatusOK,
			dynDnsErrorCode: "",
			body:            "good 127.0.0.1",
		},
		{
			name:            "Create IPv6 record",
			zoneEnv:         zone,
			tokenEnv:        token,
			authHeader:      authHeader,
			urlParamString:  "?hostname=dyndns." + zone + "&myip=::1",
			statusCode:      http.StatusOK,
			dynDnsErrorCode: "",
			body:            "good ::1",
		},
		{
			name:            "No change",
			zoneEnv:         zone,
			tokenEnv:        token,
			authHeader:      authHeader,
			urlParamString:  "?hostname=dyndns." + zone + "&myip=127.0.0.1,::1",
			statusCode:      http.StatusOK,
			dynDnsErrorCode: "",
			body:            "nochg 127.0.0.1, ::1",
		},
		{
			name:            "Update IPv4 record",
			zoneEnv:         zone,
			tokenEnv:        token,
			authHeader:      authHeader,
			urlParamString:  "?hostname=dyndns." + zone + "&myip=127.0.0.2,::1",
			statusCode:      http.StatusOK,
			dynDnsErrorCode: "",
			body:            "good 127.0.0.2, ::1",
		},
		{
			name:            "Update IPv6 record",
			zoneEnv:         zone,
			tokenEnv:        token,
			authHeader:      authHeader,
			urlParamString:  "?hostname=dyndns." + zone + "&myip=127.0.0.2,::2",
			statusCode:      http.StatusOK,
			dynDnsErrorCode: "",
			body:            "good 127.0.0.2, ::2",
		},
	}

	teardownSuite := setupSuite(t, token, zone)
	defer teardownSuite(t)

	t.Run("DynDnsHandler", func(t *testing.T) {
		for _, _tc := range tt {
			tc := _tc
			t.Run(_tc.name, func(t *testing.T) {
				if tc.zoneEnv != "" || tc.tokenEnv != "" {
					t.Setenv("ZONE", tc.zoneEnv)
					t.Setenv("TOKEN", tc.tokenEnv)
				} else {
					t.Parallel()
				}
				request, _ := http.NewRequest(http.MethodGet, "/nic/update"+tc.urlParamString, nil)
				if tc.authHeader != "" {
					request.Header.Set("Authorization", tc.authHeader)
				}

				response := httptest.NewRecorder()
				DynDnsHandler{DynDnsRequest: DynDnsRequest}.ServeHTTP(response, request)

				assertStatus(t, response.Code, tc.statusCode)
				if tc.dynDnsErrorCode != "" {
					assertDynDnsErrorCode(t, response.Body.String(), tc.dynDnsErrorCode)
				}
				if tc.body != "" {
					assertBody(t, response.Body.String(), tc.body)
				}
			})
		}
	})
}

func assertStatus(t testing.TB, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("did not get correct status, got %d, want %d", got, want)
	}
}

func assertDynDnsErrorCode(t testing.TB, got string, want DynDnsErrorCode) {
	t.Helper()
	if DynDnsErrorCode(got) != want {
		t.Errorf("did not get correct DynDnsErrorCode, got %s, want %s", got, want)
	}
}

func assertBody(t testing.TB, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("did not get correct body, got %s, want %s", got, want)
	}
}
