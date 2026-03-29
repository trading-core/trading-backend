package fetchsentiment

import (
	"context"
	"time"

	"github.com/kduong/trading-backend/internal/config"
)

type Classification string

const (
	ClassificationExtremeFear  Classification = "Extreme Fear"
	ClassificationFear         Classification = "Fear"
	ClassificationNeutral      Classification = "Neutral"
	ClassificationGreed        Classification = "Greed"
	ClassificationExtremeGreed Classification = "Extreme Greed"
)

type Strategy interface {
	GetFearGreedIndex(ctx context.Context) (output *GetFearGreedIndexOutput, err error)
}

type GetFearGreedIndexOutput struct {
	Value          int             `json:"value"`
	Classification Classification  `json:"classification"`
	PreviousClose  *int            `json:"previous_close,omitempty"`
	Previous1Week  *int            `json:"previous_1_week,omitempty"`
	Previous1Month *int            `json:"previous_1_month,omitempty"`
	Previous1Year  *int            `json:"previous_1_year,omitempty"`
	Timeline       []TimelinePoint `json:"timeline,omitempty"`
	Source         string          `json:"source"`
	FetchedAt      string          `json:"fetched_at"`
}

type TimelinePoint struct {
	Timestamp int64  `json:"timestamp"`
	Value     int    `json:"value"`
	Rating    string `json:"rating"`
}

func StrategyFromEnv() Strategy {
	implementation := config.EnvStringOrFatal("FETCH_SENTIMENT_STRATEGY_IMPLEMENTATION")
	switch implementation {
	case "CNN_MARKET":
		return NewCNNMarketStrategy(NewCNNMarketStrategyInput{
			Timeout:           config.EnvDuration("FETCH_SENTIMENT_STRATEGY_CNN_MARKET_TIMEOUT", 10*time.Second),
			GraphDataEndpoint: config.EnvStringOrFatal("FETCH_SENTIMENT_STRATEGY_CNN_MARKET_ENDPOINT"),
		})
	default:
		panic("unknown fetch sentiment strategy implementation: " + implementation)
	}
}

func ClassifySentimentValue(value int) Classification {
	switch {
	case value <= 24:
		return ClassificationExtremeFear
	case value <= 44:
		return ClassificationFear
	case value <= 55:
		return ClassificationNeutral
	case value <= 75:
		return ClassificationGreed
	default:
		return ClassificationExtremeGreed
	}
}
