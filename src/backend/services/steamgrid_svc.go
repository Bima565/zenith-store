package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"zenith-store/backend/config"
	"zenith-store/backend/models"
)

const (
	steamGridBaseURL = "https://www.steamgriddb.com/api/v2"
	itunesSearchURL  = "https://itunes.apple.com/search"
)

var gameSearchAliases = map[string]string{
	"mobile legends":               "Mobile Legends: Bang Bang",
	"free fire":                    "Garena Free Fire",
	"pubg mobile":                  "PUBG Mobile",
	"call of duty mobile":          "Call of Duty: Mobile",
	"ragnarok x":                   "Ragnarok X: Next Generation",
	"one-punch man":                "One Punch Man: The Strongest",
	"ea sports fc mobile":          "EA SPORTS FC Mobile",
	"the king of fighters allstar": "The King of Fighters ALLSTAR",
	"uma musume: pretty derby":     "Umamusume: Pretty Derby",
	"brown dust 2":                 "BrownDust2",
	"nier reincarnation":           "NieR Re[in]carnation",
	"asphalt legends":              "Asphalt Legends Unite",
	"grandchase":                   "GrandChase",
}

type steamGridAsset struct {
	GameID int
	Grid   string
	Icon   string
}

type SteamGridService struct {
	apiKey     string
	httpClient *http.Client

	mu    sync.RWMutex
	cache map[string]steamGridAsset
}

func NewSteamGridService(cfg *config.Config) *SteamGridService {
	if cfg.SteamGridAPIKey == "" {
		log.Printf("[steamgrid] WARNING: STEAMGRID API key is empty; grids will fall back to icons")
	} else {
		log.Printf("[steamgrid] API key loaded (len=%d)", len(cfg.SteamGridAPIKey))
	}

	dialer := &net.Dialer{Timeout: 12 * time.Second}
	return &SteamGridService{
		apiKey: cfg.SteamGridAPIKey,
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
					return dialer.DialContext(ctx, "tcp4", address)
				},
			},
		},
		cache: make(map[string]steamGridAsset),
	}
}

// EnrichServices resolves crisp square mobile icons for all games (Priority: existing mobile icon / valid square favicon -> mobile App Store icon -> SteamGridDB square icon)
// and portrait 600x900 SteamGridDB grids for the first maxGridGames popular games.
// Non-game / voucher entries are skipped so they keep their favicons untouched.
func (s *SteamGridService) EnrichServices(ctx context.Context, list []models.GameService, maxGridGames int) []models.GameService {
	if len(list) == 0 {
		return list
	}

	type targetGame struct {
		name        string
		cacheKey    string
		currentIcon string
		needGrid    bool
	}

	targets := make([]targetGame, 0, len(list))
	seen := make(map[string]struct{}, len(list))
	for _, svc := range list {
		if isNonGameService(svc) {
			continue
		}
		key := normalizeGameKey(svc.Game)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			needGrid := maxGridGames <= 0 || len(targets) < maxGridGames
			targets = append(targets, targetGame{
				name:        svc.Game,
				cacheKey:    key,
				currentIcon: svc.Icon,
				needGrid:    needGrid,
			})
		}
	}

	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	for _, t := range targets {
		s.mu.RLock()
		cached := s.cache[t.cacheKey]
		s.mu.RUnlock()

		hasIcon := cached.Icon != ""
		hasGrid := !t.needGrid || cached.Grid != "" || s.apiKey == ""
		if hasIcon && hasGrid {
			continue
		}

		wg.Add(1)
		go func(tg targetGame, prev steamGridAsset) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			updated := s.resolveGameAssets(ctx, tg.name, tg.currentIcon, prev, tg.needGrid)
			if updated.GameID != 0 || updated.Icon != "" || updated.Grid != "" {
				s.mu.Lock()
				s.cache[tg.cacheKey] = updated
				s.mu.Unlock()
			}
		}(t, cached)
	}
	wg.Wait()

	enriched := make([]models.GameService, len(list))
	gridHits, iconHits := 0, 0
	s.mu.RLock()
	for i, svc := range list {
		if !isNonGameService(svc) {
			if asset, ok := s.cache[normalizeGameKey(svc.Game)]; ok {
				if asset.Icon != "" {
					svc.Icon = asset.Icon
					iconHits++
				}
				if asset.Grid != "" {
					svc.Grid = asset.Grid
					gridHits++
				}
			}
		}
		enriched[i] = svc
	}
	s.mu.RUnlock()

	log.Printf("[steamgrid] resolved icons=%d/%d, popular_grids=%d/%d", iconHits, len(targets), gridHits, minInt(maxGridGames, len(targets)))
	return enriched
}

