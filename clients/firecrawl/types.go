package firecrawl

import "time"

// ScrapeOptions represents the request options for the scrape endpoint
type ScrapeOptions struct {
	URL                 string                 `json:"url"`
	Formats             []Format               `json:"formats,omitempty"`
	OnlyMainContent     *bool                  `json:"onlyMainContent,omitempty"`
	IncludeTags         []string               `json:"includeTags,omitempty"`
	ExcludeTags         []string               `json:"excludeTags,omitempty"`
	MaxAge              *int                   `json:"maxAge,omitempty"`
	Headers             map[string]string      `json:"headers,omitempty"`
	WaitFor             *int                   `json:"waitFor,omitempty"`
	Mobile              *bool                  `json:"mobile,omitempty"`
	SkipTLSVerification *bool                  `json:"skipTlsVerification,omitempty"`
	Timeout             *int                   `json:"timeout,omitempty"`
	Parsers             []Parser               `json:"parsers,omitempty"`
	Actions             []Action               `json:"actions,omitempty"`
	Location            *Location              `json:"location,omitempty"`
	RemoveBase64Images  *bool                  `json:"removeBase64Images,omitempty"`
	BlockAds            *bool                  `json:"blockAds,omitempty"`
	Proxy               *string                `json:"proxy,omitempty"`
	StoreInCache        *bool                  `json:"storeInCache,omitempty"`
	ZeroDataRetention   *bool                  `json:"zeroDataRetention,omitempty"`
}

// Format represents the output format configuration
type Format interface {
	formatMarker()
}

// FormatType represents simple format types
type FormatType string

const (
	FormatMarkdown FormatType = "markdown"
	FormatSummary  FormatType = "summary"
	FormatHTML     FormatType = "html"
	FormatRawHTML  FormatType = "rawHtml"
	FormatLinks    FormatType = "links"
)

// SimpleFormat represents a simple format type
type SimpleFormat struct {
	Type FormatType `json:"type"`
}

func (SimpleFormat) formatMarker() {}

// ScreenshotFormat represents screenshot format configuration
type ScreenshotFormat struct {
	Type     string    `json:"type"`
	FullPage *bool     `json:"fullPage,omitempty"`
	Quality  *int      `json:"quality,omitempty"`
	Viewport *Viewport `json:"viewport,omitempty"`
}

func (ScreenshotFormat) formatMarker() {}

// JSONFormat represents JSON extraction format
type JSONFormat struct {
	Type   string                 `json:"type"`
	Schema map[string]interface{} `json:"schema,omitempty"`
	Prompt string                 `json:"prompt,omitempty"`
}

func (JSONFormat) formatMarker() {}

// ChangeTrackingFormat represents change tracking format
type ChangeTrackingFormat struct {
	Type   string                 `json:"type"`
	Modes  []string               `json:"modes,omitempty"`
	Schema map[string]interface{} `json:"schema,omitempty"`
	Prompt string                 `json:"prompt,omitempty"`
	Tag    *string                `json:"tag,omitempty"`
}

func (ChangeTrackingFormat) formatMarker() {}

// Viewport represents viewport dimensions
type Viewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Parser represents a parser configuration
type Parser interface {
	parserMarker()
}

// SimpleParser represents a simple parser type
type SimpleParser string

const (
	ParserPDF SimpleParser = "pdf"
)

func (SimpleParser) parserMarker() {}

// PDFParser represents PDF parser configuration
type PDFParser struct {
	Type     string `json:"type"`
	MaxPages *int   `json:"maxPages,omitempty"`
}

func (PDFParser) parserMarker() {}

// Action represents an action to perform on the page
type Action interface {
	actionMarker()
}

// WaitAction represents a wait action
type WaitAction struct {
	Type         string  `json:"type"`
	Milliseconds *int    `json:"milliseconds,omitempty"`
	Selector     *string `json:"selector,omitempty"`
}

func (WaitAction) actionMarker() {}

// ScreenshotAction represents a screenshot action
type ScreenshotAction struct {
	Type     string    `json:"type"`
	FullPage *bool     `json:"fullPage,omitempty"`
	Quality  *int      `json:"quality,omitempty"`
	Viewport *Viewport `json:"viewport,omitempty"`
}

