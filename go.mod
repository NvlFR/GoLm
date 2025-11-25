module github.com/username/golm

go 1.23.0

require (
	github.com/bogdanfinn/fhttp v0.5.30
	github.com/bogdanfinn/tls-client v1.7.10
	github.com/joho/godotenv v1.5.1
	golang.org/x/net v0.33.0
)

require (
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/bogdanfinn/utls v1.6.2 // indirect
	github.com/cloudflare/circl v1.5.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/quic-go/quic-go v0.48.1 // indirect
	github.com/tam7t/hpkp v0.0.0-20160821193359-2b70b4024ed5 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

// INI KUNCINYA: Paksa pakai v1.6.3 yang jodohnya tls-client v1.7.10
replace github.com/bogdanfinn/utls => github.com/bogdanfinn/utls v1.6.3
