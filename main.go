package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	flags "github.com/jessevdk/go-flags"
)

var opts struct {
	Backend       string   `short:"b" long:"backend" description:"IP of the DNS resolver to use for lookups" choice:"maxmind" choice:"cymru" default:"cymru"`
	IP            []string `short:"a" description:"IP Addresses to look up"`
	HumanReadable bool     `short:"r" long:"human-readable" description:"Print human readable output"`
	Short         bool     `short:"s" description:"Print short output"`
}

//AS Autonomous System
type AS struct {
	ASNumber,
	ASName,
	QueryAddress,
	BGPPrefix,
	CC,
	Registry,
	Allocated string
}

//Perform an AS lookup for a given IP against the cymru.com whois Service
func cymruWhoisLookup(ip string, wg *sync.WaitGroup, res chan AS) {
	defer wg.Done()

	conn, err := net.Dial("tcp", "v4.whois.cymru.com:43")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(conn, "-v -f %s\n", ip)
	status, err := bufio.NewReader(conn).ReadString('\n')

	if strings.HasPrefix(status, "Error:") {
		fmt.Fprintf(os.Stderr, "cymru returned: '%s' on '%s'!\n", strings.Trim(status, "\n"), ip)
		return
	}

	if err == nil {
		response := strings.SplitN(status, "|", 7)
		r := AS{
			strings.TrimSpace(response[0]),
			strings.TrimSpace(response[6]),
			strings.TrimSpace(response[1]),
			strings.TrimSpace(response[2]),
			strings.TrimSpace(response[3]),
			strings.TrimSpace(response[4]),
			strings.TrimSpace(response[5]),
		}

		res <- r
	} else {
		panic(err)
	}
}

func output(asinfo AS) {
	if opts.HumanReadable {
		fmt.Printf("IP %s is in AS%s (prefix %s), belongs to %s and is assigned since %s to %s\n",
			asinfo.QueryAddress, asinfo.ASNumber, asinfo.BGPPrefix, asinfo.Registry, asinfo.Allocated, asinfo.ASName)
	} else {
		fmt.Printf("%+v\n", asinfo)
	}
}

func main() {
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		os.Exit(1)
	}

	var wg sync.WaitGroup
	res := make(chan AS)

	for _, e := range opts.IP {
		wg.Add(1)
		if opts.Backend == "cymru" {
			go cymruWhoisLookup(e, &wg, res)
		} else {
			fmt.Fprintf(os.Stderr, "Sorry, only implemented backend is \"cymru\" for the moment.")
			os.Exit(128)
		}
	}
	go func() {
		wg.Wait()
		close(res)
	}()

	for r := range res {
		output(r)
	}
}
