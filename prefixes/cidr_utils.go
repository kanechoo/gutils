package prefixes

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	URL                = "https://www.cidr-report.org/cgi-bin/as-report?as=AS%d&v=4&view=2.0"
	TypeAnnounced Type = iota
	TypeWithdrawn Type = iota
)

type Options struct {
	WithAnnounced bool
	WithWithdrawn bool
	HttpClient    *http.Client
	regex         *regexp.Regexp
	RetryTimes    int
}

type Type int
type NetBlock struct {
	Prefixes string
	Type     Type
}

type Client struct {
	Options Options
}

func WithRetryTimes(retryTimes int) func(options *Options) {
	return func(options *Options) {
		options.RetryTimes = retryTimes
	}
}
func OnlyAnnounced(options *Options) {
	options.WithAnnounced = true
}

func WithHttpClient(client *http.Client) func(options *Options) {
	return func(options *Options) {
		options.HttpClient = client
	}
}

func OnlyWithdrawn(options *Options) {
	options.WithWithdrawn = true
}
func NewClient() *Client {
	return &Client{
		Options: Options{
			HttpClient: getHttpC(),
			regex:      regexp.MustCompile("\\b(?:\\d{1,3}\\.){3}\\d{1,3}\\/\\d{1,2}\\b"),
			RetryTimes: 0,
		},
	}
}
func (c *Client) appendOptions(options ...func(*Options)) {
	if nil == options || len(options) == 0 {
		return
	}
	for _, option := range options {
		option(&c.Options)
	}
}
func (c *Client) GetByAsn(asn int, options ...func(*Options)) (*[]NetBlock, error) {
	c.appendOptions(options...)
	url := fmt.Sprintf(URL, asn)
	var resp *http.Response
	var err error
	currentRetry := 0
	for {
		if currentRetry > c.Options.RetryTimes {
			break
		}
		r, e := c.Options.HttpClient.Get(url)
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
	if prefixes == nil || len(*prefixes) == 0 {
		return prefixes, nil
	}
	if c.Options.WithAnnounced {
		var withAnnounced []NetBlock
		for _, prefix := range *prefixes {
			if prefix.Type == TypeAnnounced {
				withAnnounced = append(withAnnounced, prefix)
			}
		}
		return removeDuplicates(&withAnnounced), nil
	}
	if c.Options.WithWithdrawn {
		var withWithdrawn []NetBlock
		for _, prefix := range *prefixes {
			if prefix.Type == TypeWithdrawn {
				withWithdrawn = append(withWithdrawn, prefix)
			}
		}
		return removeDuplicates(&withWithdrawn), nil
	}
	return removeDuplicates(prefixes), nil
}

func removeDuplicates(list *[]NetBlock) *[]NetBlock {
	if list == nil || len(*list) == 0 {
		return list
	}
	keys := make(map[string]bool)
	var result []NetBlock
	for _, item := range *list {
		if _, value := keys[item.Prefixes]; !value {
			keys[item.Prefixes] = true
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
		if strings.Contains(line, "Prefixes added and withdrawn by this origin") {
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
				Prefixes: f[0],
				Type:     TypeWithdrawn,
			})
		} else {
			result = append(result, NetBlock{
				Prefixes: f[0],
				Type:     TypeAnnounced,
			})
		}
	}
	return &result, nil
}

func getHttpC() *http.Client {
	transport := &http.Transport{
		Proxy: nil,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}
}