func (ScreenshotAction) actionMarker() {}

// ClickAction represents a click action
type ClickAction struct {
	Type     string `json:"type"`
	Selector string `json:"selector"`
	All      *bool  `json:"all,omitempty"`
}

func (ClickAction) actionMarker() {}

// WriteAction represents a write text action
type WriteAction struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (WriteAction) actionMarker() {}

// PressAction represents a key press action
type PressAction struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

func (PressAction) actionMarker() {}

// ScrollAction represents a scroll action
type ScrollAction struct {
	Type      string  `json:"type"`
	Direction *string `json:"direction,omitempty"`
	Selector  *string `json:"selector,omitempty"`
}

func (ScrollAction) actionMarker() {}

// ScrapeAction represents a scrape action
type ScrapeAction struct {
	Type string `json:"type"`
}

func (ScrapeAction) actionMarker() {}

// ExecuteJavaScriptAction represents a JavaScript execution action
type ExecuteJavaScriptAction struct {
	Type   string `json:"type"`
	Script string `json:"script"`
}

func (ExecuteJavaScriptAction) actionMarker() {}

// PDFAction represents a PDF generation action
type PDFAction struct {
	Type      string  `json:"type"`
	Format    *string `json:"format,omitempty"`
	Landscape *bool   `json:"landscape,omitempty"`
	Scale     *float64 `json:"scale,omitempty"`
}

func (PDFAction) actionMarker() {}

// Location represents location settings for the request
type Location struct {
	Country   string   `json:"country,omitempty"`
	Languages []string `json:"languages,omitempty"`
}

// ScrapeResponse represents the response from the scrape endpoint
type ScrapeResponse struct {
	Success bool       `json:"success"`
	Data    *ScrapeData `json:"data,omitempty"`
	Error   string     `json:"error,omitempty"`
	Code    string     `json:"code,omitempty"`
}

// ScrapeData represents the scraped data
type ScrapeData struct {
	Markdown       string          `json:"markdown,omitempty"`
	Summary        *string         `json:"summary,omitempty"`
	HTML           *string         `json:"html,omitempty"`
	RawHTML        *string         `json:"rawHtml,omitempty"`
	Screenshot     *string         `json:"screenshot,omitempty"`
	Links          []string        `json:"links,omitempty"`
	Actions        *ActionsResult  `json:"actions,omitempty"`
	Metadata       *Metadata       `json:"metadata,omitempty"`
	Warning        *string         `json:"warning,omitempty"`
	ChangeTracking *ChangeTracking `json:"changeTracking,omitempty"`
}

// ActionsResult represents the results of actions
type ActionsResult struct {
	Screenshots       []string           `json:"screenshots,omitempty"`
	Scrapes          []ScrapeResult     `json:"scrapes,omitempty"`
	JavaScriptReturns []JavaScriptReturn `json:"javascriptReturns,omitempty"`
	PDFs             []string           `json:"pdfs,omitempty"`
}

// ScrapeResult represents a scrape action result
type ScrapeResult struct {
	URL  string `json:"url"`
	HTML string `json:"html"`
}

// JavaScriptReturn represents a JavaScript execution result
type JavaScriptReturn struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// Metadata represents page metadata
type Metadata struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Language    *string `json:"language,omitempty"`
	SourceURL   string `json:"sourceURL,omitempty"`
	StatusCode  int    `json:"statusCode,omitempty"`
	Error       *string `json:"error,omitempty"`
}

// ChangeTracking represents change tracking information
type ChangeTracking struct {
	PreviousScrapeAt *time.Time             `json:"previousScrapeAt,omitempty"`
	ChangeStatus     string                 `json:"changeStatus,omitempty"`
	Visibility       string                 `json:"visibility,omitempty"`
	Diff             *string                `json:"diff,omitempty"`
	JSON             map[string]interface{} `json:"json,omitempty"`
}

// ProxyType represents the type of proxy to use
type ProxyType string

const (
	ProxyBasic   ProxyType = "basic"
	ProxyStealth ProxyType = "stealth"
	ProxyAuto    ProxyType = "auto"
)