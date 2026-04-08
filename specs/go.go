package specs

func init() {
	Register("go", &GoSpec)
}

var GoSpec = Spec{
	Name: "go",
	Subcommands: []Subcommand{
		{Name: "run", Description: "Compile and run Go file", Options: []Option{
			{Names: []string{"-v", "--verbose"}, Description: "Verbose"},
			{Names: []string{"-x"}, Description: "Print commands"},
		}},
		{Name: "build", Description: "Compile packages", Options: []Option{
			{Names: []string{"-v", "--verbose"}, Description: "Verbose"},
			{Names: []string{"-x"}, Description: "Print commands"},
			{Names: []string{"-o", "--output"}, Description: "Output file"},
			{Names: []string{"-a", "--all"}, Description: "Force rebuild"},
			{Names: []string{"-race"}, Description: "Race detector"},
			{Names: []string{"-ldflags"}, Description: "Linker flags"},
			{Names: []string{"-gcflags"}, Description: "Compiler flags"},
			{Names: []string{"-tags"}, Description: "Build tags"},
		}},
		{Name: "test", Description: "Run tests", Options: []Option{
			{Names: []string{"-v", "--verbose"}, Description: "Verbose"},
			{Names: []string{"-run"}, Description: "Run tests matching regex"},
			{Names: []string{"-bench"}, Description: "Run benchmarks matching regex"},
			{Names: []string{"-benchtime"}, Description: "Benchmark time"},
			{Names: []string{"-count"}, Description: "Run count"},
			{Names: []string{"-cover"}, Description: "Coverage"},
			{Names: []string{"-coverprofile"}, Description: "Coverage profile"},
			{Names: []string{"-race"}, Description: "Race detector"},
			{Names: []string{"-timeout"}, Description: "Timeout"},
			{Names: []string{"-short"}, Description: "Short mode"},
		}},
		{Name: "vet", Description: "Run go vet", Options: []Option{
			{Names: []string{"-v"}, Description: "Verbose"},
		}},
		{Name: "fmt", Description: "Format packages", Options: []Option{
			{Names: []string{"-l", "--list"}, Description: "List files"},
			{Names: []string{"-w", "--write"}, Description: "Write result"},
			{Names: []string{"-r"}, Description: "Rewrite rule"},
		}},
		{Name: "mod", Description: "Module maintenance", Options: []Option{
			{Names: []string{"init"}, Description: "Initialize go.mod"},
			{Names: []string{"tidy"}, Description: "Update go.mod"},
			{Names: []string{"download"}, Description: "Download modules"},
			{Names: []string{"graph"}, Description: "Module graph"},
			{Names: []string{"why"}, Description: "Explain dependencies"},
		}},
		{Name: "get", Description: "Add dependency", Options: []Option{
			{Names: []string{"-v", "--verbose"}, Description: "Verbose"},
			{Names: []string{"-u"}, Description: "Update"},
		}},
		{Name: "install", Description: "Build and install packages", Options: []Option{
			{Names: []string{"-v"}, Description: "Verbose"},
			{Names: []string{"-race"}, Description: "Race detector"},
		}},
		{Name: "clean", Description: "Clean cache", Options: []Option{
			{Names: []string{"-cache"}, Description: "Clean build cache"},
			{Names: []string{"-testcache"}, Description: "Clean test cache"},
			{Names: []string{"-modcache"}, Description: "Clean module cache"},
		}},
		{Name: "doc", Description: "Show documentation", Options: []Option{
			{Names: []string{"-v", "--verbose"}, Description: "Verbose"},
			{Names: []string{"-all"}, Description: "Show unexported"},
		}},
		{Name: "list", Description: "List packages", Options: []Option{
			{Names: []string{"-m"}, Description: "List modules"},
			{Names: []string{"-f"}, Description: "Format"},
			{Names: []string{"-templates"}, Description: "Templates"},
		}},
		{Name: "version", Description: "Print Go version", Options: []Option{}},
		{Name: "env", Description: "Print environment", Options: []Option{
			{Names: []string{"-json"}, Description: "JSON output"},
			{Names: []string{"-w"}, Description: "Set env vars"},
		}},
	},
}
