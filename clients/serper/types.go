package serper

import "time"

// SearchType represents the type of search
type SearchType string

const (
	SearchTypeSearch   SearchType = "search"
	SearchTypeImages   SearchType = "images"
	SearchTypeVideos   SearchType = "videos"
	SearchTypeNews     SearchType = "news"
	SearchTypeShopping SearchType = "shopping"
	SearchTypePlaces   SearchType = "places"
	SearchTypeScholar  SearchType = "scholar"
	SearchTypePatents  SearchType = "patents"
)

// SearchRequest represents the request options for the search endpoint
type SearchRequest struct {
	Query      string            `json:"q"`
	Type       SearchType        `json:"type,omitempty"`
	GL         string            `json:"gl,omitempty"`
	HL         string            `json:"hl,omitempty"`
	Num        int               `json:"num,omitempty"`
	Page       int               `json:"page,omitempty"`
	SafeSearch bool              `json:"safe,omitempty"`
	TBS        string            `json:"tbs,omitempty"`
	Location   string            `json:"location,omitempty"`
	AutoCorrect bool             `json:"autocorrect,omitempty"`
}

// SearchResponse represents the response from the search endpoint
type SearchResponse struct {
	SearchParameters SearchParameters  `json:"searchParameters,omitempty"`
	Organic          []OrganicResult   `json:"organic,omitempty"`
	AnswerBox        *AnswerBox        `json:"answerBox,omitempty"`
	KnowledgeGraph   *KnowledgeGraph   `json:"knowledgeGraph,omitempty"`
	TopStories       []TopStory        `json:"topStories,omitempty"`
	PeopleAlsoAsk    []PeopleAlsoAsk   `json:"peopleAlsoAsk,omitempty"`
	RelatedSearches  []RelatedSearch   `json:"relatedSearches,omitempty"`
	Images           []ImageResult     `json:"images,omitempty"`
	Videos           []VideoResult     `json:"videos,omitempty"`
	News             []NewsResult      `json:"news,omitempty"`
	Shopping         []ShoppingResult  `json:"shopping,omitempty"`
	Places           []PlaceResult     `json:"places,omitempty"`
	Scholar          []ScholarResult   `json:"scholar,omitempty"`
	Patents          []PatentResult    `json:"patents,omitempty"`
	Credits          int               `json:"credits,omitempty"`
	RequestId        string            `json:"requestId,omitempty"`
}

// SearchParameters represents the search parameters used
type SearchParameters struct {
	Q            string `json:"q,omitempty"`
	Type         string `json:"type,omitempty"`
	Engine       string `json:"engine,omitempty"`
	GoogleDomain string `json:"googleDomain,omitempty"`
	GL           string `json:"gl,omitempty"`
	HL           string `json:"hl,omitempty"`
	Num          int    `json:"num,omitempty"`
	Page         int    `json:"page,omitempty"`
}

// OrganicResult represents an organic search result
type OrganicResult struct {
	Title       string      `json:"title,omitempty"`
	Link        string      `json:"link,omitempty"`
	Snippet     string      `json:"snippet,omitempty"`
	Position    int         `json:"position,omitempty"`
	Date        string      `json:"date,omitempty"`
	Sitelinks   []Sitelink  `json:"sitelinks,omitempty"`
	RichSnippet RichSnippet `json:"richSnippet,omitempty"`
}

// Sitelink represents a sitelink in search results
type Sitelink struct {
	Title string `json:"title,omitempty"`
	Link  string `json:"link,omitempty"`
}

// RichSnippet represents rich snippet data
type RichSnippet struct {
	Top    map[string]interface{} `json:"top,omitempty"`
	Bottom map[string]interface{} `json:"bottom,omitempty"`
}

// AnswerBox represents an answer box in search results
type AnswerBox struct {
	Snippet          string `json:"snippet,omitempty"`
	SnippetHighlighted []string `json:"snippetHighlighted,omitempty"`
	Title            string `json:"title,omitempty"`
	Link             string `json:"link,omitempty"`
}

