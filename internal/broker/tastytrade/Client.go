package tastytrade

import (
	"context"
	"fmt"
)

var ErrSymbolNotFound = fmt.Errorf("symbol not found")

type Client interface {
	ListAccounts(ctx context.Context) ([]*Accounts, error)
	GetAccountBalance(ctx context.Context, accountID string) (*AccountBalance, error)
	GetAccountPositions(ctx context.Context, accountID string) (*AccountPositionsOutput, error)
	SearchSymbol(ctx context.Context, symbol string) (*Symbol, error)
	GetAPIQuoteToken(ctx context.Context) (*GetAPIQuoteTokenOutput, error)
	PlaceEquityOrder(ctx context.Context, input PlaceEquityOrderInput) (*PlaceEquityOrderOutput, error)
	GetLiveOrders(ctx context.Context, accountID string) (*LiveOrdersOutput, error)
	GetAccountTransactions(ctx context.Context, input GetAccountTransactionsInput) (*AccountTransactionsOutput, error)
}

type GetAccountTransactionsInput struct {
	AccountID string
	StartDate string
	EndDate   string
	PageOffset int
	PerPage    int
}

type AccountTransactionsOutput struct {
	Data       AccountTransactionsData `json:"data"`
	Pagination AccountTransactionsPagination `json:"pagination"`
}

type AccountTransactionsData struct {
	Items []AccountTransaction `json:"items"`
}

type AccountTransactionsPagination struct {
	PerPage     int `json:"per-page"`
	PageOffset  int `json:"page-offset"`
	ItemOffset  int `json:"item-offset"`
	TotalItems  int `json:"total-items"`
	TotalPages  int `json:"total-pages"`
	CurrentItemCount int `json:"current-item-count"`
	PreviousLink string `json:"previous-link"`
	NextLink    string `json:"next-link"`
	PagingLinkTemplate string `json:"paging-link-template"`
}

type AccountTransaction struct {
	ID                  int    `json:"id"`
	AccountNumber       string `json:"account-number"`
	Symbol              string `json:"symbol"`
	InstrumentType      string `json:"instrument-type"`
	TransactionType     string `json:"transaction-type"`
	TransactionSubType  string `json:"transaction-sub-type"`
	Action              string `json:"action"`
	Quantity            string `json:"quantity"`
	Price               string `json:"price"`
	Value               string `json:"value"`
	ValueEffect         string `json:"value-effect"`
	RegulatoryFees      string `json:"regulatory-fees"`
	RegulatoryFeesEffect string `json:"regulatory-fees-effect"`
	ClearingFees        string `json:"clearing-fees"`
	ClearingFeesEffect  string `json:"clearing-fees-effect"`
	Commission          string `json:"commission"`
	CommissionEffect    string `json:"commission-effect"`
	NetValue            string `json:"net-value"`
	NetValueEffect      string `json:"net-value-effect"`
	ExecutedAt          string `json:"executed-at"`
	TransactionDate     string `json:"transaction-date"`
	Description         string `json:"description"`
}

type AccountPositionsOutput struct {
	Data AccountPositionsData `json:"data"`
}

type AccountPositionsData struct {
	Items []AccountPosition `json:"items"`
}

type AccountPosition struct {
	Symbol            string `json:"symbol"`
	InstrumentType    string `json:"instrument-type"`
	Quantity          any    `json:"quantity"`
	QuantityDirection string `json:"quantity-direction"`
}

type Accounts struct {
	Data AccountsData `json:"data"`
}

type AccountsData struct {
	Items []AccountsItem `json:"items"`
}

type AccountsItem struct {
	Account        Account `json:"account"`
	AuthorityLevel string  `json:"authority-level"`
}

