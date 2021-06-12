package controllers

import (
	"encoding/json"
	"github.com/bobesa/go-domain-util/domainutil"
	"github.com/go-errors/errors"
	"github.com/stellar/go/support/log"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ResolverController struct {
	http         *http.Client
	lock         *sync.RWMutex
	resolveCache map[string]string
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

type EthResolutionResponse struct {
	Resolved string `json:"resolved"`
}

type resolutionRequest struct {
	Hostname string `json:"hostname"`
}

func NewResolverController() *ResolverController {
	resolver := &ResolverController{
		http: &http.Client{
			Timeout: time.Second * 10,
		},
		lock:         &sync.RWMutex{},
		resolveCache: map[string]string{},
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

	r.lock.Lock()
	defer r.lock.Unlock()

	var resolvedName = ""

	if strings.HasSuffix(domain, ".eth") {
		// Use ENS resolving
		ethName, err := r.resolveByEthLink(domain)
		if err != nil {
			log.Errorf("Error resolving domain (ens)(%s): %s",domain, err.Error())
			Respond(w, MessageWithStatus(http.StatusInternalServerError, "Error during resolution process"))
			return
		}

		resolvedName = ethName
	} else {

		topLevelDomain := domainutil.Domain(domain)

		// Use DNS resolving
		dnsNames, err := net.LookupTXT(topLevelDomain)

		if err != nil {
			log.Errorf("Error resolving domain (dns)(%s): %s",domain, err.Error())
			Respond(w, MessageWithStatus(http.StatusInternalServerError, "Error during DNS resolution process: "+err.Error()))
			return
		}

		if len(dnsNames) == 0 {
			log.Errorf("Error resolving domain (dns)(%s): No TXT entry", domain)
			Respond(w, MessageWithStatus(http.StatusInternalServerError, "No TXT entry was found for provided domain: "+domain))
			return
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

	r.resolveCache[domainKey] = resolvedName
	log.Infof("Associated %s => %s ", domainKey, resolvedName)
	w.WriteHeader(http.StatusOK)
}

func (r *ResolverController) DoResolve(w http.ResponseWriter, req *http.Request) {

	var d resolutionRequest

	domain := req.Header.Get("domain")

	if domain == "" {
		err := json.NewDecoder(req.Body).Decode(&d)

		if err != nil {			
			log.Errorf("Couldn't parse data in headers or body (%s): %s",domain, err.Error())
			Respond(w, MessageWithStatus(http.StatusBadRequest, "Error parsing domain"))
			return
		}

		domain = d.Hostname

		if domain == "" {
			Respond(w, MessageWithStatus(http.StatusBadRequest, "Domain is empty"))
		}
	}

	r.lock.RLock()
	defer r.lock.RUnlock()

	if resolvedDomain, ok := r.resolveCache[domain]; ok {
		response := EthResolutionResponse{
			Resolved: resolvedDomain,
		}

		bytes, err := json.Marshal(response)
		if err != nil {
			log.Errorf("Error serializing domain resolution (%s => %s): %s",domain, resolvedDomain, err.Error())
			Respond(w, MessageWithStatus(http.StatusInternalServerError, "Error serializing resolved domain"))
			return
		}

		w.Write(bytes)
		w.WriteHeader(http.StatusOK)
		return
	}

	Respond(w, MessageWithStatus(http.StatusNoContent, "Domain resolution not found"))

}