// KnowledgeGraph represents knowledge graph data
type KnowledgeGraph struct {
	Title       string                 `json:"title,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Description string                 `json:"description,omitempty"`
	DescriptionSource string            `json:"descriptionSource,omitempty"`
	DescriptionLink   string            `json:"descriptionLink,omitempty"`
	ImageUrl    string                 `json:"imageUrl,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

// TopStory represents a top story result
type TopStory struct {
	Title     string `json:"title,omitempty"`
	Link      string `json:"link,omitempty"`
	Source    string `json:"source,omitempty"`
	Date      string `json:"date,omitempty"`
	ImageUrl  string `json:"imageUrl,omitempty"`
}

// PeopleAlsoAsk represents a "People also ask" item
type PeopleAlsoAsk struct {
	Question string `json:"question,omitempty"`
	Snippet  string `json:"snippet,omitempty"`
	Title    string `json:"title,omitempty"`
	Link     string `json:"link,omitempty"`
}

// RelatedSearch represents a related search query
type RelatedSearch struct {
	Query string `json:"query,omitempty"`
	Link  string `json:"link,omitempty"`
}

// ImageResult represents an image search result
type ImageResult struct {
	Title        string  `json:"title,omitempty"`
	ImageUrl     string  `json:"imageUrl,omitempty"`
	ImageWidth   int     `json:"imageWidth,omitempty"`
	ImageHeight  int     `json:"imageHeight,omitempty"`
	ThumbnailUrl string  `json:"thumbnailUrl,omitempty"`
	ThumbnailWidth  int  `json:"thumbnailWidth,omitempty"`
	ThumbnailHeight int  `json:"thumbnailHeight,omitempty"`
	Source       string  `json:"source,omitempty"`
	Domain       string  `json:"domain,omitempty"`
	Link         string  `json:"link,omitempty"`
	Position     int     `json:"position,omitempty"`
}

// VideoResult represents a video search result
type VideoResult struct {
	Title       string    `json:"title,omitempty"`
	Link        string    `json:"link,omitempty"`
	Snippet     string    `json:"snippet,omitempty"`
	Date        string    `json:"date,omitempty"`
	ImageUrl    string    `json:"imageUrl,omitempty"`
	Duration    string    `json:"duration,omitempty"`
	Source      string    `json:"source,omitempty"`
	Channel     string    `json:"channel,omitempty"`
	Platform    string    `json:"platform,omitempty"`
	Position    int       `json:"position,omitempty"`
}

// NewsResult represents a news search result
type NewsResult struct {
	Title       string    `json:"title,omitempty"`
	Link        string    `json:"link,omitempty"`
	Snippet     string    `json:"snippet,omitempty"`
	Date        string    `json:"date,omitempty"`
	Source      string    `json:"source,omitempty"`
	ImageUrl    string    `json:"imageUrl,omitempty"`
	Position    int       `json:"position,omitempty"`
}

// ShoppingResult represents a shopping search result
type ShoppingResult struct {
	Title       string  `json:"title,omitempty"`
	Source      string  `json:"source,omitempty"`
	Link        string  `json:"link,omitempty"`
	Price       string  `json:"price,omitempty"`
	ExtractedPrice float64 `json:"extractedPrice,omitempty"`
	Currency    string  `json:"currency,omitempty"`
	ImageUrl    string  `json:"imageUrl,omitempty"`
	Rating      float64 `json:"rating,omitempty"`
	RatingCount int     `json:"ratingCount,omitempty"`
	Delivery    string  `json:"delivery,omitempty"`
	Position    int     `json:"position,omitempty"`
}

// PlaceResult represents a place search result
type PlaceResult struct {
	Position    int       `json:"position,omitempty"`
	Title       string    `json:"title,omitempty"`
	Address     string    `json:"address,omitempty"`
	Latitude    float64   `json:"latitude,omitempty"`
	Longitude   float64   `json:"longitude,omitempty"`
	Rating      float64   `json:"rating,omitempty"`
	RatingCount int       `json:"ratingCount,omitempty"`
	Category    string    `json:"category,omitempty"`
	PhoneNumber string    `json:"phoneNumber,omitempty"`
	Website     string    `json:"website,omitempty"`
	CID         string    `json:"cid,omitempty"`
}

// ScholarResult represents a Google Scholar search result
type ScholarResult struct {
	Title       string    `json:"title,omitempty"`
	Link        string    `json:"link,omitempty"`
	Snippet     string    `json:"snippet,omitempty"`
	Publication string    `json:"publication,omitempty"`
	Authors     []string  `json:"authors,omitempty"`
	Year        int       `json:"year,omitempty"`
	Citations   int       `json:"citations,omitempty"`
	PDFLink     string    `json:"pdfLink,omitempty"`
	Position    int       `json:"position,omitempty"`
}

// PatentResult represents a patent search result
type PatentResult struct {
	Title       string    `json:"title,omitempty"`
	Link        string    `json:"link,omitempty"`
	Snippet     string    `json:"snippet,omitempty"`
	PatentId    string    `json:"patentId,omitempty"`
	Assignee    string    `json:"assignee,omitempty"`
	Inventor    []string  `json:"inventor,omitempty"`
	Date        time.Time `json:"date,omitempty"`
	PDFLink     string    `json:"pdfLink,omitempty"`
	Position    int       `json:"position,omitempty"`
}