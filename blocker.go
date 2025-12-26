package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Blocker struct {
	exactDomains   map[string]bool
	wildcardSuffix []string
	wildcardPrefix []string
	wildcardMiddle []wildcardPattern
	mu             sync.RWMutex
	logger         *Logger
}

type wildcardPattern struct {
	prefix string
	suffix string
}

func NewBlocker(filename string, logger *Logger) (*Blocker, error) {
	b := &Blocker{
		exactDomains:   make(map[string]bool),
		wildcardSuffix: make([]string, 0),
		wildcardPrefix: make([]string, 0),
		wildcardMiddle: make([]wildcardPattern, 0),
		logger:         logger,
	}

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("Blacklist-Datei nicht gefunden, lade bekannte Ad-Domains...")
			if err := b.createDefaultBlacklist(filename); err != nil {
				logger.Error("Fehler beim Erstellen der Standard-Blacklist: %v", err)
				return b, nil
			}
			file, err = os.Open(filename)
			if err != nil {
				return b, nil
			}
		} else {
			return nil, err
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.ToLower(line)

		if strings.Contains(line, "*") {
			b.addWildcard(line)
		} else {
			b.exactDomains[line] = true
		}
	}

	return b, scanner.Err()
}

func (b *Blocker) addWildcard(pattern string) {
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*.")
		b.wildcardSuffix = append(b.wildcardSuffix, suffix)
	} else if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		b.wildcardPrefix = append(b.wildcardPrefix, prefix)
	} else if strings.Contains(pattern, "*") {
		parts := strings.SplitN(pattern, "*", 2)
		b.wildcardMiddle = append(b.wildcardMiddle, wildcardPattern{
			prefix: parts[0],
			suffix: parts[1],
		})
	}
}

func (b *Blocker) IsBlocked(domain string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	domain = strings.ToLower(domain)
	domain = strings.TrimSuffix(domain, ".")

	if b.exactDomains[domain] {
		return true
	}

	for _, suffix := range b.wildcardSuffix {
		if strings.HasSuffix(domain, "."+suffix) || domain == suffix {
			return true
		}
	}

	for _, prefix := range b.wildcardPrefix {
		if strings.HasPrefix(domain, prefix+".") || domain == prefix {
			return true
		}
	}

	for _, pattern := range b.wildcardMiddle {
		if strings.HasPrefix(domain, pattern.prefix) && strings.HasSuffix(domain, pattern.suffix) {
			return true
		}
	}

	return false
}

func (b *Blocker) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.exactDomains) + len(b.wildcardSuffix) + len(b.wildcardPrefix) + len(b.wildcardMiddle)
}

func (b *Blocker) createDefaultBlacklist(filename string) error {
	urls := []string{
		"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
		"https://raw.githubusercontent.com/anudeepND/blacklist/master/adservers.txt",
	}

	domains := make(map[string]bool)

	for _, url := range urls {
		b.logger.Info("Lade Blacklist von: %s", url)
		if err := b.fetchAndParse(url, domains); err != nil {
			b.logger.Error("Fehler beim Laden von %s: %v", url, err)
			continue
		}
	}

	if len(domains) == 0 {
		b.logger.Info("Keine Domains geladen, erstelle Basis-Blacklist")
		domains = b.getBasicAdDomains()
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	file.WriteString("# Auto-generated DNS blacklist\n")
	file.WriteString("# Sources: StevenBlack/hosts, anudeepND/blacklist\n")
	file.WriteString("# Date of Creation: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")

	for domain := range domains {
		file.WriteString(domain + "\n")
	}

	b.logger.Info("Blacklist created with %d domains", len(domains))
	return nil
}

func (b *Blocker) fetchAndParse(url string, domains map[string]bool) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			domain := fields[1]
			if strings.Contains(domain, ".") && !strings.Contains(domain, "localhost") {
				domain = strings.ToLower(domain)
				domains[domain] = true
			}
		} else if len(fields) == 1 {
			domain := fields[0]
			if strings.Contains(domain, ".") {
				domain = strings.ToLower(domain)
				domains[domain] = true
			}
		}
	}

	return scanner.Err()
}

func (b *Blocker) getBasicAdDomains() map[string]bool {
	return map[string]bool{
		"*.doubleclick.net":       true,
		"*.googlesyndication.com": true,
		"*.googleadservices.com":  true,
		"*.google-analytics.com":  true,
		"*.googletagmanager.com":  true,
		"*.facebook.net":          true,
		"*.scorecardresearch.com": true,
		"*.advertising.com":       true,
		"ads.youtube.com":         true,
		"pixel.facebook.com":      true,
		"*.adnxs.com":             true,
		"*.adsafeprotected.com":   true,
		"*.moatads.com":           true,
		"*.adservice.google.com":  true,
		"*.amazon-adsystem.com":   true,
		"*.criteo.com":            true,
		"*.outbrain.com":          true,
		"*.taboola.com":           true,
		"*.2mdn.net":              true,
		"*.adsrvr.org":            true,
	}
}
