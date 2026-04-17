module github.com/nguyenlinh13602/goshell

go 1.21

require (
	github.com/BurntSushi/toml v1.3.2
	github.com/UserExistsError/conpty v0.1.4
	github.com/creack/pty v1.1.24
	github.com/hinshun/vt10x v0.0.0-20220301184237-5011da428d02
	github.com/peterh/liner v1.2.2
	golang.org/x/term v0.18.0
)

require (
	github.com/mattn/go-runewidth v0.0.3 // indirect
	golang.org/x/sys v0.18.0 // indirect
)

// Dùng vterm + go-runewidth để improve từng phần:
//   - Giữ architecture hiện tại
//   - Thay RuneAccumulator + escape sequence logic → vterm
//   - Thay RuneWidth() → go-runewidth
//   - Giữ nguyên suggestion engine, linebuf, config
