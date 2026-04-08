package specs

func init() {
	Register("npm", &NPMSpec)
}

var NPMSpec = Spec{
	Name: "npm",
	Subcommands: []Subcommand{
		{Name: "install", Description: "Install dependencies", Options: []Option{
			{Names: []string{"-g", "--global"}, Description: "Install globally"},
			{Names: []string{"--save"}, Description: "Save to dependencies"},
			{Names: []string{"--save-dev"}, Description: "Save to devDependencies"},
			{Names: []string{"--save-optional"}, Description: "Save to optionalDependencies"},
			{Names: []string{"--no-save"}, Description: "Don't save"},
			{Names: []string{"-D", "--save-dev"}, Description: "Save to devDependencies"},
			{Names: []string{"-P", "--save-prod"}, Description: "Save to dependencies"},
			{Names: []string{"-O", "--save-optional"}, Description: "Save to optionalDependencies"},
			{Names: []string{"--legacy-peer-deps"}, Description: "Legacy peer deps"},
			{Names: []string{"--force"}, Description: "Force install"},
		}},
		{Name: "uninstall", Description: "Remove dependencies", Options: []Option{
			{Names: []string{"-g", "--global"}, Description: "Uninstall globally"},
			{Names: []string{"--save"}, Description: "Remove from dependencies"},
			{Names: []string{"--save-dev"}, Description: "Remove from devDependencies"},
		}},
		{Name: "run", Description: "Run a script", Options: []Option{
			{Names: []string{"--if-present"}, Description: "Run if script exists"},
			{Names: []string{"--ignore-scripts"}, Description: "Ignore scripts"},
		}},
		{Name: "test", Description: "Run tests", Options: []Option{
			{Names: []string{"--watch"}, Description: "Watch mode"},
		}},
		{Name: "start", Description: "Run start script", Options: []Option{}},
		{Name: "dev", Description: "Run dev script", Options: []Option{}},
		{Name: "build", Description: "Run build script", Options: []Option{}},
		{Name: "publish", Description: "Publish package", Options: []Option{
			{Names: []string{"--access"}, Description: "Access (public/restricted)"},
			{Names: []string{"--tag"}, Description: "Distribution tag"},
			{Names: []string{"--dry-run"}, Description: "Dry run"},
		}},
		{Name: "init", Description: "Initialize package", Options: []Option{
			{Names: []string{"-y", "--yes"}, Description: "Accept defaults"},
			{Names: []string{"-n"}, Description: "No modifications"},
		}},
		{Name: "version", Description: "Bump version", Options: []Option{
			{Names: []string{"--major"}, Description: "Major version bump"},
			{Names: []string{"--minor"}, Description: "Minor version bump"},
			{Names: []string{"--patch"}, Description: "Patch version bump"},
			{Names: []string{"--git-tag-version"}, Description: "Git tag"},
		}},
		{Name: "audit", Description: "Check for vulnerabilities", Options: []Option{
			{Names: []string{"--fix"}, Description: "Fix automatically"},
		}},
		{Name: "ci", Description: "Clean install", Options: []Option{
			{Names: []string{"--legacy-peer-deps"}, Description: "Legacy peer deps"},
		}},
		{Name: "update", Description: "Update packages", Options: []Option{
			{Names: []string{"-g", "--global"}, Description: "Update global"},
		}},
		{Name: "list", Description: "List installed packages", Options: []Option{
			{Names: []string{"-g", "--global"}, Description: "Global packages"},
			{Names: []string{"--depth"}, Description: "Max depth"},
		}},
		{Name: "outdated", Description: "Check outdated packages", Options: []Option{
			{Names: []string{"--depth"}, Description: "Max depth"},
		}},
	},
}
