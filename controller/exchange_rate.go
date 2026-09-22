/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package controller

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

// USD→CNY central parity rate is published by 中国货币网 (China Foreign Exchange
// Trade System, authorised by PBOC). The public webpage at
// https://www.chinamoney.com.cn/chinese/bkccpr/ is powered by the JSON feed
// below; it does not send CORS headers so browsers cannot fetch it directly.
// This admin-only endpoint proxies it and caches for a few minutes.
const (
	chinaMoneyCCPRURL = "https://www.chinamoney.com.cn/r/cms/www/chinamoney/data/fx/ccpr.json"
	chinaMoneyRefURL  = "https://www.chinamoney.com.cn/chinese/bkccpr/"
	exchangeRateCache = 5 * time.Minute
	exchangeRateHTTP  = 8 * time.Second
)

type usdCnyRate struct {
	Rate   float64 `json:"rate"`
	Base   string  `json:"base"`
	Quote  string  `json:"quote"`
	AsOf   string  `json:"as_of"`
	Source string  `json:"source"`
}

type usdCnyCache struct {
	mu    sync.Mutex
	value *usdCnyRate
	at    time.Time
}

var usdCnyCacheStore usdCnyCache

// ccprPayload models the subset of the chinamoney.com.cn ccpr.json response
// that we need. Only the first record (USD/CNY) is consumed.
type ccprPayload struct {
	Data struct {
		LastDate string `json:"lastDate"`
	} `json:"data"`
	Records []struct {
		VrtEName     string `json:"vrtEName"`
		ForeignCName string `json:"foreignCName"`
		Price        string `json:"price"`
	} `json:"records"`
}

func fetchUsdCnyFromChinaMoney(ctx context.Context) (*usdCnyRate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, chinaMoneyCCPRURL, nil)
	if err != nil {
		return nil, err
	}
	// The upstream CDN rejects requests without a browser-like UA and Referer.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; new-api/exchange-rate)")
	req.Header.Set("Referer", chinaMoneyRefURL)
	req.Header.Set("Accept", "application/json,text/plain,*/*")

	client := &http.Client{Timeout: exchangeRateHTTP}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("upstream returned status " + strconv.Itoa(resp.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, err
	}

	var payload ccprPayload
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	for _, rec := range payload.Records {
		// The USD row is normally the first record; scan defensively.
		if strings.EqualFold(rec.ForeignCName, "USD") || strings.HasPrefix(strings.ToUpper(rec.VrtEName), "USD/") {
			price, err := strconv.ParseFloat(strings.TrimSpace(rec.Price), 64)
			if err != nil || price <= 0 {
				return nil, errors.New("invalid USD price in upstream payload")
			}
			return &usdCnyRate{
				Rate:   price,
				Base:   "USD",
				Quote:  "CNY",
				AsOf:   payload.Data.LastDate,
				Source: "chinamoney.com.cn",
			}, nil
		}
	}
	return nil, errors.New("USD/CNY row missing in upstream payload")
}

// GetUsdCnyExchangeRate returns the latest CNY-per-USD central parity rate for
// use as a reference in the admin billing settings page. Results are cached
// for a few minutes to avoid hammering the upstream.
func GetUsdCnyExchangeRate(c *gin.Context) {
	usdCnyCacheStore.mu.Lock()
	if usdCnyCacheStore.value != nil && time.Since(usdCnyCacheStore.at) < exchangeRateCache {
		cached := *usdCnyCacheStore.value
		usdCnyCacheStore.mu.Unlock()
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": cached})
		return
	}
	usdCnyCacheStore.mu.Unlock()

	ctx, cancel := context.WithTimeout(c.Request.Context(), exchangeRateHTTP+2*time.Second)
	defer cancel()

	rate, err := fetchUsdCnyFromChinaMoney(ctx)
	if err != nil {
		// Serve stale value if we still have one, so a transient upstream
		// hiccup does not blank the reference panel.
		usdCnyCacheStore.mu.Lock()
		if usdCnyCacheStore.value != nil {
			cached := *usdCnyCacheStore.value
			usdCnyCacheStore.mu.Unlock()
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "",
				"data":    cached,
				"stale":   true,
			})
			return
		}
		usdCnyCacheStore.mu.Unlock()
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	usdCnyCacheStore.mu.Lock()
	usdCnyCacheStore.value = rate
	usdCnyCacheStore.at = time.Now()
	usdCnyCacheStore.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": rate})
}