type Account struct {
	AccountNumber        string `json:"account-number"`
	AccountTypeName      string `json:"account-type-name"`
	CreatedAt            string `json:"created-at"`
	DayTraderStatus      bool   `json:"day-trader-status"`
	ExtCrmID             string `json:"ext-crm-id"`
	ExternalID           string `json:"external-id"`
	FundingDate          string `json:"funding-date"`
	InvestmentObjective  string `json:"investment-objective"`
	IsClosed             bool   `json:"is-closed"`
	IsFirmError          bool   `json:"is-firm-error"`
	IsFirmProprietary    bool   `json:"is-firm-proprietary"`
	IsForeign            bool   `json:"is-foreign"`
	IsFuturesApproved    bool   `json:"is-futures-approved"`
	LiquidityNeeds       string `json:"liquidity-needs"`
	MarginOrCash         string `json:"margin-or-cash"`
	Nickname             string `json:"nickname"`
	OpenedAt             string `json:"opened-at"`
	RegulatoryDomain     string `json:"regulatory-domain"`
	SuitableOptionsLevel string `json:"suitable-options-level"`
}

type AccountBalance struct {
	Data    AccountBalanceData `json:"data"`
	Context string             `json:"context"`
}

type AccountBalanceData struct {
	AccountNumber                          string `json:"account-number"`
	AvailableTradingFunds                  string `json:"available-trading-funds"`
	BondMarginRequirement                  string `json:"bond-margin-requirement"`
	CashAvailableToWithdraw                string `json:"cash-available-to-withdraw"`
	CashBalance                            string `json:"cash-balance"`
	CashSettleBalance                      string `json:"cash-settle-balance"`
	ClosedLoopAvailableBalance             string `json:"closed-loop-available-balance"`
	CryptocurrencyMarginRequirement        string `json:"cryptocurrency-margin-requirement"`
	Currency                               string `json:"currency"`
	DayEquityCallValue                     string `json:"day-equity-call-value"`
	DayTradeExcess                         string `json:"day-trade-excess"`
	DayTradingBuyingPower                  string `json:"day-trading-buying-power"`
	DayTradingCallValue                    string `json:"day-trading-call-value"`
	DerivativeBuyingPower                  string `json:"derivative-buying-power"`
	EquityBuyingPower                      string `json:"equity-buying-power"`
	EquityOfferingMarginRequirement        string `json:"equity-offering-margin-requirement"`
	FixedIncomeSecurityMarginRequirement   string `json:"fixed-income-security-margin-requirement"`
	FuturesMarginRequirement               string `json:"futures-margin-requirement"`
	IntradayEquitiesCashAmount             string `json:"intraday-equities-cash-amount"`
	IntradayEquitiesCashEffect             string `json:"intraday-equities-cash-effect"`
	IntradayEquitiesCashEffectiveDate      string `json:"intraday-equities-cash-effective-date"`
	IntradayFuturesCashAmount              string `json:"intraday-futures-cash-amount"`
	IntradayFuturesCashEffect              string `json:"intraday-futures-cash-effect"`
	LongBondValue                          string `json:"long-bond-value"`
	LongCryptocurrencyValue                string `json:"long-cryptocurrency-value"`
	LongDerivativeValue                    string `json:"long-derivative-value"`
	LongEquityValue                        string `json:"long-equity-value"`
	LongFixedIncomeSecurityValue           string `json:"long-fixed-income-security-value"`
	LongFuturesDerivativeValue             string `json:"long-futures-derivative-value"`
	LongFuturesValue                       string `json:"long-futures-value"`
	LongMargineableValue                   string `json:"long-margineable-value"`
	MaintenanceCallValue                   string `json:"maintenance-call-value"`
	MaintenanceRequirement                 string `json:"maintenance-requirement"`
	MarginEquity                           string `json:"margin-equity"`
	MarginSettleBalance                    string `json:"margin-settle-balance"`
	NetLiquidatingValue                    string `json:"net-liquidating-value"`
	PendingCash                            string `json:"pending-cash"`
	PendingCashEffect                      string `json:"pending-cash-effect"`
	PreviousDayCryptocurrencyFiatAmount    string `json:"previous-day-cryptocurrency-fiat-amount"`
	PreviousDayCryptocurrencyFiatEffect    string `json:"previous-day-cryptocurrency-fiat-effect"`
	RegTCallValue                          string `json:"reg-t-call-value"`
	ShortCryptocurrencyValue               string `json:"short-cryptocurrency-value"`
	ShortDerivativeValue                   string `json:"short-derivative-value"`
	ShortEquityValue                       string `json:"short-equity-value"`
	ShortFuturesDerivativeValue            string `json:"short-futures-derivative-value"`
	ShortFuturesValue                      string `json:"short-futures-value"`
	ShortMargineableValue                  string `json:"short-margineable-value"`
	SmaEquityOptionBuyingPower             string `json:"sma-equity-option-buying-power"`
	SpecialMemorandumAccountApexAdjustment string `json:"special-memorandum-account-apex-adjustment"`
	SpecialMemorandumAccountValue          string `json:"special-memorandum-account-value"`
	TotalSettleBalance                     string `json:"total-settle-balance"`
	UnsettledCryptocurrencyFiatAmount      string `json:"unsettled-cryptocurrency-fiat-amount"`
	UnsettledCryptocurrencyFiatEffect      string `json:"unsettled-cryptocurrency-fiat-effect"`
	UsedDerivativeBuyingPower              string `json:"used-derivative-buying-power"`
	SnapshotDate                           string `json:"snapshot-date"`
	RegTMarginRequirement                  string `json:"reg-t-margin-requirement"`
	FuturesOvernightMarginRequirement      string `json:"futures-overnight-margin-requirement"`
	FuturesIntradayMarginRequirement       string `json:"futures-intraday-margin-requirement"`
	MaintenanceExcess                      string `json:"maintenance-excess"`
	PendingMarginInterest                  string `json:"pending-margin-interest"`
	BuyingPowerAdjustment                  string `json:"buying-power-adjustment"`
	BuyingPowerAdjustmentEffect            string `json:"buying-power-adjustment-effect"`
	EffectiveCryptocurrencyBuyingPower     string `json:"effective-cryptocurrency-buying-power"`
	TotalPendingLiquidityPoolRebate        string `json:"total-pending-liquidity-pool-rebate"`
	LongIndexDerivativeValue               string `json:"long-index-derivative-value"`
	ShortIndexDerivativeValue              string `json:"short-index-derivative-value"`
	UpdatedAt                              string `json:"updated-at"`
}

