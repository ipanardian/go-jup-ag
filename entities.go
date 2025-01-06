package jupag

import (
	"encoding/json"
	"strconv"
)

const (
	SwapModeExactIn  = "ExactIn"
	SwapModeExactOut = "ExactOut"
)

type Response struct {
	Data        json.RawMessage `json:"data"`
	TimeTaken   float64         `json:"timeTaken"`
	ContextSlot int64           `json:"contextSlot"`
}

// MarketInfo is a market info object structure.
type MarketInfo struct {
	ID                 string  `json:"id"`
	Label              string  `json:"label"`
	InputMint          string  `json:"inputMint"`
	OutputMint         string  `json:"outputMint"`
	NotEnoughLiquidity bool    `json:"notEnoughLiquidity"`
	InAmount           string  `json:"inAmount"`
	OutAmount          string  `json:"outAmount"`
	MinInAmount        string  `json:"minInAmount,omitempty"`
	MinOutAmount       string  `json:"minOutAmount,omitempty"`
	PriceImpactPct     float64 `json:"priceImpactPct"`
	LpFee              *Fee    `json:"lpFee"`
	PlatformFee        *Fee    `json:"platformFee"`
}

// Fee is a fee object structure.
type Fee struct {
	Amount string  `json:"amount"`
	Mint   string  `json:"mint"`
	Pct    float64 `json:"pct"`
}

// Route is a route object structure.
type Route struct {
	Percent  int `json:"percent"`
	SwapInfo struct {
		AmmKey     string `json:"ammKey"`
		FeeAmount  string `json:"feeAmount"`
		FeeMint    string `json:"feeMint"`
		InAmount   string `json:"inAmount"`
		InputMint  string `json:"inputMint"`
		Label      string `json:"label"`
		OutAmount  string `json:"outAmount"`
		OutputMint string `json:"outputMint"`
	} `json:"swapInfo"`
}

// Price is a price object structure.
type Price struct {
	ID            string  `json:"id"`            // Address of the token
	MintSymbol    string  `json:"mintSymbol"`    // Symbol of the token
	VsToken       string  `json:"vsToken"`       // Address of the token to compare against
	VsTokenSymbol string  `json:"vsTokenSymbol"` // Symbol of the token to compare against
	Price         float64 `json:"price"`         // Price of the token in relation to the vsToken. Default to 1 unit of the token worth in USDC if vsToken is not specified.
}

// PriceMap is a price map objects structure.
type PriceMap map[string]Price

// QuoteParams are the parameters for a quote request.
type QuoteParams struct {
	InputMint  string `url:"inputMint"`  // required
	OutputMint string `url:"outputMint"` // required
	Amount     uint64 `url:"amount"`     // required

	SwapMode            string `url:"swapMode,omitempty"` // Swap mode, default is ExactIn; Available values : ExactIn, ExactOut.
	DynamicSlippage     bool   `url:"dynamicSlippage,omitempty"`
	OnlyDirectRoutes    bool   `url:"onlyDirectRoutes,omitempty"`    // Only return direct routes (no hoppings and split trade)
	AsLegacyTransaction bool   `url:"asLegacyTransaction,omitempty"` // Only return routes that can be done in a single legacy transaction. (Routes might be limited)
	MinimizeSlippage    bool   `url:"minimizeSlippage,omitempty"`
}

// QuoteResponse is the response from a quote request.
type QuoteResponse struct {
	ContextSlot          int         `json:"contextSlot"`
	InAmount             string      `json:"inAmount"`
	InputMint            string      `json:"inputMint"`
	OtherAmountThreshold string      `json:"otherAmountThreshold"`
	OutAmount            string      `json:"outAmount"`
	OutputMint           string      `json:"outputMint"`
	PlatformFee          interface{} `json:"platformFee"`
	PriceImpactPct       string      `json:"priceImpactPct"`
	RoutePlan            Route       `json:"routePlan"`
	ScoreReport          interface{} `json:"scoreReport"`
	SlippageBps          int         `json:"slippageBps"`
	SwapMode             string      `json:"swapMode"`
	SwapType             string      `json:"swapType"`
	TimeTaken            float64     `json:"timeTaken"`
}

