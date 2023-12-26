package jupag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gojek/heimdall/v7"
	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/verzth/go-jup-ag/utils"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Jupag interface {
	request(method, endpoint string, params, body any) (*http.Response, error)
	parseResponse(resp *http.Response) (json.RawMessage, error)
	Quote(params QuoteParams) (QuoteResponse, error)
	Swap(params SwapParams) (string, error)
	Price(params PriceParams) (PriceMap, error)
	RoutesMap(onlyDirectRoutes bool) (IndexedRoutesMap, error)
	BestSwap(params BestSwapParams) (string, error)
	ExchangeRate(params ExchangeRateParams) (Rate, error)
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
	timeout := 500 * time.Millisecond
	cl := httpclient.NewClient(
		httpclient.WithHTTPTimeout(timeout),
		httpclient.WithRetryCount(1),
		httpclient.WithRetrier(heimdall.NewRetrier(heimdall.NewConstantBackoff(500*time.Millisecond, 1000*time.Millisecond))),
	)

	return &JupagImpl{
		jupagImpl:     cl,
		apiUrl:        "https://price.jup.ag/v6",
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
func (c *JupagImpl) Quote(params QuoteParams) (QuoteResponse, error) {
	resp, err := c.request(http.MethodGet, fmt.Sprintf("%s%s", c.apiUrl, c.quotePath), params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make quote request: %w", err)
	}

	data, err := c.parseResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse quote response: %w", err)
	}

	var quotes QuoteResponse
	if err := json.Unmarshal(data, &quotes); err != nil {
		return nil, fmt.Errorf("failed to parse quote response: %w", err)
	}

	if len(quotes) == 0 {
		return nil, fmt.Errorf("no quotes returned")
	}

	return quotes, nil
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

// BestSwap returns the ebase64 encoded transaction for the best swap route
// for a given input mint, output mint and amount.
// Default swap mode: ExactOut, so the amount is the amount of output token.
// Default wrap unwrap sol: true
func (c *JupagImpl) BestSwap(params BestSwapParams) (string, error) {
	if params.SwapMode == "" {
		params.SwapMode = SwapModeExactIn
	}
	routes, err := c.Quote(QuoteParams{
		InputMint:        params.InputMint,
		OutputMint:       params.OutputMint,
		Amount:           params.Amount,
		FeeBps:           params.FeeAmount,
		SwapMode:         params.SwapMode,
		OnlyDirectRoutes: false,
	})
	if err != nil {
		return "", err
	}

	route, err := routes.GetBestRoute()
	if err != nil {
		return "", err
	}

	swap, err := c.Swap(SwapParams{
		Route:               route,
		UserPublicKey:       params.UserPublicKey,
		DestinationWallet:   params.DestinationPublicKey,
		FeeAccount:          params.FeeAccount,
		WrapUnwrapSol:       utils.Pointer(true),
		AsLegacyTransaction: utils.Pointer(true),
	})
	if err != nil {
		return "", err
	}

	return swap, nil
}

// ExchangeRate returns the exchange rate for a given input mint, output mint and amount.
// Default swap mode: ExactOut, so the amount is the amount of output token.
func (c *JupagImpl) ExchangeRate(params ExchangeRateParams) (Rate, error) {
	result := Rate{
		InputMint:  params.InputMint,
		OutputMint: params.OutputMint,
	}
	routes, err := c.Quote(QuoteParams{
		InputMint:        params.InputMint,
		OutputMint:       params.OutputMint,
		Amount:           params.Amount,
		SwapMode:         params.SwapMode,
		OnlyDirectRoutes: false,
	})
	if err != nil {
		return result, err
	}

	route, err := routes.GetBestRoute()
	if err != nil {
		return result, err
	}

	inAmount, err := strconv.ParseInt(route.InAmount, 10, 64)
	if err != nil {
		return result, fmt.Errorf("failed to parse in amount: %w", err)
	}
	outAmount, err := strconv.ParseInt(route.OutAmount, 10, 64)
	if err != nil {
		return result, fmt.Errorf("failed to parse out amount: %w", err)
	}

	result.InAmount = uint64(inAmount)
	result.OutAmount = uint64(outAmount)

	return result, nil
}
