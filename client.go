package jupag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gojek/heimdall/v7"
	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/verzth/go-jup-ag/utils"
)

type Jupag interface {
	request(method, endpoint string, params, body any) (*http.Response, error)
	parseResponse(resp *http.Response) (json.RawMessage, error)
	Quote(params QuoteParams) (QuoteResponse, error)
	Swap(params SwapParams) (string, error)
	Price(params PriceParams) (PriceMap, error)
	RoutesMap(onlyDirectRoutes bool) (IndexedRoutesMap, error)
}

type JupagImpl struct {
	jupagImpl     *httpclient.Client
	apiUrl        string
	quotePath     string
	swapPath      string
	pricePath     string
	routesMapPath string
}

func NewJupag() Jupag {
	timeout := 3000 * time.Millisecond
	cl := httpclient.NewClient(
		httpclient.WithHTTPTimeout(timeout),
		httpclient.WithRetryCount(1),
		httpclient.WithRetrier(heimdall.NewRetrier(heimdall.NewConstantBackoff(500*time.Millisecond, 1000*time.Millisecond))),
	)

	return &JupagImpl{
		jupagImpl:     cl,
		apiUrl:        "https://quote-proxy.jup.ag",
		quotePath:     "/quote",
		swapPath:      "/swap",
		pricePath:     "/price",
		routesMapPath: "/indexed-route-map",
	}
}

func (c *JupagImpl) request(method, endpoint string, params, body any) (*http.Response, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	uv, err := utils.StructToUrlValues(params)
	if err != nil {
		return nil, fmt.Errorf("failed to convert params to url values: %w", err)
	}

	u.RawQuery = uv.Encode()

	completeUrl := u.String()

	data, err := json.Marshal(body)
	if body != nil && err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, completeUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cache-Control", "no-cache")

	return c.jupagImpl.Do(req)
}

// parseResponse parses the response body into the given response structure.
func (c *JupagImpl) parseResponse(resp *http.Response) (json.RawMessage, error) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Data, nil
}

// Quote returns a quote for a given input mint, output mint and amount
func (c *JupagImpl) Quote(params QuoteParams) (quote QuoteResponse, err error) {
	resp, err := c.request(http.MethodGet, fmt.Sprintf("%s%s", c.apiUrl, c.quotePath), params, nil)
	if err != nil {
		return
	}

	data, err := c.parseResponse(resp)
	if err != nil {
		err = fmt.Errorf("failed to parse quote response: %w", err)
		return
	}

	if err = json.Unmarshal(data, &quote); err != nil {
		err = fmt.Errorf("failed to parse quote response: %w", err)
		return
	}

	return
}

// Swap returns swap base64 serialized transaction for a route.
// The caller is responsible for signing the transactions.
func (c *JupagImpl) Swap(params SwapParams) (string, error) {
	resp, err := c.request(http.MethodPost, fmt.Sprintf("%s%s", c.apiUrl, c.swapPath), nil, params)
	if err != nil {
		return "", fmt.Errorf("failed to make swap request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response SwapResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return response.SwapTransaction, nil
}

// Price returns simple price for a given input mint, output mint and amount.
func (c *JupagImpl) Price(params PriceParams) (PriceMap, error) {
	resp, err := c.request(http.MethodGet, fmt.Sprintf("%s%s", c.apiUrl, c.pricePath), params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make price request: %w", err)
	}

	data, err := c.parseResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price response: %w", err)
	}

	var price PriceMap
	if err := json.Unmarshal(data, &price); err != nil {
		return nil, fmt.Errorf("failed to parse price response: %w", err)
	}

	return price, nil
}

// RoutesMap returns a hash map, input mint as key and an array of valid output mint as values,
// token mints are indexed to reduce the file size.
func (c *JupagImpl) RoutesMap(onlyDirectRoutes bool) (IndexedRoutesMap, error) {
	resp, err := c.request(http.MethodGet, fmt.Sprintf("%s%s", c.apiUrl, c.routesMapPath), url.Values{
		"onlyDirectRoutes": []string{strconv.FormatBool(onlyDirectRoutes)},
	}, nil)
	if err != nil {
		return IndexedRoutesMap{}, fmt.Errorf("failed to make routes map request: %w", err)
	}

	var routesMap IndexedRoutesMap
	if err := json.NewDecoder(resp.Body).Decode(&routesMap); err != nil {
		return IndexedRoutesMap{}, fmt.Errorf("failed to parse routes map response: %w", err)
	}

	return routesMap, nil
}
