module github.com/username/golm

go 1.23.0

require (
	github.com/bogdanfinn/fhttp v0.5.28
	github.com/bogdanfinn/tls-client v1.7.2
	github.com/joho/godotenv v1.5.1
	golang.org/x/net v0.33.0
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/bogdanfinn/utls v1.6.1 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/cloudflare/circl v1.3.6 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/manifoldco/promptui v0.9.0 // indirect
	github.com/quic-go/quic-go v0.37.4 // indirect
	github.com/tam7t/hpkp v0.0.0-20160821193359-2b70b4024ed5 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

// INI JURUS KUNCINYA:
// Kita paksa UTLS memakai v1.6.1 yang kompatibel dengan tls-client v1.7.2
replace github.com/bogdanfinn/utls => github.com/bogdanfinn/utls v1.6.1