func (s *SteamGridService) resolveGameAssets(ctx context.Context, gameName, currentIcon string, prev steamGridAsset, needGrid bool) steamGridAsset {
	asset := prev

	// 1. Resolve icon: prioritize official mobile app icon / crisp non-round square favicon -> App Store mobile icon -> SteamGridDB square fallback.
	if asset.Icon == "" {
		if isTrustedMobileIconURL(currentIcon) {
			// Already a crisp 512x512 official mobile app icon from App Store.
			asset.Icon = currentIcon
		} else if currentIcon != "" && s.isCleanSquareImage(ctx, currentIcon) {
			// Favicon exists, is sharp (>=64x64), square, and solid (not circular/cutout).
			asset.Icon = currentIcon
		} else if mobileIcon := s.fetchMobileAppIcon(ctx, gameName); mobileIcon != "" {
			// Official mobile game app icon from App Store (always 512x512 square).
			asset.Icon = mobileIcon
		}
	}

	// Only hit SteamGridDB if we need a popular portrait grid OR we still need a square icon fallback.
	needSteamGrid := s.apiKey != "" && ((needGrid && asset.Grid == "") || asset.Icon == "")
	if !needSteamGrid {
		return asset
	}

	if asset.GameID == 0 {
		query := gameName
		if alias, ok := gameSearchAliases[normalizeGameKey(gameName)]; ok {
			query = alias
		}

		gameID, err := s.searchGameID(ctx, query)
		if err != nil {
			log.Printf("[steamgrid] search %q error: %v", query, err)
		}
		if gameID == 0 && query != gameName {
			gameID, err = s.searchGameID(ctx, gameName)
			if err != nil {
				log.Printf("[steamgrid] search fallback %q error: %v", gameName, err)
			}
		}
		if gameID == 0 {
			log.Printf("[steamgrid] game not found: %q", gameName)
			return asset
		}
		asset.GameID = gameID
	}

	// 2. Popular slider portrait grid (600x900).
	if needGrid && asset.Grid == "" {
		queries := []string{
			"dimensions=600x900&styles=alternate&types=static&nsfw=false&humor=false&epilepsy=false",
			"dimensions=600x900&types=static&nsfw=false&humor=false&epilepsy=false",
			"dimensions=660x930,342x482&types=static&nsfw=false&humor=false&epilepsy=false",
			"types=static&nsfw=false&humor=false&epilepsy=false",
		}
		for _, q := range queries {
			if gridURL := s.fetchBestGrid(ctx, asset.GameID, q); gridURL != "" {
				asset.Grid = gridURL
				break
			}
		}
	}

	// 3. SteamGridDB square icon fallback (only when favicon was missing/blurry/round AND no mobile App Store icon found).
	if asset.Icon == "" {
		// Prefer 512x512 / 1024x1024 square grids first (guaranteed full square, never round).
		asset.Icon = s.fetchBestGrid(ctx, asset.GameID, "dimensions=512x512,1024x1024&types=static&nsfw=false&humor=false&epilepsy=false")
		if asset.Icon == "" {
			// Fallback to official SteamGridDB icon only if verified square & non-round.
			asset.Icon = s.fetchVerifiedSquareOfficialIcon(ctx, asset.GameID)
		}
		if asset.Icon == "" {
			if asset.Grid != "" {
				asset.Icon = asset.Grid
			} else {
				asset.Icon = s.fetchBestGrid(ctx, asset.GameID, "dimensions=600x900&types=static&nsfw=false&humor=false&epilepsy=false")
			}
		}
	}

	return asset
}

func isTrustedMobileIconURL(rawURL string) bool {
	return strings.Contains(rawURL, "mzstatic.com/")
}

