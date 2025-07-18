module github.com/antibyte/retroterm

go 1.23.0

toolchain go1.24.2

require (
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	golang.org/x/crypto v0.39.0
	modernc.org/sqlite v1.38.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	modernc.org/libc v1.66.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/antibyte/retroterm/pkg/terminal => ./pkg/terminal

replace github.com/antibyte/retroterm/pkg/tinybasic => ./pkg/tinybasic

replace github.com/antibyte/retroterm/pkg/tinyos => ./pkg/tinyos

replace github.com/antibyte/retroterm/pkg/virtualfs => ./pkg/virtualfs
