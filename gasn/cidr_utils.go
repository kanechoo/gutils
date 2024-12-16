package gasn

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/kanechoo/gutils/genv"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	asnUrl             = "https://www.cidr-report.org/cgi-bin/as-report?as=AS%d&v=4&view=2.0"
	TypeAnnounced Type = iota
	TypeWithdrawn Type = iota
	countryAsnUrl      = "https://bgp.he.net/country/%s"
)

var (
	CountryCodes = &[]string{
		"AF", "AX", "AL", "DZ", "AS", "AD", "AO", "AI", "AQ", "AG", "AR", "AM", "AW", "AU", "AT",
		"AZ", "BS", "BH", "BD", "BB", "BY", "BE", "BZ", "BJ", "BM", "BT", "BO", "BQ", "BA", "BW",
		"BV", "BR", "IO", "BN", "BG", "BF", "BI", "CV", "KH", "CM", "CA", "KY", "CF", "TD", "CL",
		"CN", "CX", "CC", "CO", "KM", "CG", "CD", "CK", "CR", "CI", "HR", "CU", "CW", "CY", "CZ",
		"DK", "DJ", "DM", "DO", "EC", "EG", "SV", "GQ", "ER", "EE", "SZ", "ET", "FK", "FO", "FJ",
		"FI", "FR", "GF", "PF", "TF", "GA", "GM", "GE", "DE", "GH", "GI", "GR", "GL", "GD", "GP",
		"GU", "GT", "GG", "GN", "GW", "GY", "HT", "HM", "VA", "HN", "HK", "HU", "IS", "IN", "ID",
		"IR", "IQ", "IE", "IM", "IL", "IT", "JM", "JP", "JE", "JO", "KZ", "KE", "KI", "KP", "KR",
		"KW", "KG", "LA", "LV", "LB", "LS", "LR", "LY", "LI", "LT", "LU", "MO", "MG", "MW", "MY",
		"MV", "ML", "MT", "MH", "MQ", "MR", "MU", "YT", "MX", "FM", "MD", "MC", "MN", "ME", "MS",
		"MA", "MZ", "MM", "NA", "NR", "NP", "NL", "NC", "NZ", "NI", "NE", "NG", "NU", "NF", "MK",
		"MP", "NO", "OM", "PK", "PW", "PS", "PA", "PG", "PY", "PE", "PH", "PN", "PL", "PT", "PR",
		"QA", "RE", "RO", "RU", "RW", "BL", "SH", "KN", "LC", "MF", "PM", "VC", "WS", "SM", "ST",
		"SA", "SN", "RS", "SC", "SL", "SG", "SX", "SK", "SI", "SB", "SO", "ZA", "GS", "SS", "ES",
		"LK", "SD", "SR", "SJ", "SE", "CH", "SY", "TW", "TJ", "TZ", "TH", "TL", "TG", "TK", "TO",
		"TT", "TN", "TR", "TM", "TC", "TV", "UG", "UA", "AE", "GB", "US", "UM", "UY", "UZ", "VU",
		"VE", "VN", "VG", "VI", "WF", "EH", "YE", "ZM", "ZW",
	}
)

type PrefixOptions struct {
	WithAnnounced bool
	WithWithdrawn bool
	regex         *regexp.Regexp
}
type ClientOptions struct {
	HttpClient *http.Client
	RetryTimes int
}

type Type int
type ASNTiny struct {
	As   int
	Name string
}
type ASN struct {
	As         int
	Name       string
	NetBlock   *[]NetBlock
	Upstream   *[]string
	Downstream *[]string
}
type NetBlock struct {
	Prefix string
	Type   Type
}

type Client struct {
	Options       PrefixOptions
	ClientOptions ClientOptions
}

func WithRetryTimes(retryTimes int) func(options *ClientOptions) {
	return func(options *ClientOptions) {
		options.RetryTimes = retryTimes
	}
}
func OnlyAnnounced(options *PrefixOptions) {
	options.WithAnnounced = true
}

func WithHttpClient(client *http.Client) func(options *ClientOptions) {
	return func(options *ClientOptions) {
		options.HttpClient = client
	}
}

