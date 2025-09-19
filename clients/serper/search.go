package serper

import (
	"context"
	"fmt"
)

// Search performs a search operation with the specified options
func (c *Client) Search(ctx context.Context, request *SearchRequest) (*SearchResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("search request cannot be nil")
	}

	if request.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	// Set default values if not provided
	if request.Type == "" {
		request.Type = SearchTypeSearch
	}

	if request.GL == "" {
		request.GL = "us"
	}

	if request.HL == "" {
		request.HL = "en"
	}

	if request.Num == 0 {
		request.Num = 10
	}

	var response SearchResponse
	
	// Create request with rate limiting
	req, err := c.createRequestWithContext(ctx)
	if err != nil {
		return nil, err
	}
	
	req = req.
		SetContext(ctx).
		SetBody(request).
		SetResult(&response)

	resp, err := req.Post(fmt.Sprintf("%s/search", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// Handle API errors
	if err := handleError(resp); err != nil {
		return nil, err
	}
	
	// Resty automatically unmarshals JSON response into the struct we provided with SetResult
	// The response variable should now contain the unmarshaled data

	return &response, nil
}

// SearchSimple performs a simple search with minimal options
func (c *Client) SearchSimple(ctx context.Context, query string) (*SearchResponse, error) {
	return c.Search(ctx, &SearchRequest{
		Query: query,
	})
}

// SearchWithType performs a search with a specific search type
func (c *Client) SearchWithType(ctx context.Context, query string, searchType SearchType) (*SearchResponse, error) {
	return c.Search(ctx, &SearchRequest{
		Query: query,
		Type:  searchType,
	})
}

// SearchImages performs an image search
func (c *Client) SearchImages(ctx context.Context, query string) (*SearchResponse, error) {
	return c.SearchWithType(ctx, query, SearchTypeImages)
}

// SearchVideos performs a video search
func (c *Client) SearchVideos(ctx context.Context, query string) (*SearchResponse, error) {
	return c.SearchWithType(ctx, query, SearchTypeVideos)
}

// SearchNews performs a news search
func (c *Client) SearchNews(ctx context.Context, query string) (*SearchResponse, error) {
	return c.SearchWithType(ctx, query, SearchTypeNews)
}

// SearchShopping performs a shopping search
func (c *Client) SearchShopping(ctx context.Context, query string) (*SearchResponse, error) {
	return c.SearchWithType(ctx, query, SearchTypeShopping)
}

// SearchPlaces performs a places search
func (c *Client) SearchPlaces(ctx context.Context, query string) (*SearchResponse, error) {
	return c.SearchWithType(ctx, query, SearchTypePlaces)
}

// SearchScholar performs a Google Scholar search
func (c *Client) SearchScholar(ctx context.Context, query string) (*SearchResponse, error) {
	return c.SearchWithType(ctx, query, SearchTypeScholar)
}

// SearchPatents performs a patent search
func (c *Client) SearchPatents(ctx context.Context, query string) (*SearchResponse, error) {
	return c.SearchWithType(ctx, query, SearchTypePatents)
}

// SearchWithLocation performs a search with a specific location
func (c *Client) SearchWithLocation(ctx context.Context, query string, location string) (*SearchResponse, error) {
	return c.Search(ctx, &SearchRequest{
		Query:    query,
		Location: location,
	})
}

// SearchWithPagination performs a search with pagination
func (c *Client) SearchWithPagination(ctx context.Context, query string, page int, num int) (*SearchResponse, error) {
	return c.Search(ctx, &SearchRequest{
		Query: query,
		Page:  page,
		Num:   num,
	})
}

// SearchWithLanguage performs a search with specific language settings
func (c *Client) SearchWithLanguage(ctx context.Context, query string, gl string, hl string) (*SearchResponse, error) {
	return c.Search(ctx, &SearchRequest{
		Query: query,
		GL:    gl,
		HL:    hl,
	})
}