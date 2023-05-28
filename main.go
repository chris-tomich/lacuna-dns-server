package main

import (
	"log"
	"net"
	"os"

	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
)

// DNSRecord represents a DNS record.
type DNSRecord struct {
	Hostname string `yaml:"hostname"`
	IP       string `yaml:"ip"`
}

// DNSRecords represents a collection of DNS records.
type DNSRecords struct {
	Records []DNSRecord `yaml:"records"`
}

// LoadRecords loads DNS records from a YAML file.
func LoadRecords(filename string) (*DNSRecords, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	records := &DNSRecords{}
	err = decoder.Decode(records)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// SaveRecords saves DNS records to a YAML file.
func SaveRecords(filename string, records *DNSRecords) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	err = encoder.Encode(records)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	// Path to the YAML file
	filename := "dns_records.yaml"

	// Load the DNS records from the YAML file
	records, err := LoadRecords(filename)
	if err != nil {
		log.Fatalf("Failed to load DNS records: %v", err)
	}

	// Start the DNS server
	server := &dnsServer{
		records: records,
	}
	server.Run()
}

type dnsServer struct {
	records *DNSRecords
}

func (s *dnsServer) handleRequest(conn *net.UDPConn, addr *net.UDPAddr, buf []byte) {
	// Create a new DNS message
	request := new(dns.Msg)

	log.Printf("Received new request: %v", request)

	// Parse the DNS query
	err := request.Unpack(buf)
	if err != nil {
		log.Printf("Failed to parse DNS query: %v", err)
		return
	}

	// Check if the message contains any question
	if len(request.Question) == 0 {
		log.Printf("Received DNS message with no question")
		return
	}

	// Get the first question from the message
	question := request.Question[0]

	log.Printf("Searching for recrod: %v", question)

	// Search for the corresponding DNS record
	var record DNSRecord
	for _, r := range s.records.Records {
		log.Printf("Comparing record: %v", r)

		if r.Hostname == question.Name {
			record = r
			break
		}
	}

	// Create a new DNS message for the response
	response := new(dns.Msg)
	response.SetReply(request)

	if record.Hostname != "" {
		// If a record was found, add it as an answer
		ip := net.ParseIP(record.IP)
		if ip == nil {
			log.Printf("Invalid IP address for hostname %s", record.Hostname)
			return
		}

		answer := new(dns.A)
		answer.Hdr = dns.RR_Header{
			Name:   question.Name,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    300, // Time-to-live in seconds
		}
		answer.A = ip

		response.Answer = append(response.Answer, answer)
	} else {
		// If no record was found, construct a not found response
		response.SetRcode(request, dns.RcodeNameError)
	}

	// Encode the DNS response
	outBuf, err := response.Pack()
	if err != nil {
		log.Printf("Failed to encode DNS response: %v", err)
		return
	}

	// Send the DNS response back to the client
	_, err = conn.WriteToUDP(outBuf, addr)
	if err != nil {
		log.Printf("Failed to send DNS response: %v", err)
		return
	}
}

func (s *dnsServer) Run() {
	// Set up the UDP listener
	addr := net.UDPAddr{
		Port: 53,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Failed to set up UDP listener: %v", err)
	}
	defer conn.Close()

	log.Println("DNS server is running")

	// Start listening for DNS queries
	buf := make([]byte, 512)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error while reading from UDP: %v", err)
			continue
		}

		go s.handleRequest(conn, addr, buf[:n])
	}
}
