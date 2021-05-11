package main

import (
	"math/rand"
	"os"
	"time"
	"testing"

	"github.com/jetstack/cert-manager/test/acme/dns"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
	fqdn string
)

func TestRunsSuite(t *testing.T) {
	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.
	//

	fqdn = GetRandomString(5) + "." + zone
	timeLimit, _ := time.ParseDuration("5m");
	pollInterval, _ := time.ParseDuration("15s");

	fixture := dns.NewFixture(&zoneEuDNSProviderSolver{},
		dns.SetResolvedZone(zone),
		dns.SetResolvedFQDN(fqdn),
		dns.SetAllowAmbientCredentials(false),
		dns.SetManifestPath("testdata/zone-eu"),
		dns.SetBinariesPath("_test/kubebuilder/bin"),
		dns.SetPropagationLimit(timeLimit),
		dns.SetPollInterval(pollInterval),
	)

	fixture.RunConformance(t)
}

func GetRandomString(n int) string {
	rand.Seed(time.Now().UnixNano())

	letters := []rune("abcdefghijklmnopqrstuvwxyz")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}