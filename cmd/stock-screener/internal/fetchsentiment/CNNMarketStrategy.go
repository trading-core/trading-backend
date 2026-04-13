package fetchsentiment

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/kduong/trading-backend/internal/httpx"
)

type CNNMarketStrategy struct {
	httpClient        *http.Client
	graphDataEndpoint string
}

type NewCNNMarketStrategyInput struct {
	Timeout           time.Duration
	GraphDataEndpoint string
}

func NewCNNMarketStrategy(input NewCNNMarketStrategyInput) *CNNMarketStrategy {
	return &CNNMarketStrategy{
		httpClient: &http.Client{
			Timeout: input.Timeout,
		},
		graphDataEndpoint: input.GraphDataEndpoint,
	}
}

func (strategy *CNNMarketStrategy) GetFearGreedIndex(ctx context.Context) (output *GetFearGreedIndexOutput, err error) {
	graphData, err := strategy.getGraphData(ctx)
	if err != nil {
		return
	}
	value := int(math.Round(graphData.FearGreed.Score))
	output = &GetFearGreedIndexOutput{
		Value:          value,
		Classification: ClassifySentimentValue(value),
		PreviousClose:  roundOptional(graphData.FearGreed.PreviousClose),
		Previous1Week:  roundOptional(graphData.FearGreed.Previous1Week),
		Previous1Month: roundOptional(graphData.FearGreed.Previous1Month),
		Previous1Year:  roundOptional(graphData.FearGreed.Previous1Year),
		Timeline:       buildTimeline(graphData.FearGreedHistorical.Data),
		Source:         "CNN",
		FetchedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	return
}

type graphData struct {
	FearGreed           fearGreed           `json:"fear_and_greed"`
	FearGreedHistorical fearGreedHistorical `json:"fear_and_greed_historical"`
}

type fearGreed struct {
	Score          float64  `json:"score"`
	PreviousClose  *float64 `json:"previous_close"`
	Previous1Week  *float64 `json:"previous_1_week"`
	Previous1Month *float64 `json:"previous_1_month"`
	Previous1Year  *float64 `json:"previous_1_year"`
}

type fearGreedHistorical struct {
	Data []dataPoint `json:"data"`
}

type dataPoint struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Rating string  `json:"rating"`
}

func (strategy *CNNMarketStrategy) getGraphData(ctx context.Context) (output *graphData, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strategy.graphDataEndpoint, nil)
	if err != nil {
		return
	}
	strategy.setBrowserHeaders(request)
	response, err := strategy.httpClient.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err = httpx.ExtractResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

// CNN blocks bot-like requests with HTTP 418 unless common browser headers are present.
func (strategy *CNNMarketStrategy) setBrowserHeaders(request *http.Request) {
	request.Header.Set("Accept", "application/json, text/plain, */*")
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("Pragma", "no-cache")
	request.Header.Set("Origin", "https://www.cnn.com")
	request.Header.Set("Referer", "https://www.cnn.com/markets/fear-and-greed")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
}

func buildTimeline(data []dataPoint) []TimelinePoint {
	if len(data) == 0 {
		return nil
	}
	const oneYearInMilliseconds = int64(365 * 24 * 60 * 60 * 1000)
	latestTimestamp := int64(data[len(data)-1].X)
	cutoffTimestamp := latestTimestamp - oneYearInMilliseconds
	points := make([]TimelinePoint, 0, len(data))
	for _, item := range data {
		timestamp := int64(item.X)
		if timestamp < cutoffTimestamp {
			continue
		}
		value := int(math.Round(item.Y))
		if value < 0 || value > 100 {
			continue
		}
		points = append(points, TimelinePoint{
			Timestamp: timestamp,
			Value:     value,
			Rating:    item.Rating,
		})
	}
	return points
}

func roundOptional(f *float64) *int {
	if f == nil {
		return nil
	}
	v := int(math.Round(*f))
	if v < 0 || v > 100 {
		return nil
	}
	return &v
}
