package services

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"zenith-store/backend/config"
	"zenith-store/backend/models"
)

type VIPaymentService struct {
	config     *config.Config
	httpClient *http.Client
}

func NewVIPaymentService(cfg *config.Config) *VIPaymentService {
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	return &VIPaymentService{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
					return dialer.DialContext(ctx, "tcp4", address)
				},
			},
		},
	}
}

func (s *VIPaymentService) GenerateSign() string {
	sum := md5.Sum([]byte(s.config.VIPaymentAPIID + s.config.VIPaymentAPIKey))
	return hex.EncodeToString(sum[:])
}

func (s *VIPaymentService) GetServices(ctx context.Context) ([]models.GameService, error) {
	if s.demoMode() {
		return demoServices(), nil
	}

	var response struct {
		Result  bool                 `json:"result"`
		Message string               `json:"message"`
		Data    []models.GameService `json:"data"`
	}
	params := url.Values{"type": {"services"}}
	if err := s.request(ctx, params, &response); err != nil {
		return nil, err
	}
	if !response.Result {
		return nil, fmt.Errorf("VIPayment services: %s", response.Message)
	}
	return response.Data, nil
}

func (s *VIPaymentService) GetNickname(ctx context.Context, gameCode, userID, zoneID string) (*models.NicknameResponse, error) {
	if s.demoMode() {
		return &models.NicknameResponse{Nickname: "Demo User"}, nil
	}

	var response struct {
		Result  bool            `json:"result"`
		Message string          `json:"message"`
		Data    string          `json:"data"`
		Country *models.Country `json:"country,omitempty"`
	}
	params := url.Values{
		"type":              {"get-nickname"},
		"code":              {gameCode},
		"target":            {userID},
		"additional_target": {zoneID},
	}
	if err := s.request(ctx, params, &response); err != nil {
		return nil, err
	}
	if !response.Result {
		return nil, fmt.Errorf("VIPayment nickname: %s", response.Message)
	}
	return &models.NicknameResponse{Nickname: response.Data, Country: response.Country}, nil
}

func (s *VIPaymentService) CreateTransaction(ctx context.Context, order models.Order) (*models.Transaction, error) {
	if s.demoMode() {
		return &models.Transaction{
			TrxID:  order.ReferenceID,
			Data:   order.UserID,
			Zone:   order.ZoneID,
			Status: "waiting",
		}, nil
	}

	var response struct {
		Result  bool               `json:"result"`
		Message string             `json:"message"`
		Data    models.Transaction `json:"data"`
	}
	params := url.Values{
		"type":                 {"order"},
		"service":              {order.ServiceCode},
		"data_no":              {order.UserID},
		"data_zone":            {order.ZoneID},
		"post_additional_data": {order.AdditionalData},
	}
	if err := s.request(ctx, params, &response); err != nil {
		return nil, err
	}
	if !response.Result {
		return nil, fmt.Errorf("VIPayment order: %s", response.Message)
	}
	return &response.Data, nil
}

func (s *VIPaymentService) GetTransactionStatus(ctx context.Context, trxID string) ([]models.Transaction, error) {
	if s.demoMode() {
		return []models.Transaction{{TrxID: trxID, Status: "waiting"}}, nil
	}

	var response struct {
		Result  bool                 `json:"result"`
		Message string               `json:"message"`
		Data    []models.Transaction `json:"data"`
	}
	params := url.Values{"type": {"status"}, "trxid": {trxID}}
	if err := s.request(ctx, params, &response); err != nil {
		return nil, err
	}
	if !response.Result {
		return nil, fmt.Errorf("VIPayment status: %s", response.Message)
	}
	return response.Data, nil
}

func (s *VIPaymentService) ValidateWebhookSignature(signature string) bool {
	expected := s.GenerateSign()
	provided, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil || len(provided) != len(expected)/2 {
		return false
	}
	expectedBytes, _ := hex.DecodeString(expected)
	return subtle.ConstantTimeCompare(provided, expectedBytes) == 1
}

func (s *VIPaymentService) request(ctx context.Context, params url.Values, target interface{}) error {
	params.Set("key", s.config.VIPaymentAPIKey)
	params.Set("sign", s.GenerateSign())
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.config.VIPaymentBaseURL+s.config.VIPaymentServicesPath,
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("VIPayment request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("VIPayment returned status %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode VIPayment response: %w", err)
	}
	return nil
}