// isCleanSquareImage validates that an image URL returns a crisp (>=64x64), square (1:1),
// non-circular (solid corners) image suitable as a square mobile game icon.
func (s *SteamGridService) isCleanSquareImage(ctx context.Context, imgURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ZenithStore/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil || len(data) < 1800 {
		// Too small — usually a generic placeholder or low-res 16x16 favicon.
		return false
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return false
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 64 || h < 64 || w != h {
		return false
	}

	// Reject circular/cutout icons by checking whether all 4 corners are transparent.
	corners := [4][2]int{
		{bounds.Min.X, bounds.Min.Y},
		{bounds.Max.X - 1, bounds.Min.Y},
		{bounds.Min.X, bounds.Max.Y - 1},
		{bounds.Max.X - 1, bounds.Max.Y - 1},
	}
	transparentCorners := 0
	for _, pt := range corners {
		_, _, _, a := img.At(pt[0], pt[1]).RGBA()
		if a < 40000 {
			transparentCorners++
		}
	}
	return transparentCorners < 4
}

// fetchMobileAppIcon queries the public iTunes App Store Search API for an official 512x512 square mobile game icon.
func (s *SteamGridService) fetchMobileAppIcon(ctx context.Context, gameName string) string {
	query := gameName
	if alias, ok := gameSearchAliases[normalizeGameKey(gameName)]; ok {
		query = alias
	}

	for _, country := range []string{"us", "jp"} {
		endpoint := fmt.Sprintf("%s?term=%s&entity=software&limit=5&country=%s", itunesSearchURL, url.QueryEscape(query), country)
		var resp struct {
			Results []struct {
				TrackName     string `json:"trackName"`
				ArtworkURL512 string `json:"artworkUrl512"`
				ArtworkURL100 string `json:"artworkUrl100"`
			} `json:"results"`
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ZenithStore/1.0")
		httpResp, err := s.httpClient.Do(req)
		if err != nil {
			continue
		}
		if httpResp.StatusCode == http.StatusOK {
			_ = json.NewDecoder(httpResp.Body).Decode(&resp)
		}
		httpResp.Body.Close()

		for _, item := range resp.Results {
			if !matchesGameTitle(gameName, item.TrackName) {
				continue
			}
			if item.ArtworkURL512 != "" {
				return item.ArtworkURL512
			}
			if item.ArtworkURL100 != "" {
				return strings.Replace(item.ArtworkURL100, "100x100bb", "512x512bb", 1)
			}
		}
	}
	return ""
}

func matchesGameTitle(expected, actual string) bool {
	normExpected := cleanTitleTokens(expected)
	normActual := cleanTitleTokens(actual)
	if normExpected == "" || normActual == "" {
		return false
	}
	if strings.Contains(normActual, normExpected) || strings.Contains(normExpected, normActual) {
		return true
	}
	tokens := strings.Fields(normExpected)
	if len(tokens) == 0 {
		return false
	}
	matched := 0
	for _, tok := range tokens {
		if len(tok) <= 1 {
			continue
		}
		if strings.Contains(normActual, tok) {
			matched++
		}
	}
	return matched*2 >= len(tokens)
}

func cleanTitleTokens(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func (s *SteamGridService) searchGameID(ctx context.Context, term string) (int, error) {
	endpoint := fmt.Sprintf("%s/search/autocomplete/%s", steamGridBaseURL, url.PathEscape(term))
	var resp struct {
		Success bool `json:"success"`
		Data    []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := s.doGet(ctx, endpoint, &resp); err != nil {
		return 0, err
	}
	if !resp.Success || len(resp.Data) == 0 {
		return 0, nil
	}

	lowerTerm := strings.ToLower(strings.TrimSpace(term))
	for _, item := range resp.Data {
		if strings.ToLower(strings.TrimSpace(item.Name)) == lowerTerm {
			return item.ID, nil
		}
	}
	return resp.Data[0].ID, nil
}

func (s *SteamGridService) fetchVerifiedSquareOfficialIcon(ctx context.Context, gameID int) string {
	endpoint := fmt.Sprintf("%s/icons/game/%d?styles=official&nsfw=false&humor=false&epilepsy=false", steamGridBaseURL, gameID)
	var resp struct {
		Success bool `json:"success"`
		Data    []struct {
			URL    string `json:"url"`
			Thumb  string `json:"thumb"`
			Mime   string `json:"mime"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"data"`
	}
	if err := s.doGet(ctx, endpoint, &resp); err != nil {
		log.Printf("[steamgrid] official icons game=%d error: %v", gameID, err)
		return ""
	}
	if !resp.Success || len(resp.Data) == 0 {
		return ""
	}

	for _, item := range resp.Data {
		if item.Width > 0 && item.Height > 0 && item.Width != item.Height {
			continue
		}
		candidate := ""
		if strings.EqualFold(item.Mime, "image/png") && item.URL != "" {
			candidate = item.URL
		} else if item.Thumb != "" {
			candidate = item.Thumb
		} else if item.URL != "" {
			candidate = item.URL
		}
		if candidate != "" && s.isCleanSquareImage(ctx, candidate) {
			return candidate
		}
	}
	return ""
}

func (s *SteamGridService) fetchBestGrid(ctx context.Context, gameID int, query string) string {
	endpoint := fmt.Sprintf("%s/grids/game/%d?%s", steamGridBaseURL, gameID, query)
	var resp struct {
		Success bool `json:"success"`
		Data    []struct {
			URL   string `json:"url"`
			Thumb string `json:"thumb"`
		} `json:"data"`
	}
	if err := s.doGet(ctx, endpoint, &resp); err != nil {
		log.Printf("[steamgrid] grids game=%d error: %v", gameID, err)
		return ""
	}
	if !resp.Success || len(resp.Data) == 0 {
		return ""
	}
	if resp.Data[0].URL != "" {
		return resp.Data[0].URL
	}
	return resp.Data[0].Thumb
}

func (s *SteamGridService) doGet(ctx context.Context, endpoint string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ZenithStore/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func normalizeGameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

var nonGameNames = map[string]struct{}{
	"steam":       {},
	"playstation": {},
	"xbox":        {},
	"google play": {},
	"epic games":  {},
}

func isNonGameService(svc models.GameService) bool {
	if strings.EqualFold(svc.Category, "voucher") || strings.EqualFold(svc.Category, "non-game") {
		return true
	}
	_, ok := nonGameNames[normalizeGameKey(svc.Game)]
	return ok
}

func minInt(a, b int) int {
	if a <= 0 || b < a {
		return b
	}
	return a
}