func OnlyWithdrawn(options *PrefixOptions) {
	options.WithWithdrawn = true
}
func New(options ...func(options *ClientOptions)) *Client {
	client := &Client{
		Options: PrefixOptions{
			regex: regexp.MustCompile("\\b(?:\\d{1,3}\\.){3}\\d{1,3}\\/\\d{1,2}\\b"),
		},
		ClientOptions: ClientOptions{},
	}
	for _, o := range options {
		o(&client.ClientOptions)
	}
	if client.ClientOptions.HttpClient == nil {
		client.ClientOptions.HttpClient = defaultHttpClient()
	}
	if client.ClientOptions.RetryTimes == 0 {
		client.ClientOptions.RetryTimes = 1
	}
	return client
}
func (c *Client) appendOptions(options ...func(*PrefixOptions)) {
	if nil == options || len(options) == 0 {
		return
	}
	for _, option := range options {
		option(&c.Options)
	}
}
func (c *Client) ByCountry(country string) (*[]ASNTiny, error) {
	flag := false
	for _, cc := range *CountryCodes {
		if cc == country {
			flag = true
			break
		}
	}
	if !flag {
		return nil, errors.New("unsupported country")
	}
	resp, err := c.ClientOptions.HttpClient.Get(fmt.Sprintf(countryAsnUrl, country))
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	asnList := make([]ASNTiny, 0)
	doc.Find("#asns > tbody > tr").Each(func(i int, selection *goquery.Selection) {
		as := selection.Find("td:nth-child(1)").Text()
		name := selection.Find("td:nth-child(2)").Text()
		strings.TrimPrefix(as, "AS")
		intASN, err := strconv.Atoi(strings.TrimPrefix(as, "AS"))
		if err != nil {
			return
		}
		asn := ASNTiny{
			As:   intASN,
			Name: name,
		}
		asnList = append(asnList, asn)
	})
	return &asnList, nil
}
func (c *Client) ByAsn(asn int, options ...func(*PrefixOptions)) (*ASN, error) {
	c.appendOptions(options...)
	url := fmt.Sprintf(asnUrl, asn)
	var resp *http.Response
	var err error
	currentRetry := 0
	for {
		if currentRetry > c.ClientOptions.RetryTimes {
			break
		}
		r, e := c.ClientOptions.HttpClient.Get(url)
		err = e
		if e != nil {
			continue
		}
		resp = r
		break
	}
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("response is nil")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	text := doc.Find("body > ul:nth-of-type(3)").Text()
	prefixes, err := c.parseTextToPrefixes(&text)
	if err != nil {
		return nil, err
	}
	if c.Options.WithAnnounced && prefixes != nil {
		var withAnnounced []NetBlock
		for _, prefix := range *prefixes {
			if prefix.Type == TypeAnnounced {
				withAnnounced = append(withAnnounced, prefix)
			}
		}
		prefixes = removeDuplicates(&withAnnounced)
	}
	if c.Options.WithWithdrawn && prefixes != nil {
		var withWithdrawn []NetBlock
		for _, prefix := range *prefixes {
			if prefix.Type == TypeWithdrawn {
				withWithdrawn = append(withWithdrawn, prefix)
			}
		}
		prefixes = removeDuplicates(&withWithdrawn)
	}
	prefixes = removeDuplicates(prefixes)
	text = doc.Find("body > ul:nth-child(13) > pre").Text()
	upstream, downstream := parseToUpstreamDownstream(text)
	name := doc.Find("body > ul:nth-child(7)").Text()
	a1 := &ASN{
		As:         asn,
		Name:       name,
		NetBlock:   prefixes,
		Upstream:   upstream,
		Downstream: downstream,
	}
	return a1, nil
}

func removeDuplicates(list *[]NetBlock) *[]NetBlock {
	if list == nil || len(*list) == 0 {
		return list
	}
	keys := make(map[string]bool)
	var result []NetBlock
	for _, item := range *list {
		if _, value := keys[item.Prefix]; !value {
			keys[item.Prefix] = true
			result = append(result, item)
		}
	}
	return &result
}

func (c *Client) parseTextToPrefixes(text *string) (*[]NetBlock, error) {
	if text == nil || *text == "" {
		return nil, errors.New("html text is empty")
	}
	var result []NetBlock
	skip := false
	for _, line := range strings.Split(*text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, "Advertisements that are fragments of the original") {
			skip = true
		}
		if strings.Contains(line, "Prefix added and withdrawn by this origin") {
			skip = false
		}
		if skip {
			continue
		}
		f := c.Options.regex.FindStringSubmatch(line)
		if len(f) == 0 {
			continue
		}
		if strings.Contains(strings.ToLower(line), "withdrawn") {
			result = append(result, NetBlock{
				Prefix: f[0],
				Type:   TypeWithdrawn,
			})
		} else {
			result = append(result, NetBlock{
				Prefix: f[0],
				Type:   TypeAnnounced,
			})
		}
	}
	return &result, nil
}

func parseToUpstreamDownstream(text string) (*[]string, *[]string) {
	upstream := make([]string, 0)
	downstream := make([]string, 0)
	upstreamFlag := false
	downstreamFlag := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "Upstream Adjacent AS list" {
			upstreamFlag = true
			continue
		}
		if line == "Downstream Adjacent AS list" {
			downstreamFlag = true
			upstreamFlag = false
			continue
		}
		if upstreamFlag {
			info := strings.Split(line, " ")
			if len(info) < 2 {
				continue
			}
			as := strings.TrimPrefix(info[0], "AS")
			upstream = append(upstream, as)
			continue
		}
		if downstreamFlag {
			info := strings.Split(line, " ")
			if len(info) < 2 {
				continue
			}
			as := strings.TrimPrefix(info[0], "AS")
			downstream = append(downstream, as)
			continue
		}
	}
	return &upstream, &downstream
}

func defaultHttpClient() *http.Client {
	proxy := genv.HttpProxy()
	transport := &http.Transport{
		Proxy:                 proxy,
		TLSHandshakeTimeout:   2 * time.Second,
		IdleConnTimeout:       2 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
}
