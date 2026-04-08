package specs

func init() {
	Register("kubectl", &KubectlSpec)
}

var KubectlSpec = Spec{
	Name: "kubectl",
	Subcommands: []Subcommand{
		{Name: "get", Description: "Display resources", Options: []Option{
			{Names: []string{"-o", "--output"}, Description: "Output format (yaml/json/wide)"},
			{Names: []string{"-A", "--all-namespaces"}, Description: "All namespaces"},
			{Names: []string{"-n", "--namespace"}, Description: "Namespace"},
			{Names: []string{"-w", "--watch"}, Description: "Watch for changes"},
			{Names: []string{"--show-kind"}, Description: "Show resource kind"},
			{Names: []string{"-l", "--selector"}, Description: "Label selector"},
		}},
		{Name: "describe", Description: "Show details of resources", Options: []Option{
			{Names: []string{"-A", "--all-namespaces"}, Description: "All namespaces"},
			{Names: []string{"-n", "--namespace"}, Description: "Namespace"},
		}},
		{Name: "apply", Description: "Apply configuration", Options: []Option{
			{Names: []string{"-f", "--filename"}, Description: "Filename or URL"},
			{Names: []string{"-k", "--kustomize"}, Description: "Kustomize directory"},
			{Names: []string{"--dry-run"}, Description: "Dry run"},
			{Names: []string{"--validate"}, Description: "Validate resources"},
		}},
		{Name: "delete", Description: "Delete resources", Options: []Option{
			{Names: []string{"-f", "--filename"}, Description: "Filename or URL"},
			{Names: []string{"--all"}, Description: "Delete all resources"},
			{Names: []string{"--cascade"}, Description: "Cascade delete"},
		}},
		{Name: "create", Description: "Create resources", Options: []Option{
			{Names: []string{"-f", "--filename"}, Description: "Filename or URL"},
			{Names: []string{"--dry-run"}, Description: "Dry run"},
		}},
		{Name: "edit", Description: "Edit resources", Options: []Option{
			{Names: []string{"--output"}, Description: "Output format"},
		}},
		{Name: "logs", Description: "Print logs", Options: []Option{
			{Names: []string{"-f", "--follow"}, Description: "Follow logs"},
			{Names: []string{"-p", "--previous"}, Description: "Previous container"},
			{Names: []string{"--tail"}, Description: "Lines to show"},
			{Names: []string{"-c", "--container"}, Description: "Container name"},
		}},
		{Name: "exec", Description: "Execute command in container", Options: []Option{
			{Names: []string{"-it"}, Description: "Interactive"},
			{Names: []string{"-c", "--container"}, Description: "Container name"},
		}},
		{Name: "port-forward", Description: "Forward port(s)", Options: []Option{
			{Names: []string{"--address"}, Description: "Bind address"},
		}},
		{Name: "scale", Description: "Scale a deployment", Options: []Option{
			{Names: []string{"--replicas"}, Description: "Number of replicas"},
		}},
		{Name: "rollout", Description: "Manage rollout", Options: []Option{
			{Names: []string{"status"}, Description: "Check rollout status"},
			{Names: []string{"undo"}, Description: "Undo previous rollout"},
			{Names: []string{"restart"}, Description: "Restart deployment"},
		}},
		{Name: "top", Description: "Show resource usage", Options: []Option{
			{Names: []string{"-A", "--all-namespaces"}, Description: "All namespaces"},
			{Names: []string{"-n", "--namespace"}, Description: "Namespace"},
		}},
		{Name: "explain", Description: "Explain resource definitions", Options: []Option{}},
		{Name: "config", Description: "Manage config", Options: []Option{
			{Names: []string{"use-context"}, Description: "Switch context"},
			{Names: []string{"get-contexts"}, Description: "List contexts"},
			{Names: []string{"current-context"}, Description: "Current context"},
		}},
		{Name: "cluster", Description: "Manage cluster", Options: []Option{
			{Names: []string{"info"}, Description: "Cluster info"},
		}},
		{Name: "api-resources", Description: "List API resources", Options: []Option{
			{Names: []string{"--api-group"}, Description: "API group"},
			{Names: []string{"-o", "--output"}, Description: "Output format"},
		}},
	},
}
