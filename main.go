package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	fmt.Println("Route53 Dynamic DNS client Version", version)
	fmt.Println("Git Repo:", gitrepo)

	domain := flag.String("domain", "", "Domain name e.g. example.com")
	host := flag.String("host", "", "Subdomain or hostname e.g. www. Use '@' or leave empty for the apex record.")
	zoneID := flag.String("zone-id", "", "Route53 hosted zone ID. If omitted it is looked up by domain.")
	ttl := flag.Int64("ttl", defaultTTL, "TTL in seconds for the managed record.")
	pollInterval := flag.Duration("poll-interval", defaultPollInterval, "How often to check the public IP and reconcile the record, e.g. 15m, 5m, 30s.")

	flag.Parse()

	if *domain == "" {
		fmt.Println("ERROR domain is mandatory")
		fmt.Printf("\nUsage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *pollInterval <= 0 {
		fmt.Println("ERROR poll-interval must be greater than zero")
		os.Exit(1)
	}

	name := fqdn(*host, *domain)
	ctx := context.Background()

	client, err := newRoute53Client(ctx)
	if err != nil {
		DDNSLogger(ErrorLog, name, "Unable to initialize AWS client. "+err.Error())
	}

	resolvedZoneID := *zoneID
	if resolvedZoneID == "" {
		resolvedZoneID, err = resolveZoneID(ctx, client, *domain)
		if err != nil {
			DDNSLogger(ErrorLog, name, "Unable to resolve hosted zone ID. "+err.Error())
		}
		DDNSLogger(InformationLog, name, "Resolved hosted zone ID "+resolvedZoneID)
	}

	// Perform an initial reconcile immediately so the record is correct on startup.
	reconcile(ctx, client, resolvedZoneID, name, *ttl)

	updateRecord(ctx, client, resolvedZoneID, name, *ttl, *pollInterval)
}
