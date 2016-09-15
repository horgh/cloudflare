// This program makes a CloudFlare API request to update an A record IP.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"summercat.com/cloudflare"
	"summercat.com/icanhazip"
)

// Args are command line arguments.
type Args struct {
	Email           string
	Domain          string
	Hostname        string
	KeyFile         string
	IP              net.IP
	OnlyIfDifferent bool
	Verbose         bool
}

func main() {
	log.SetFlags(0)

	args, err := getArgs()
	if err != nil {
		flag.PrintDefaults()
		os.Exit(1)
	}

	key, err := getKeyFromFile(args.KeyFile)
	if err != nil {
		log.Fatalf("Unable to read key: %s", err)
	}

	// Decide which IP to set. Use the CLI arg value if given.
	ip := args.IP
	if ip == nil {
		myIP, err := icanhazip.Lookup()
		if err != nil {
			log.Fatalf("Unable to look up IP from icanhazip.com: %s", err)
		}
		if args.Verbose {
			log.Printf("Found current IP is %s", myIP)
		}
		ip = myIP
	}

	// We may want to make an update.

	// If we want to make it without checking if there is a difference, then do so
	if !args.OnlyIfDifferent {
		err := updateIP(key, args.Email, args.Domain, args.Hostname, args.Verbose,
			ip)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// We only want to make an update if there is a difference.
	// To know the current IP, look up its A record.
	currentIPs, err := net.LookupIP(args.Hostname)
	if err != nil {
		log.Fatalf("Unable to look up A records: %s", err)
	}

	if len(currentIPs) == 0 {
		log.Fatalf("Unable to determine current record IP via DNS. No IPs found.")
	}

	if len(currentIPs) > 1 {
		log.Fatalf("There are %d A records. Unable to update.", len(currentIPs))
	}

	currentIP := currentIPs[0]

	if currentIP.Equal(ip) {
		if args.Verbose {
			log.Printf("DNS record's IP matches IP provided/found (%s). Not making an update.",
				ip)
		}
		return
	}

	err = updateIP(key, args.Email, args.Domain, args.Hostname, args.Verbose,
		ip)
	if err != nil {
		log.Fatal(err)
	}
}

func getArgs() (Args, error) {
	email := flag.String("email", "", "Email address on your CloudFlare account.")
	domain := flag.String("domain", "", "Domain involved in the update.")
	hostname := flag.String("hostname", "", "Hostname to update.")
	keyFile := flag.String("key-file", "", "Path to file containing API key. The file should contain nothing but your key.")
	ipString := flag.String("ip", "", "IP to set. If you don't provide this, then we query icanhazip.com for your current IP.")
	onlyIfDifferent := flag.Bool("only-if-different", false, "If true, we check the current IP of the host via DNS, and only contact the CloudFlare API if it does not match the IP you provided (or we found as current).")
	verbose := flag.Bool("verbose", false, "Toggle verbose output.")

	flag.Parse()

	if len(*email) == 0 {
		return Args{}, fmt.Errorf("You must provide an email.")
	}

	if len(*domain) == 0 {
		return Args{}, fmt.Errorf("You must provide a domain.")
	}

	if len(*hostname) == 0 {
		return Args{}, fmt.Errorf("You must provide a hostname.")
	}

	if len(*keyFile) == 0 {
		return Args{}, fmt.Errorf("You must provide an API key file.")
	}

	var ip net.IP
	if len(*ipString) > 0 {
		ip = net.ParseIP(*ipString)
		if ip == nil {
			return Args{}, fmt.Errorf("Invalid IP address.")
		}
	}

	return Args{
		Email:           *email,
		Domain:          *domain,
		Hostname:        *hostname,
		KeyFile:         *keyFile,
		IP:              ip,
		OnlyIfDifferent: *onlyIfDifferent,
		Verbose:         *verbose,
	}, nil
}

func getKeyFromFile(keyFile string) (string, error) {
	fh, err := os.Open(keyFile)
	if err != nil {
		return "", err
	}
	defer fh.Close()

	content, err := ioutil.ReadAll(fh)
	if err != nil {
		return "", fmt.Errorf("Problem reading from file: %s", err)
	}

	key := strings.TrimSpace(string(content))

	if len(key) == 0 {
		return "", fmt.Errorf("No key found in file.")
	}

	return key, nil
}

func updateIP(key, email, domain, hostname string, verbose bool,
	ip net.IP) error {
	client := cloudflare.NewClient(key, email)

	zones, err := client.ListZones(domain, "", -1, -1, "", "", "")
	if err != nil {
		return fmt.Errorf("Unable to list zones: %s", err)
	}

	// This program is specifically for updating A records.
	recordType := "A"

	// There may be multiple A records for a host.
	matchingRecords := []cloudflare.DNSRecord{}

	for _, zone := range zones {
		if verbose {
			log.Printf("Zone: %+v", zone)
		}

		records, err := client.ListDNSRecords(zone.ID, recordType, hostname,
			"", -1, -1, "", "", "")
		if err != nil {
			return fmt.Errorf("Unable to list DNS records: %s", err)
		}

		for _, record := range records {
			if verbose {
				log.Printf("Record: %+v", record)
			}
			if record.Name == hostname && record.Type == recordType {
				matchingRecords = append(matchingRecords, record)
			}
		}
	}

	if len(matchingRecords) == 0 {
		return fmt.Errorf("Record not found. No update performed.")
	}

	if len(matchingRecords) > 1 {
		return fmt.Errorf("Multiple matching records found. Unable to perform update.")
	}

	record := matchingRecords[0]

	if record.Content == ip.String() {
		log.Printf("Record already has IP [%s]. No update performed.", ip.String())
		return nil
	}

	record.Content = ip.String()

	if verbose {
		log.Printf("Updating record to: %+v", record)
	}

	err = client.UpdateDNSRecord(record)
	if err != nil {
		return fmt.Errorf("Unable to update DNS record: %s", err)
	}

	log.Printf("Updated A record of [%s] to IP [%s]", hostname, ip.String())
	return nil
}
