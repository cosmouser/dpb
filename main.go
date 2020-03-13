package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-ldap/ldap"
	log "github.com/sirupsen/logrus"
)

const (
	_defaultLdapAddr = "ldaps://ldap-server.example.edu"
	_defaultBaseDN   = "ou=People,dc=example,dc=edu"
)

var (
	_ldapUri        = flag.String("ldapURI", _defaultLdapAddr, "Full uri path to ldap server for lookups.")
	_baseDN         = flag.String("baseDN", _defaultBaseDN, "Base DN for search requests.")
	_serverPort     = flag.String("port", "9090", "Port number to listen on")
	_apiKeyFilePath = flag.String("apiKeyFile", "./apikeys.txt", "Path to api key file.")
	_loadedKeys     []string
)

// ProxySearchRequest contains the filter to apply and the attribute names to return
type ProxySearchRequest struct {
	Filter         string   `json:"filter"`
	AttributeNames []string `json:"attributeNames"`
}

func main() {
	flag.Parse()
	if err := loadKeys(*_apiKeyFilePath); err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(apiKeyChecker)

	r.Post("/search", dirlookup)
	http.ListenAndServe(":"+*_serverPort, r)
}
func loadKeys(path string) error {
	fh, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer fh.Close()
	slurp, err := ioutil.ReadAll(fh)
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(slurp), "\n")
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			_loadedKeys = append(_loadedKeys, line)
		}
	}
	return nil
}
func dirlookup(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()
	var psr ProxySearchRequest
	if r.Method != "POST" {
		http.Error(w, http.StatusText(400), 400)
		return
	}
	if err := dec.Decode(&psr); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	l, err := ldap.DialURL(*_ldapUri)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}
	defer l.Close()
	searchRequest := ldap.NewSearchRequest(
		*_baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		psr.Filter,
		psr.AttributeNames,
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}
	result := make([]map[string][]string, 0)
	for _, entry := range sr.Entries {
		profile := make(map[string][]string)
		for _, v := range psr.AttributeNames {
			profile[v] = entry.GetAttributeValues(v)
		}
		result = append(result, profile)
	}
	enc := json.NewEncoder(w)
	enc.Encode(result)
}

func apiKeyChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var auth bool
		reqKey := r.Header.Get("x-api-key")
		for _, v := range _loadedKeys {
			if reqKey == v {
				auth = true
			}
		}
		if reqKey == "" || !auth {
			http.Error(w, http.StatusText(401), 401)
			return
		}
		next.ServeHTTP(w, r)
	})
}
