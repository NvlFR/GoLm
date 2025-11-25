package repository

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

const (
	FileAccounts = "database/accounts.json"
	FileSettings = "database/settings.json"
	FileWakda    = "database/wakda.json"
)

// Helper: Cek file ada/tidak, jika tidak buat baru
func ensureFile(filename string, defaultContent interface{}) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		data, _ := json.MarshalIndent(defaultContent, "", "  ")
		_ = ioutil.WriteFile(filename, data, 0644)
	}
}

func InitDB() {
	_ = os.Mkdir("database", 0755)
	ensureFile(FileAccounts, []Account{})
	
	// Default settings dengan list kosong
	ensureFile(FileSettings, Settings{
		ProxyList:     []string{}, // Kosong dulu
		SiteID:        "3",
		WarTime:       "07:00:00",
		Debug:         true,
	})
	
	ensureFile(FileWakda, WakdaData{})
}

// --- CRUD ACCOUNTS ---
func GetAccounts() ([]Account, error) {
	var accounts []Account
	data, err := ioutil.ReadFile(FileAccounts)
	if err != nil { return nil, err }
	json.Unmarshal(data, &accounts)
	return accounts, nil
}

func SaveAccount(acc Account) error {
	accounts, _ := GetAccounts()
	accounts = append(accounts, acc)
	data, _ := json.MarshalIndent(accounts, "", "  ")
	return ioutil.WriteFile(FileAccounts, data, 0644)
}

func DeleteAccount(index int) error {
	accounts, _ := GetAccounts()
	if index < 0 || index >= len(accounts) { return nil }
	// Remove element
	accounts = append(accounts[:index], accounts[index+1:]...)
	data, _ := json.MarshalIndent(accounts, "", "  ")
	return ioutil.WriteFile(FileAccounts, data, 0644)
}

// --- CRUD SETTINGS ---
func GetSettings() (Settings, error) {
	var settings Settings
	data, err := ioutil.ReadFile(FileSettings)
	if err != nil { return settings, err }
	json.Unmarshal(data, &settings)
	return settings, nil
}

func SaveSettings(s Settings) error {
	data, _ := json.MarshalIndent(s, "", "  ")
	return ioutil.WriteFile(FileSettings, data, 0644)
}