func (s *VIPaymentService) demoMode() bool {
	return true
}

func demoServices() []models.GameService {
	return []models.GameService{
		demoArtwork("MLBB-DIAMOND", "Mobile Legends", "Diamond Mobile Legends", "Isi ulang diamond Mobile Legends", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/11/2c/17/112c17b5-3220-e5c5-85a5-437ee5a14b6c/AppIcon-0-0-1x_U007emarketing-0-7-0-85-220.png/512x512bb.jpg", 10000),
		demoArtwork("FF-DIAMOND", "Free Fire", "Diamond Free Fire", "Isi ulang diamond Free Fire", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/ef/16/6b/ef166bba-1d69-85f5-ed84-6b7c6e73b55c/AppIcon-1788350393-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 12000),
		demoArtwork("PUBG-UC", "PUBG Mobile", "Unknown Cash", "Isi ulang UC PUBG Mobile", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/0c/1a/dc/0c1adcc0-b63f-6bdb-3cba-d74161aad06a/AppIcon-0-0-1x_U007emarketing-0-7-0-85-220.png/512x512bb.jpg", 15000),
		demoArtwork("CODM-CP", "Call of Duty Mobile", "COD Points", "Isi ulang COD Points", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/2d/4c/30/2d4c300d-738d-a754-60aa-5e75e22f1e1a/AppIcon-0-0-1x_U007emarketing-0-10-0-85-220.png/512x512bb.jpg", 14500),
		demoArtwork("GENSHIN-CRYSTAL", "Genshin Impact", "Genesis Crystal", "Isi ulang Genesis Crystal", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/15/9e/b1/159eb1a7-fc58-2b43-b330-c5e619a77e2d/AppIcon-0-0-1x_U007epad-0-1-85-220.png/512x512bb.jpg", 16000),
		demoArtwork("HSR-JADE", "Honkai: Star Rail", "Oneiric Shard", "Isi ulang Oneiric Shard", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/b9/19/0a/b9190ac1-9dfc-6d88-5ac4-96dd4e6d6ff4/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 17000),
		demoArtwork("ZZZ-POLAR", "Zenless Zone Zero", "Polychrome", "Isi ulang Polychrome", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/c8/ee/27/c8ee2769-fe15-e1ea-aeb7-2247a535767a/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 16500),
		demoArtwork("HI3-CRYSTAL", "Honkai Impact 3rd", "Honkai Crystal", "Isi ulang crystal Honkai Impact 3rd", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/20/4b/e7/204be75d-5108-d55b-eac2-3dac080e7f3e/AppIcon-1x_U007emarketing-0-10-0-85-220-0.png/512x512bb.jpg", 13500),
		demoArtwork("WILD-RIFT", "League of Legends: Wild Rift", "Wild Core", "Isi ulang Wild Core", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/4c/10/e5/4c10e5d1-2fbb-48f2-0851-49d5c0b30ee9/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 14000),
		demoArtwork("CLASH-ROYALE", "Clash Royale", "Gems", "Isi ulang gems Clash Royale", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/82/9e/a7/829ea73c-c333-1953-f5fb-c16ea2ed80e3/AppIcon-0-0-1x_U007emarketing-0-7-0-85-220.png/512x512bb.jpg", 12500),
		demoArtwork("CLASH-CLANS", "Clash of Clans", "Gems", "Isi ulang gems Clash of Clans", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/01/a1/b9/01a1b9e9-7dd8-8978-8379-0f08843ae829/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 13000),
		demoArtwork("BRAWL-STARS", "Brawl Stars", "Gems", "Isi ulang gems Brawl Stars", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/b3/65/2d/b3652d49-c00c-7f1f-f63e-941eb32ffeef/AppIcon-0-0-1x_U007epad-0-1-85-220.png/512x512bb.jpg", 12000),
		demoArtwork("ROBLOX-CREDIT", "Roblox", "Robux", "Isi ulang Robux Roblox", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/05/b8/b8/05b8b8cc-1410-a32b-eb2c-3e24acd99189/AppIcon-0-0-1x_U007epad-0-1-0-85-220.png/512x512bb.jpg", 13000),
		demoArtwork("STUMBLE-GUYS", "Stumble Guys", "Gems", "Isi ulang gems Stumble Guys", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/12/37/b9/1237b998-08b7-605f-f41d-f336b33e712f/AppIcon-0-0-1x_U007emarketing-0-5-0-85-220.png/512x512bb.jpg", 11500),
		demoArtwork("EFOOTBALL-COIN", "eFootball", "Coin", "Isi ulang coin eFootball", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/f2/98/c4/f298c428-1e55-a238-3c3c-a2c59b50523a/PesMobile-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 11000),
		demoArtwork("FC-POINT", "EA SPORTS FC Mobile", "FC Points", "Isi ulang FC Points", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/41/d1/84/41d18457-ec03-6812-f0a7-8a83686c6009/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 15500),
		demoArtwork("RAGNAROK-X", "Ragnarok X", "Diamond", "Isi ulang diamond Ragnarok X", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/45/89/a7/4589a73c-b346-6fec-9430-c1bac5df350c/AppIcon-1x_U007emarketing-0-8-0-85-220-0.png/512x512bb.jpg", 12500),
		demoArtwork("TOF-CRYSTAL", "Tower of Fantasy", "Tanium", "Isi ulang Tanium", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/20/d6/9a/20d69a3e-897b-32a9-85c5-fab781a6f1ee/AppIcon-1x_U007emarketing-0-8-0-85-220-0.png/512x512bb.jpg", 15000),
		demoArtwork("DRAGON-CITY", "Dragon City", "Gems", "Isi ulang gems Dragon City", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/21/81/16/21811675-76dc-6ae8-9653-00b4b8d49042/AppIcon-0-0-1x_U007emarketing-0-7-0-85-220.png/512x512bb.jpg", 12500),
		demoArtwork("LORDS-MOBILE", "Lords Mobile", "Gems", "Isi ulang gems Lords Mobile", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/da/cf/ec/dacfec37-c2d2-a59c-30ad-b4d2835ea3a1/AppIcon-1x_U007emarketing-0-8-0-85-220-0.png/512x512bb.jpg", 14500),
		demoArtwork("AFK-ARENA", "AFK Arena", "Diamonds", "Isi ulang diamonds AFK Arena", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/8d/36/c5/8d36c5b4-8359-0001-c0a1-16c16609c49f/AppIcon-0-0-1x_U007emarketing-0-11-0-85-220.png/512x512bb.jpg", 12000),
		demoArtwork("COOKIE-RUN", "Cookie Run: Kingdom", "Rainbow Cube", "Isi ulang Rainbow Cube", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/07/10/31/071031fb-8fbc-917f-60d0-9327d1618d85/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 15000),
		demoArtwork("FGO-SQ", "Fate/Grand Order", "Saint Quartz", "Isi ulang Saint Quartz", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/f1/9b/ed/f19bed9e-cbe9-775f-cc98-ea308814f8f0/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 18000),
		demoArtwork("KOF-ALLSTAR", "The King of Fighters ALLSTAR", "Gem", "Isi ulang gem KOF ALLSTAR", "https://is1-ssl.mzstatic.com/image/thumb/Purple3/v4/6f/b9/0c/6fb90c63-896b-22fe-f77d-e267751bbd0f/pr_source.png/512x512bb.jpg", 13000),
		demoArtwork("MLB-26", "MLB 9 Innings", "Star", "Isi ulang star MLB 9 Innings", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/e4/2c/96/e42c96f8-6bbb-c856-5deb-0f10e89c7a1d/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 14000),
		demoArtwork("STEAM-WALLET", "Steam", "Steam Wallet", "Tambahkan saldo Steam Wallet", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/8b/f1/60/8bf16027-4f91-3fdc-5bd3-e2cdbdce528f/AppIcon-0-0-1x_U007emarketing-0-5-85-220.png/512x512bb.jpg", 20000),
		demoArtwork("PSN-WALLET", "PlayStation", "PSN Wallet", "Tambahkan saldo PSN", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/7a/47/a6/7a47a62b-e11d-e11a-62b0-529d38ba42b3/PlayStation-0-0-1x_U007epad-0-1-0-sRGB-85-220.png/512x512bb.jpg", 23000),
		demoArtwork("XBOX-WALLET", "Xbox", "Xbox Gift Card", "Tambahkan saldo Xbox", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/33/1f/f3/331ff31b-ec3d-9215-a53d-406f451393ac/AppIcon-1x_U007epad-0-1-85-220-0.png/512x512bb.jpg", 22000),
		demoService("GOOGLE-PLAY", "Google Play", "Google Play Gift Card", "Tambahkan saldo Google Play", "play.google.com", 25000),
		demoArtwork("EPIC-WALLET", "Epic Games", "Epic Wallet", "Tambahkan saldo Epic Games", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/5b/63/2b/5b632bca-c114-bf8d-cc8d-27db59550a3b/AppIcon-Companion-0-0-1x_U007emarketing-0-11-0-85-220.png/512x512bb.jpg", 24000),
		demoArtwork("POKEMON-GO", "Pokémon GO", "PokéCoins", "Isi ulang PokéCoins", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/d6/7b/eb/d67beb51-9b15-fa10-ee24-7d600eaab720/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 12000),
		demoArtwork("UNDAWN", "Undawn", "RC", "Isi ulang RC Undawn", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/2f/65/5f/2f655f20-bce6-6f29-b76e-ce36ca9f9c00/AppIcon-1x_U007emarketing-0-8-0-85-220-0.png/512x512bb.jpg", 14000),
		demoArtwork("MU-ORIGIN-3", "MU ORIGIN 3", "Diamond", "Isi ulang diamond MU ORIGIN 3", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/8c/32/a2/8c32a25c-637a-0e40-5217-a493c7d4011e/AppIcon-1x_U007emarketing-0-8-0-85-220-0.png/512x512bb.jpg", 15000),
		demoArtwork("ONE-PUNCH-MAN", "One-Punch Man", "Gem", "Isi ulang gem One-Punch Man", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/89/ff/99/89ff99c2-3979-dd2a-7746-c65b7c8c4ead/AppIcon-1x_U007emarketing-0-11-0-85-220-0.png/512x512bb.jpg", 13000),
		demoService("RAID-SHADOW", "RAID: Shadow Legends", "Gem", "Isi ulang gem RAID: Shadow Legends", "raidshadowlegends.com", 16000),
		demoArtwork("ASPHALT-LEGENDS", "Asphalt Legends", "Token", "Isi ulang token Asphalt Legends", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/b0/5d/4c/b05d4cdb-f259-f7cf-6be3-77c4d0d5a9fb/AppIcon-0-0-1x_U007epad-0-1-85-220.png/512x512bb.jpg", 14500),
		demoArtwork("HAY-DAY", "Hay Day", "Diamond", "Isi ulang diamond Hay Day", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/0e/50/bc/0e50bce7-2b3a-3964-832d-9951131a2746/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 13500),
		demoArtwork("DIABLO-IMMORTAL", "Diablo Immortal", "Eternal Orb", "Isi ulang Eternal Orb", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/93/61/2a/93612a05-ae56-5e4f-74db-9387b334f518/AppIcon-0-0-1x_U007emarketing-0-11-0-85-220.png/512x512bb.jpg", 18000),
		demoArtwork("IDENTITY-V", "Identity V", "Echo", "Isi ulang Echo Identity V", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/cd/1c/78/cd1c78b5-de24-734e-749a-507f872745a7/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 14500),
		demoArtwork("BLACK-DESERT", "Black Desert Mobile", "Pearl", "Isi ulang Pearl Black Desert", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/78/34/a0/7834a0a9-ed00-de6f-702e-c0085a4fa751/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 17500),
		demoArtwork("DRAGON-BALL", "DRAGON BALL LEGENDS", "Chrono Crystal", "Isi ulang Chrono Crystal", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/cf/d8/23/cfd82328-b684-41ae-2a49-9910f12c732a/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 15500),
		demoArtwork("GUNDAM-UC", "Mobile Suit Gundam U.C. Engage", "Diamond", "Isi ulang diamond Gundam UC", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/7e/94/b9/7e94b9b4-7c5b-25ea-dd87-f2c177971278/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 17000),
		demoArtwork("MECH-ARENA", "Mech Arena", "A-Coins", "Isi ulang A-Coins Mech Arena", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/be/95/0f/be950f9b-b7c6-2455-17c1-50aad6a92aed/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 12500),
		demoArtwork("MARVEL-SNAP", "MARVEL SNAP", "Gold", "Isi ulang Gold MARVEL SNAP", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/aa/51/a7/aa51a7bb-64a1-9bdb-1a27-1f3630a14f29/AppIcon-0-0-1x_U007emarketing-0-6-0-85-220.png/512x512bb.jpg", 15000),
		demoArtwork("YUGIOH-MASTER", "Yu-Gi-Oh! Master Duel", "Gem", "Isi ulang gem Master Duel", "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/7a/3b/45/7a3b450d-accc-a441-5fb1-8bf288766a63/AppIcon-0-0-1x_U007emarketing-0-8-0-85-220.png/512x512bb.jpg", 16500),
		demoService("WUTHERING-WAVES", "Wuthering Waves", "Astrite", "Isi ulang Astrite Wuthering Waves", "wutheringwaves.kurogames.com", 18000),
		demoService("NEVERNESS-NTE", "Neverness to Everness", "Everglass", "Isi ulang mata uang Neverness to Everness", "nevernesstoeverness.com", 17000),
		demoService("ARKNIGHTS-ENDFIELD", "Arknights: Endfield", "Origeometry", "Isi ulang Origeometry Arknights: Endfield", "endfield.gryphline.com", 19000),
		demoService("NIKKE", "Goddess of Victory: NIKKE", "Gem", "Isi ulang gem NIKKE", "nikke-en.com", 14500),
		demoService("BLUE-ARCHIVE", "Blue Archive", "Pyroxene", "Isi ulang Pyroxene Blue Archive", "bluearchive.nexon.com", 15000),
		demoService("DISLYTE", "Dislyte", "Crystal", "Isi ulang crystal Dislyte", "dislyte.farlightgames.com", 14000),
		demoService("UMA-MUSUME", "Uma Musume: Pretty Derby", "Jewels", "Isi ulang jewels Uma Musume", "umamusume.com", 15000),
		demoService("GF2-EXILIUM", "Girls' Frontline 2: Exilium", "Collapse Piece", "Isi ulang Collapse Piece", "gf2.sunborngame.com", 16000),
		demoService("PUNISHING-GRAY", "Punishing: Gray Raven", "Black Card", "Isi ulang Black Card", "pgr.kurogame.net", 15500),
		demoService("PRINCESS-CONNECT", "Princess Connect! Re:Dive", "Jewel", "Isi ulang jewel Princess Connect", "priconne-redive.jp", 14500),
		demoService("GRAND-CHASE", "GrandChase", "Rune", "Isi ulang rune GrandChase", "grandchase.net", 13000),
		demoService("NOBLESSE", "Noblesse", "Soul Gem", "Isi ulang Soul Gem Noblesse", "noblesse.com", 12500),
		demoService("AETHER-GAZER", "Aether Gazer", "Modifier", "Isi ulang modifier Aether Gazer", "aethergazer.com", 15000),
		demoService("BROWN-DUST-2", "Brown Dust 2", "Lustrous Crystal", "Isi ulang Lustrous Crystal", "browndust2.com", 14000),
		demoService("NIER-REINCAR", "NieR Reincarnation", "Gem", "Isi ulang gem NieR Reincarnation", "nierrein.com", 15500),
	}
}

func demoService(code, game, name, description, domain string, price int64) models.GameService {
	return models.GameService{
		Code:        code,
		Game:        game,
		Name:        name,
		Price:       models.ServicePrice{Basic: price, Premium: price + 5000, Special: price + 10000},
		Description: description,
		Status:      "available",
		Icon:        fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=256", domain),
	}
}

func demoIconService(code, game, name, description, iconSlug string, price int64) models.GameService {
	return models.GameService{
		Code:        code,
		Game:        game,
		Name:        name,
		Price:       models.ServicePrice{Basic: price, Premium: price + 5000, Special: price + 10000},
		Description: description,
		Status:      "available",
		Icon:        fmt.Sprintf("https://cdn.simpleicons.org/%s", iconSlug),
	}
}

func demoArtwork(code, game, name, description, artworkURL string, price int64) models.GameService {
	return models.GameService{
		Code:        code,
		Game:        game,
		Name:        name,
		Price:       models.ServicePrice{Basic: price, Premium: price + 5000, Special: price + 10000},
		Description: description,
		Status:      "available",
		Icon:        artworkURL,
	}
}
