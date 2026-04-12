package replay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const replayLoadCacheVersion = 1

type cacheStrategyDecorator struct {
	base Strategy
}

type cacheIdentity struct {
	Version    int             `json:"version"`
	Source     string          `json:"source"`
	Symbol     string          `json:"symbol"`
	Timeframe  string          `json:"timeframe"`
	Start      string          `json:"start"`
	End        string          `json:"end"`
	WarmupBars int             `json:"warmupBars"`
	Alpaca     AlpacaInput     `json:"alpaca"`
	TastyTrade TastyTradeInput `json:"tastyTrade"`
}

func (decorator *cacheStrategyDecorator) Load(ctx context.Context, input LoadInput) (output *LoadOutput, err error) {
	cachePath, err := decorator.cachePathForInput(input)
	if err != nil {
		return
	}
	output, err = decorator.readLoadOutputCache(cachePath)
	if err != nil {
		output, err = decorator.base.Load(ctx, input)
		if err != nil {
			return
		}
		decorator.writeLoadOutputCache(cachePath, output)
	}
	return
}

func (decorator *cacheStrategyDecorator) cachePathForInput(input LoadInput) (string, error) {
	identity := cacheIdentity{
		Version:    replayLoadCacheVersion,
		Source:     strings.TrimSpace(strings.ToLower(input.Source)),
		Symbol:     strings.TrimSpace(strings.ToUpper(input.Symbol)),
		Timeframe:  strings.TrimSpace(input.Timeframe),
		Start:      strings.TrimSpace(input.Start),
		End:        strings.TrimSpace(input.End),
		WarmupBars: input.WarmupBars,
		Alpaca:     input.Alpaca,
		TastyTrade: input.TastyTrade,
	}
	payload, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	cacheKey := hex.EncodeToString(sum[:])
	cacheDir := strings.TrimSpace(input.CacheDir)
	if cacheDir == "" {
		cacheDir = "./tmp/cache"
	}
	return filepath.Join(cacheDir, fmt.Sprintf("load-%s.json", cacheKey)), nil
}

func (decorator *cacheStrategyDecorator) readLoadOutputCache(path string) (*LoadOutput, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var output LoadOutput
	if err := json.Unmarshal(bytes, &output); err != nil {
		return nil, err
	}
	if len(output.Prices) == 0 {
		return nil, fmt.Errorf("cache file %s has no prices", path)
	}
	return &output, nil
}

func (decorator *cacheStrategyDecorator) writeLoadOutputCache(path string, output *LoadOutput) error {
	if output == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	bytes, err := json.Marshal(output)
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, bytes, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
