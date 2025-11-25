package repository

// [BARU] Struct untuk menyimpan satu baris cookie
type CookieEntry struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

type Account struct {
	Username string        `json:"username"`
	Password string        `json:"password"`
	// [BARU] Field ini wajib ada untuk menyimpan sesi login
	Cookies  []CookieEntry `json:"cookies"` 
	Active   bool          `json:"active"`
}

type Settings struct {
	ProxyList     []string `json:"proxy_list"`
	SiteID        string   `json:"site_id"`
	WarTime       string   `json:"war_time"`
	TwoCaptchaKey string   `json:"twocaptcha_key"`
	Debug         bool     `json:"debug"`
}

type WakdaData struct {
	LastUpdate string `json:"last_update"`
	SiteID     string `json:"site_id"`
	WakdaID    string `json:"wakda_id"`
	Label      string `json:"label"`
}