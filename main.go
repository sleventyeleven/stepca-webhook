package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"
	"fmt"
	"embed"

	"github.com/smallstep/certificates/webhook"
	"github.com/smallstep/webhooks/pkg/server"
	"github.com/smallstep/webhooks/pkg/db"
	"github.com/smallstep/webhooks/pkg/scep"
)

//go:embed dashboard/templates/*
var dashboardTemplates embed.FS

//go:embed dashboard/static/*
var dashboardStatic embed.FS

var certFile = "webhook.crt"
var keyFile = "webhook.key"
var clientCAs = []string{"root_ca.crt"}
var address = ":4443"
var dbFile = "scep.db"

type data struct {
	Role string `json:"role"`
}

// For demonstration only. Do not hardcode or commit actual webhook secrets.
var webhookIDsToSecrets = map[string]server.Secret{
	"8509cf3b-c657-4f69-bf78-636be7cd91fc": server.Secret{
		Signing: "G0syl5ee8W1zFTMjhJXpYFuK0QVmZfG++ImzslyVyyciv58ftmX7NMXKJeWCA/A3shjX+xrsoGO0f1+nfu/FSw==",
	},
}

// Global database instance
var database *db.Database

func main() {
	// Initialize database
	var err error
	database, err = db.NewDatabase(dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	// Initialize SCEP challenge handler
	scepHandler := scep.NewSCEPChallengeHandler(database)

	caCertPool := x509.NewCertPool()

	for _, clientCA := range clientCAs {
		caCert, err := os.ReadFile(clientCA)
		if err != nil {
			log.Panic(err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}

	s := http.Server{
		Addr: address,
		TLSConfig: &tls.Config{
			ClientCAs:  caCertPool,
			ClientAuth: tls.RequireAndVerifyClientCert,
		},
	}

	h := &server.Handler{
		Secrets: webhookIDsToSecrets,
		LookupX509: func(key string, csr *webhook.X509CertificateRequest) (any, bool, error) {
			item, ok := db[key]
			return item, ok, nil
		},
		LookupSSH: func(key string, cr *webhook.SSHCertificateRequest) (any, bool, error) {
			item, ok := db[key]
			return item, ok, nil
		},
		AllowX509: func(cert *webhook.X509Certificate) (bool, error) {
			cn := cert.Subject.CommonName
			_, ok := db[cn]
			return ok, nil
		},
		AllowSSH: func(cert *webhook.SSHCertificate) (bool, error) {
			return true, nil
		},
	}

	// Setup SCEP handlers
	scep.SetupSCEPHandlers(&s, database)

	// Setup dashboard if embedded
	http.Handle("/dashboard/", http.StripPrefix("/dashboard/", http.FileServer(http.FS(dashboardTemplates))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(dashboardStatic))))

	// Handle all other requests
	http.HandleFunc("/", h.EnrichX509)
	http.HandleFunc("/ssh/", h.EnrichSSH)
	http.HandleFunc("/auth/", h.Authorize)
	http.HandleFunc("/auth-ssh/", h.AuthorizeSSH)

	fmt.Printf("Listening on %s\n", s.Addr)
	fmt.Printf("Webhook endpoints:\n")
	fmt.Printf("  - X509: https://localhost:9443/{{.Token.sub}}\n")
	fmt.Printf("  - SSH:  https://localhost:9443/ssh/{{.Token.sub}}\n")
	fmt.Printf("  - Auth: https://localhost:9443/auth/{{.Token.sub}}\n")
	fmt.Printf("  - Auth-SSH: https://localhost:9443/auth-ssh/{{.Token.sub}}\n")
	fmt.Printf("  - SCEP Challenge: https://localhost:9443/scep/challenge/{serial}\n")
	fmt.Printf("  - Register Device: https://localhost:9443/scep/register\n")
	fmt.Printf("  - Get Challenge: https://localhost:9443/scep/challenge/get/{serial}\n")
	fmt.Printf("  - List Devices: https://localhost:9443/scep/devices\n")
	fmt.Printf("  - Delete Device: https://localhost:9443/scep/device/{serial}\n")
	fmt.Printf("  - Rotate Passphrase: https://localhost:9443/scep/device/rotate/{serial}\n")
	fmt.Printf("  - Dashboard: https://localhost:9443/dashboard/\n")

	err = s.ListenAndServeTLS(certFile, keyFile)
	log.Fatal(err)
}