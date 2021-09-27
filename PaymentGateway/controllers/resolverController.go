package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bobesa/go-domain-util/domainutil"
	"github.com/go-errors/errors"
	"paidpiper.com/payment-gateway/log"
)

type ResolverController struct {
	http         *http.Client
	lock         *sync.RWMutex
	resolveCache map[string]string
	resolveKey   string
}

type AnswerData struct {
	TTL  int    `json:"TTL"`
	Data string `json:"data"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

type QuestionData struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}

type EthLinkResult struct {
	AD       bool           `json:"AD"`
	CD       bool           `json:"CD"`
	RA       bool           `json:"RA"`
	RD       bool           `json:"RD"`
	TC       bool           `json:"TC"`
	Status   int            `json:"Status"`
	Answer   []AnswerData   `json:"Answer"`
	Question []QuestionData `json:"Question"`
}

// TODO REMOVE RESOLVED PROPERTY
type EthResolutionResponse struct {
	Resolved string `json:"resolved"`
	Hostname string `json:"hostname"`
}

type resolutionRequest struct {
	Hostname string `json:"hostname"`
}

func NewResolverController(resolveKey string) *ResolverController {
	resolver := &ResolverController{
		http: &http.Client{
			Timeout: time.Second * 10,
		},
		lock:         &sync.RWMutex{},
		resolveCache: map[string]string{},
		resolveKey:   resolveKey,
	}

	//resolver.resolveCache["video.torplus.eth"] = "yta5tsernfhyqg4rztgqtxpfw5mzdvkwxavfny67xlf3lw5g5jrsz4qd.onion"

	return resolver
}

func (r *ResolverController) resolveByEthLink(domain string) (string, error) {

	log.Debugf("resolveByEthLink: %s ", domain)

	req, err := http.NewRequest(http.MethodGet, "https://eth.link/dns-query", nil)

	if err != nil {
		return "", errors.Errorf("Error creating request for dns-query")
	}

	q := req.URL.Query()
	q.Add("type", "TXT")
	q.Add("name", domain)

	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/dns-json")

	res, err := r.http.Do(req)

	if err != nil {
		return "", errors.Errorf("Error executing request (dns-query): %s - %w", err.Error(), err)
	}

	defer res.Body.Close()

	log.Infof("resolveByEthLink response for %s : %s, %v", domain, res.Status, res.Header)

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "", errors.Errorf("Error reading response from stream: %s - %w", err.Error(), err)
	}

	log.Debugf("resolveByEthLink response body: %s ", string(data))

	var result EthLinkResult

	if err := json.Unmarshal(data, &result); err != nil {
		return "", errors.Errorf("Error unmarshalling data (json): %s - %w", err.Error(), err)
	}

	var resolvedData string

	for _, answer := range result.Answer {

		var token = strings.Trim(answer.Data, "\"'\n ")
		if strings.HasPrefix(token, "dnslink") {
			resolvedData = strings.TrimPrefix(token, "dnslink=")
			log.Debugf("resolveByEthLink dnslink response: %s ", answer)
		}
	}

	return resolvedData, nil
}

func (r *ResolverController) SetupResolving(w http.ResponseWriter, req *http.Request) {
	domain := req.Header.Get("domainToResolve")
	domainKey := req.Header.Get("domainKey")

	log.Infof("SetupResolving with %s, %s\n", domain, domainKey)

	if (domainKey == "") || (domain == "") {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Domain is empty"))
		return
	}

	// If there's a port, trim it
	if strings.ContainsRune(domain, ':') {
		var index = strings.LastIndex(domain, ":")
		domain = domain[:index]
	}

	if strings.ContainsRune(domainKey, ':') {
		var index = strings.LastIndex(domainKey, ":")
		domainKey = domainKey[:index]
	}

	//TODO: Refactor this to use configuration and some kind of helper class
	fragment := req.Header.Get("fragment")

	if strings.HasPrefix(domain, "sites.") {
		fragment = strings.ReplaceAll(fragment, "#", "")
		domain = strings.ReplaceAll(domain, "sites.", fragment+".")
	}
	resolvedDomain, err := r.trySetupResolving(domain, domainKey)
	if err != nil {
		log.Error("Setup resolving error:", err)

		Respond(w, MessageWithStatus(http.StatusInternalServerError, err.Error()))
		return
	}
	r.lock.RLock()
	defer r.lock.RUnlock()
	r.resolveCache[domain] = resolvedDomain
	log.Infof("Associated %s => %s ", domainKey, resolvedDomain)
	w.WriteHeader(http.StatusOK)
	// var resolvedName = ""
	// r.resolveCache[domainKey] = resolvedName // it is fix
	// if strings.HasSuffix(domain, ".eth") {
	// 	// Use ENS resolving
	// 	ethName, err := r.resolveByEthLink(domain)
	// 	if err != nil {
	// 		log.Errorf("Error resolving domain (ens)(%s): %s", domain, err.Error())
	// 		Respond(w, MessageWithStatus(http.StatusInternalServerError, "Error during resolution process"))
	// 		return
	// 	}

	// 	resolvedName = ethName
	// } else {

	// 	topLevelDomain := domainutil.Domain(domain)

	// 	// First attempt to resolve exact domain
	// 	dnsNames, err := net.LookupTXT(domain)

	// 	if err != nil {
	// 		log.Infof("Direct resolution failed (%s), attempting root domain(%s): %s", domain, topLevelDomain, err.Error())

	// 		// Use DNS resolving
	// 		dnsNames, err = net.LookupTXT(topLevelDomain)

	// 		if err != nil {
	// 			log.Errorf("Error resolving root domain (dns)(%s): %s", topLevelDomain, err.Error())
	// 			Respond(w, MessageWithStatus(http.StatusInternalServerError, "Error during DNS resolution process: "+err.Error()))
	// 			return
	// 		}
	// 	}

	// 	if len(dnsNames) == 0 {
	// 		log.Errorf("Error resolving domain (dns)(%s): No TXT entry", domain)
	// 		Respond(w, MessageWithStatus(http.StatusInternalServerError, "No TXT entry was found for provided domain: "+domain))
	// 		return
	// 	}

	// 	// Parse TXT entry into lookup
	// 	ss := strings.Split(dnsNames[0], ",")
	// 	m := make(map[string]string)
	// 	for _, pair := range ss {
	// 		z := strings.Split(pair, "=")
	// 		m[z[0]] = z[1]
	// 	}

	// 	resolvedName = m[r.resolveKey]
	// }

	// //TODO: Parse the response using multiaddr
	// if strings.HasPrefix(resolvedName, "/onion3/") {
	// 	resolvedName = strings.TrimPrefix(resolvedName, "/onion3/") + ".onion"
	// }

	// // If the address is probably an onion address without .onion extension, add it
	// if len(resolvedName) == 56 && !strings.ContainsRune(resolvedName, '.') {
	// 	resolvedName = resolvedName + ".onion"
	// }
	// r.lock.Lock()
	// defer r.lock.Unlock()
	// r.resolveCache[domainKey] = resolvedName

}
func (r *ResolverController) trySetupResolving(domain string, domainKey string) (string, error) {

	log.Infof("SetupResolving with %s, %s\n", domain, domainKey)

	if (domainKey == "") || (domain == "") {
		return "", fmt.Errorf("domain is empty")
		//Respond(w, MessageWithStatus(http.StatusBadRequest, "Domain is empty"))
		//return
	}

	// If there's a port, trim it
	if strings.ContainsRune(domain, ':') {
		var index = strings.LastIndex(domain, ":")
		domain = domain[:index]
	}

	if strings.ContainsRune(domainKey, ':') {
		var index = strings.LastIndex(domainKey, ":")
		domainKey = domainKey[:index]
	}

	var resolvedName = ""
	if strings.HasSuffix(domain, ".eth") {
		// Use ENS resolving
		ethName, err := r.resolveByEthLink(domain)
		if err != nil {
			log.Errorf("Error resolving domain (ens)(%s): %s", domain, err.Error())

			return "", fmt.Errorf("Error during resolution process")
		}

		resolvedName = ethName
	} else {

		topLevelDomain := domainutil.Domain(domain)

		// First attempt to resolve exact domain
		dnsNames, err := net.LookupTXT(domain)

		if err != nil {
			log.Infof("Direct resolution failed (%s), attempting root domain(%s): %s", domain, topLevelDomain, err.Error())

			// Use DNS resolving
			dnsNames, err = net.LookupTXT(topLevelDomain)

			if err != nil {
				log.Errorf("Error resolving root domain (dns)(%s): %s", topLevelDomain, err.Error())
				return "", fmt.Errorf("Error during DNS resolution process: " + err.Error())
			}
		}

		if len(dnsNames) == 0 {
			log.Errorf("Error resolving domain (dns)(%s): No TXT entry", domain)
			return "", fmt.Errorf("No TXT entry was found for provided domain: " + domain)
		}

		// Parse TXT entry into lookup
		ss := strings.Split(dnsNames[0], ",")
		m := make(map[string]string)
		for _, pair := range ss {
			z := strings.Split(pair, "=")
			m[z[0]] = z[1]
		}

		resolvedName = m["torplus"]
	}

	//TODO: Parse the response using multiaddr
	if strings.HasPrefix(resolvedName, "/onion3/") {
		resolvedName = strings.TrimPrefix(resolvedName, "/onion3/") + ".onion"
	}

	// If the address is probably an onion address without .onion extension, add it
	if len(resolvedName) == 56 && !strings.ContainsRune(resolvedName, '.') {
		resolvedName = resolvedName + ".onion"
	}

	log.Infof("Associated %s => %s ", domainKey, resolvedName)
	return resolvedName, nil
}

func (r *ResolverController) DoResolve(w http.ResponseWriter, req *http.Request) {

	var d resolutionRequest

	domain := req.Header.Get("domain")

	if domain == "" {
		err := json.NewDecoder(req.Body).Decode(&d)

		if err != nil {
			log.Errorf("Resolving: Couldn't parse data in headers or body (%s): %s", domain, err.Error())
			Respond(w, MessageWithStatus(http.StatusBadRequest, "Error parsing domain"))
			return
		}

		domain = d.Hostname

		if domain == "" {
			Respond(w, MessageWithStatus(http.StatusBadRequest, "Domain is empty"))
		}
	}
	log.Info("Resolving: Try resolve for: ", domain)

	resolvedDomain, ok := r.resolveCache[domain]
	if !ok {
		var err error
		resolvedDomain, err = r.trySetupResolving(domain, domain)
		if err != nil {
			log.Error("Setup resolving error:", err)
		}
		if err == nil {
			ok = true
			r.lock.RLock()
			defer r.lock.RUnlock()
			r.resolveCache[domain] = resolvedDomain
		}
	}
	if ok {
		if resolvedDomain != "" {
			response := EthResolutionResponse{
				Resolved: resolvedDomain,
				Hostname: resolvedDomain,
			}

			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Errorf("Resolving: Error serializing domain resolution (%s => %s): %s", domain, resolvedDomain, err.Error())
				Respond(w, MessageWithStatus(http.StatusInternalServerError, "Error serializing resolved domain"))
				return
			}
			log.Info("Resolving: Resolve domain success:", domain, " => ", resolvedDomain)
			w.WriteHeader(http.StatusOK)

			return
		}
	}
	log.Infof("Resolving: Domain resolution not found (%s)", domain)
	Respond(w, MessageWithStatus(http.StatusNoContent, "Domain resolution not found"))

}