type Symbol struct {
	Symbol          string `json:"symbol"`
	Description     string `json:"description"`
	ListedMarket    string `json:"listed-market"`
	PriceIncrements string `json:"price-increments"`
	TradingHours    string `json:"trading-hours"`
	AutoComplete    int    `json:"autocomplete"`
	Options         bool   `json:"options"`
	InstrumentType  string `json:"instrument-type"`
}

type GetAPIQuoteTokenOutput struct {
	Data    APIQuoteTokenData `json:"data"`
	Context string            `json:"context"`
}

type APIQuoteTokenData struct {
	DXLinkURL string `json:"dxlink-url"`
	ExpiresAt string `json:"expires-at"`
	IssuedAt  string `json:"issued-at"`
	Level     string `json:"level"`
	Token     string `json:"token"`
}

type PlaceEquityOrderInput struct {
	AccountID string
	Symbol    string
	Action    string // "Buy to Open" or "Sell to Close"
	Quantity  float64
}

type PlaceEquityOrderOutput struct {
	OrderID int
	Status  string
}

type LiveOrdersOutput struct {
	Data LiveOrdersData `json:"data"`
}

type LiveOrdersData struct {
	Items []LiveOrder `json:"items"`
}

type LiveOrder struct {
	ID     int            `json:"id"`
	Status string         `json:"status"`
	Legs   []LiveOrderLeg `json:"legs"`
}

// IsTerminal reports whether the order is in a terminal (no-further-update) state.
func (order LiveOrder) IsTerminal() bool {
	switch order.Status {
	case "Filled", "Cancelled", "Expired", "Rejected", "Removed", "Partially Removed":
		return true
	}
	return false
}

type LiveOrderLeg struct {
	Symbol string `json:"symbol"`
}

type ClientFactory interface {
	Create() Client
}
