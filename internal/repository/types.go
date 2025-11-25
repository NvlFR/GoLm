package repository

type Account struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Active   bool   `json:"active"`
}

type Settings struct {
	ProxyList     []string `json:"proxy_list"`
	SiteID        string `json:"site_id"`       // Target Cabang
	WarTime       string `json:"war_time"`      // Jam Perang (07:00:00)
	TwoCaptchaKey string `json:"twocaptcha_key"`
	Debug         bool   `json:"debug"`
}

type WakdaData struct {
	LastUpdate string `json:"last_update"` // Tanggal update (2025-11-20)
	SiteID     string `json:"site_id"`
	WakdaID    string `json:"wakda_id"`
	Label      string `json:"label"`
}