// SwapParams are the parameters for a swap request.
type SwapParams struct {
	Route                         Route  `json:"route"`                   // required
	UserPublicKey                 string `json:"userPublicKey,omitempty"` // required
	WrapUnwrapSol                 *bool  `json:"wrapUnwrapSOL,omitempty"`
	FeeAccount                    string `json:"feeAccount,omitempty"`                    // Fee token account for the platform fee (only pass in if you set a feeBps), the mint is outputMint for the default swapMode.ExactOut and inputMint for swapMode.ExactIn.
	AsLegacyTransaction           *bool  `json:"asLegacyTransaction,omitempty"`           // Request a legacy transaction rather than the default versioned transaction, needs to be paired with a quote using asLegacyTransaction otherwise the transaction might be too large.
	ComputeUnitPriceMicroLamports *int64 `json:"computeUnitPriceMicroLamports,omitempty"` // Compute unit price to prioritize the transaction, the additional fee will be compute unit consumed * computeUnitPriceMicroLamports.
	DestinationWallet             string `json:"destinationWallet,omitempty"`             // Public key of the wallet that will receive the output of the swap, this assumes the associated token account exists, currently adds a token transfer.
}

// SwapResponse is the response from a swap request.
type SwapResponse struct {
	SwapTransaction string `json:"swapTransaction"` // base64 encoded transaction string
}

// PriceParams are the parameters for a price request.
type PriceParams struct {
	IDs      string  `url:"ids"`                // required; Symbol or address of a token, (e.g. SOL or EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v). Use `,` to query multiple tokens, e.g. (sol,btc,mer,...)
	VsToken  string  `url:"vsToken,omitempty"`  // optional; Default to USDC. Symbol or address of a token, (e.g. SOL or EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v).
	VsAmount float64 `url:"vsAmount,omitempty"` // optional; Unit amount of specified input token. Default to 1.
}

// IndexedRoutesMap is a map of routes indexed by the route ID.
type IndexedRoutesMap struct {
	MintKeys        []string         `json:"mintKeys"`        // All the mints that are indexed to match in indexedRouteMap.
	IndexedRouteMap map[string][]int `json:"indexedRouteMap"` // All the possible route and their corresponding output mints.
}

// GetRoutesForMint returns the routes for a given mint.
func (r *IndexedRoutesMap) GetRoutesForMint(mint string) []string {
	// Find index of mint in mintKeys.
	var mintKeys []int
	for key, val := range r.MintKeys {
		if val == mint {
			mintKeys = r.IndexedRouteMap[strconv.Itoa(key)]
		}
	}

	// Find the mint in mintKeys.
	result := make([]string, 0, len(mintKeys))
	for _, key := range mintKeys {
		result = append(result, r.MintKeys[key])
	}

	return result
}

// BestSwapParams contains the parameters for the best swap route.
type BestSwapParams struct {
	UserPublicKey        string // user base58 encoded public key
	DestinationPublicKey string // destination base58 encoded public key (optional)
	FeeAmount            uint64 // fee amount in token basis points (optional)
	FeeAccount           string // fee token account for the platform fee (only pass in if you set a FeeAmount).
	InputMint            string // input mint
	OutputMint           string // output mint
	Amount               uint64 // amount of output token
	SwapMode             string // swap mode, default: ExactIn (Available: ExactIn, ExactOut)
}

// ExchangeRateParams contains the parameters for the exchange rate request.
type ExchangeRateParams struct {
	InputMint  string // input token mint
	OutputMint string // output token mint
	Amount     uint64 // amount of token, depending on the swap mode
	SwapMode   string // swap mode, default: ExactOut (Available: ExactIn, ExactOut)
}

// ExchangeRate returns the exchange rate for a given input mint, output mint and amount.
type Rate struct {
	InputMint  string `json:"inputMint"`  // input token mint
	OutputMint string `json:"outputMint"` // output token mint
	InAmount   uint64 `json:"inAmount"`   // amount of input token
	OutAmount  uint64 `json:"outAmount"`  // amount of output token
}
