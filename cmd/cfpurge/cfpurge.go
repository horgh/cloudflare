// cfpurge provides a way to purge all files associated with a Cloudflare
// domain.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/horgh/cloudflare"
)

// Args are command line arguments.
type Args struct {
	Email   string
	Domain  string
	KeyFile string
	Verbose bool
}

func main() {
	log.SetFlags(0)

	args, err := getArgs()
	if err != nil {
		flag.PrintDefaults()
		os.Exit(1)
	}

	key, err := cloudflare.ReadKeyFromFile(args.KeyFile)
	if err != nil {
		log.Fatalf("Unable to read key: %s", err)
	}

	client := cloudflare.NewClient(key, args.Email)

	if args.Verbose {
		client.Debug = true
	}

	// Find zone for the domain.
	zones, err := client.ListZones(args.Domain, "", -1, -1, "", "", "")
	if err != nil {
		log.Fatalf("Unable to list zones: %s", err)
	}

	if err != nil {
		log.Fatalf("Failed to list zones: %s", err)
	}

	if len(zones) != 1 {
		log.Fatalf("Zone not found for domain: %s", err)
	}

	err = client.PurgeAllFiles(zones[0].ID)
	if err != nil {
		log.Fatalf("Purge failed: %s", err)
	}

	if args.Verbose {
		log.Printf("Purge complete.")
	}
}

func getArgs() (Args, error) {
	email := flag.String("email", "", "Email address on your Cloudflare account.")
	domain := flag.String("domain", "", "Domain involved in the update.")
	keyFile := flag.String("key-file", "", "Path to file containing API key. The file should contain nothing but your key.")
	verbose := flag.Bool("verbose", false, "Toggle verbose output.")

	flag.Parse()

	if len(*email) == 0 {
		return Args{}, fmt.Errorf("you must provide an email")
	}

	if len(*domain) == 0 {
		return Args{}, fmt.Errorf("you must provide a domain")
	}

	if len(*keyFile) == 0 {
		return Args{}, fmt.Errorf("you must provide an API key file")
	}

	return Args{
		Email:   *email,
		Domain:  *domain,
		KeyFile: *keyFile,
		Verbose: *verbose,
	}, nil
}
