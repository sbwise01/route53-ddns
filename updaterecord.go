package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// fqdn builds a fully qualified domain name from host and domain. A host of "@"
// or an empty string targets the apex/root record for the domain.
func fqdn(host, domain string) string {
	if host == "" || host == "@" {
		return domain
	}
	return host + "." + domain
}

// newRoute53Client builds a Route53 client using the default AWS credential
// chain. In Kubernetes this resolves credentials from the IRSA-provided web
// identity token (AWS_ROLE_ARN / AWS_WEB_IDENTITY_TOKEN_FILE), so no static
// credentials are needed.
func newRoute53Client(ctx context.Context) (*route53.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return route53.NewFromConfig(cfg), nil
}

// resolveZoneID looks up the hosted zone ID for a domain when one is not
// explicitly provided. It returns the most specific zone whose name matches the
// domain (ignoring private/public distinction; the first public match wins).
func resolveZoneID(ctx context.Context, client *route53.Client, domain string) (string, error) {
	name := domain
	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	out, err := client.ListHostedZonesByName(ctx, &route53.ListHostedZonesByNameInput{
		DNSName: aws.String(name),
	})
	if err != nil {
		return "", err
	}

	for _, zone := range out.HostedZones {
		if aws.ToString(zone.Name) == name {
			// Hosted zone IDs are returned as "/hostedzone/ZXXXXXX"; strip the prefix.
			id := aws.ToString(zone.Id)
			return strings.TrimPrefix(id, "/hostedzone/"), nil
		}
	}

	return "", &CustomError{ErrorCode: -1, Err: errors.New("no hosted zone found for domain " + domain)}
}

// getCurrentRecord returns the current value of the A record for name within the
// given hosted zone. An empty string (with nil error) means the record does not
// yet exist.
func getCurrentRecord(ctx context.Context, client *route53.Client, zoneID, name string) (string, error) {
	recordName := name
	if !strings.HasSuffix(recordName, ".") {
		recordName += "."
	}

	out, err := client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(zoneID),
		StartRecordName: aws.String(recordName),
		StartRecordType: types.RRTypeA,
		MaxItems:        aws.Int32(1),
	})
	if err != nil {
		return "", err
	}

	for _, rrset := range out.ResourceRecordSets {
		if aws.ToString(rrset.Name) == recordName && rrset.Type == types.RRTypeA {
			if len(rrset.ResourceRecords) > 0 {
				return aws.ToString(rrset.ResourceRecords[0].Value), nil
			}
		}
	}

	return "", nil
}

// setDNSRecord UPSERTs the A record for name to point at pubIp.
func setDNSRecord(ctx context.Context, client *route53.Client, zoneID, name string, ttl int64, pubIp string) error {
	recordName := name
	if !strings.HasSuffix(recordName, ".") {
		recordName += "."
	}

	_, err := client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		ChangeBatch: &types.ChangeBatch{
			Comment: aws.String("Updated by route53-ddns"),
			Changes: []types.Change{
				{
					Action: types.ChangeActionUpsert,
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: aws.String(recordName),
						Type: types.RRTypeA,
						TTL:  aws.Int64(ttl),
						ResourceRecords: []types.ResourceRecord{
							{Value: aws.String(pubIp)},
						},
					},
				},
			},
		},
	})
	return err
}

// reconcile compares the public IP against the current Route53 record and
// updates it when they differ. It is safe to call repeatedly.
func reconcile(ctx context.Context, client *route53.Client, zoneID, name string, ttl int64) {
	pubIp, err := getPubIP()
	if err != nil {
		DDNSLogger(WarningLog, name, "Unable to determine public IP. "+err.Error())
		return
	}

	if parsedIp := net.ParseIP(pubIp); parsedIp == nil {
		DDNSLogger(WarningLog, name, "Invalid pubIp - "+pubIp+" (This could be due to non-existent public IP or an internet issue)")
		return
	}

	currentIp, err := getCurrentRecord(ctx, client, zoneID, name)
	if err != nil {
		DDNSLogger(WarningLog, name, "Unable to read current Route53 record. "+err.Error())
		return
	}

	if currentIp == pubIp {
		DDNSLogger(InformationLog, name, "DNS record is already current ("+pubIp+"). No update needed.")
		return
	}

	if err := setDNSRecord(ctx, client, zoneID, name, ttl, pubIp); err != nil {
		DDNSLogger(WarningLog, name, "Failed to update record. "+err.Error())
		return
	}

	if currentIp == "" {
		DDNSLogger(InformationLog, name, "Record created ("+pubIp+")")
	} else {
		DDNSLogger(InformationLog, name, "Record updated (ip: "+currentIp+"->"+pubIp+")")
	}
}

// updateRecord runs the reconcile loop until an interrupt signal is received.
func updateRecord(ctx context.Context, client *route53.Client, zoneID, name string, ttl int64, pollInterval time.Duration) {
	DDNSLogger(InformationLog, name, "Started daemon process (poll interval "+pollInterval.String()+")")

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for {
		select {
		case <-c:
			DDNSLogger(InformationLog, name, "Interrupt signal received. Exiting")
			return
		case <-ticker.C:
			reconcile(ctx, client, zoneID, name, ttl)
		}
	}
}

func getPubIP() (string, error) {
	type GetIPBody struct {
		IP string `json:"ip"`
	}

	var ipbody GetIPBody

	apiclient := &http.Client{Timeout: httpTimeout}

	response, err := apiclient.Get("https://api.ipify.org?format=json")
	if err != nil {
		response, err = apiclient.Get("https://ipinfo.io/json")
		if err != nil {
			return "", err
		}
	}

	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", &CustomError{ErrorCode: response.StatusCode, Err: errors.New("IP could not be fetched. " + err.Error())}
	}

	err = json.Unmarshal(bodyBytes, &ipbody)
	if err != nil {
		return "", &CustomError{ErrorCode: response.StatusCode, Err: errors.New("IP could not be fetched. " + err.Error())}
	}

	if ipbody.IP == "" {
		return "", &CustomError{ErrorCode: response.StatusCode, Err: errors.New("IP could not be fetched. Empty IP value detected")}
	}

	return ipbody.IP, nil
}
