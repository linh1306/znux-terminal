package specs

func init() {
	Register("docker", &DockerSpec)
}

var DockerSpec = Spec{
	Name: "docker",
	Subcommands: []Subcommand{
		{Name: "run", Description: "Run a container", Options: []Option{
			{Names: []string{"-d", "--detach"}, Description: "Run in background"},
			{Names: []string{"-it", "--interactive", "--tty"}, Description: "Interactive mode"},
			{Names: []string{"-p", "--publish"}, Description: "Publish port(s)"},
			{Names: []string{"-v", "--volume"}, Description: "Bind mount volume"},
			{Names: []string{"--name"}, Description: "Container name"},
			{Names: []string{"--rm"}, Description: "Remove when exits"},
			{Names: []string{"--env"}, Description: "Set environment variable"},
			{Names: []string{"-e"}, Description: "Set environment variable"},
			{Names: []string{"--network"}, Description: "Connect to network"},
		}},
		{Name: "ps", Description: "List containers", Options: []Option{
			{Names: []string{"-a", "--all"}, Description: "Show all containers"},
			{Names: []string{"-q", "--quiet"}, Description: "Only show IDs"},
			{Names: []string{"-l", "--latest"}, Description: "Show latest"},
		}},
		{Name: "images", Description: "List images", Options: []Option{
			{Names: []string{"-a", "--all"}, Description: "Show all images"},
			{Names: []string{"-q", "--quiet"}, Description: "Only show IDs"},
		}},
		{Name: "build", Description: "Build an image", Options: []Option{
			{Names: []string{"-t", "--tag"}, Description: "Name and tag"},
			{Names: []string{"-f", "--file"}, Description: "Dockerfile path"},
			{Names: []string{"--no-cache"}, Description: "Build without cache"},
			{Names: []string{"--progress"}, Description: "Build output type"},
		}},
		{Name: "exec", Description: "Execute command in container", Options: []Option{
			{Names: []string{"-it", "--interactive", "--tty"}, Description: "Interactive"},
			{Names: []string{"-u", "--user"}, Description: "Run as user"},
		}},
		{Name: "logs", Description: "Fetch container logs", Options: []Option{
			{Names: []string{"-f", "--follow"}, Description: "Follow log output"},
			{Names: []string{"--tail"}, Description: "Number of lines"},
		}},
		{Name: "stop", Description: "Stop container(s)", Options: []Option{}},
		{Name: "start", Description: "Start container(s)", Options: []Option{}},
		{Name: "rm", Description: "Remove container(s)", Options: []Option{
			{Names: []string{"-f", "--force"}, Description: "Force remove"},
		}},
		{Name: "rmi", Description: "Remove image(s)", Options: []Option{
			{Names: []string{"-f", "--force"}, Description: "Force remove"},
		}},
		{Name: "pull", Description: "Pull image", Options: []Option{
			{Names: []string{"--all-tags"}, Description: "Pull all tags"},
		}},
		{Name: "push", Description: "Push image", Options: []Option{}},
		{Name: "network", Description: "Manage networks", Options: []Option{
			{Names: []string{"ls"}, Description: "List networks"},
			{Names: []string{"rm"}, Description: "Remove network"},
			{Names: []string{"prune"}, Description: "Remove unused networks"},
		}},
		{Name: "volume", Description: "Manage volumes", Options: []Option{
			{Names: []string{"ls"}, Description: "List volumes"},
			{Names: []string{"rm"}, Description: "Remove volume"},
			{Names: []string{"prune"}, Description: "Remove unused volumes"},
		}},
		{Name: "compose", Description: "Docker Compose", Options: []Option{
			{Names: []string{"up"}, Description: "Create and start containers"},
			{Names: []string{"down"}, Description: "Stop and remove containers"},
			{Names: []string{"build"}, Description: "Build images"},
			{Names: []string{"logs"}, Description: "View logs"},
			{Names: []string{"ps"}, Description: "List containers"},
			{Names: []string{"restart"}, Description: "Restart services"},
		}},
	},
}
