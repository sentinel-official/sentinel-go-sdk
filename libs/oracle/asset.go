package oracle

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/sentinel-official/sentinelhub/v12/x/oracle/types/v1"
)

// Embeds all JSON files in the package for runtime access.
//
//go:embed *.json
var fs embed.FS

// Asset extends v1.Asset with additional metadata.
type Asset struct {
	*v1.Asset

	CoinGeckoID string `json:"coingecko_id"` // CoinGeckoID is the token's identifier on CoinGecko,
}

// LoadAssets reads embedded asset definitions from assets.json
// and returns them as a map keyed by denomination.
func LoadAssets() (map[string]Asset, error) {
	// Read embedded JSON file.
	buf, err := fs.ReadFile("assets.json")
	if err != nil {
		return nil, fmt.Errorf("reading assets.json: %w", err)
	}

	// Decode JSON into a slice of Asset.
	var items []Asset
	if err := json.Unmarshal(buf, &items); err != nil {
		return nil, fmt.Errorf("unmarshaling assets.json: %w", err)
	}

	// Build map for quick lookup by denom.
	assets := make(map[string]Asset, len(items))
	for _, item := range items {
		assets[item.Denom] = item
	}

	return assets, nil
}
