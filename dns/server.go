package dns

import (
	"fmt"

	"github.com/main-kube/util/safe"
	"github.com/miekg/dns"
	"go.uber.org/zap"
)

var (
	log     *zap.SugaredLogger
	records = safe.NewMap[string, string](nil) // name -> ip
)

func Start() {
	log = zap.S()

	// attach request handler func
	dns.HandleFunc("svc.", handleDnsRequest)   // service
	dns.HandleFunc("pod.", handleDnsRequest)   // pod
	dns.HandleFunc("local.", handleDnsRequest) // cluster.local

	// start server
	server := &dns.Server{Addr: ":53", Net: "udp"}
	log.Info("Starting at :53")
	err := server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		log.Errorf("Failed to start server: %s\n ", err.Error())
	}
}

func Register(name, ip string) {
	log.Infof("Registering dns A record for %s\n", name)
	records.Set(name, ip)
}

func Unregister(name string) {
	records.Delete(name)
}

func parseQuery(m *dns.Msg) {
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			log.Infof("Query for %s\n", q.Name)
			ip := records.Get(q.Name)
			if ip != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m)
	}

	w.WriteMsg(m)
}
