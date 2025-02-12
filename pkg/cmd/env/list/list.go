package list

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/internal/tableprinter"
	"github.com/cli/cli/v2/internal/text"
	"github.com/cli/cli/v2/pkg/cmd/env/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	BaseRepo   func() (ghrepo.Interface, error)
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	Exporter   cmdutil.Exporter

	Limit int
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := ListOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List GitHub deployment environments",
		Example: heredoc.Doc(`
			# List environments in the current repository
			$ gh env list
		`),
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// support `-R, --repo` override
			opts.BaseRepo = f.BaseRepo

			if opts.Limit < 1 {
				return cmdutil.FlagErrorf("invalid limit: %v", opts.Limit)
			}

			if runF != nil {
				return runF(&opts)
			}
			return listRun(&opts)
		},
	}

	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of environments to fetch")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, shared.EnvironmentFields)

	return cmd
}

func listRun(opts *ListOptions) error {
	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}
	client := api.NewClientFromHTTP(httpClient)

	opts.IO.StartProgressIndicator()
	result, err := listEnvironments(client, repo, opts.Limit, opts.IO.IsStdoutTTY())
	opts.IO.StopProgressIndicator()
	if err != nil {
		return fmt.Errorf("%s Failed to get environments: %w", opts.IO.ColorScheme().FailureIcon(), err)
	}

	if len(result.Environments) == 0 && opts.Exporter == nil {
		return cmdutil.NewNoResultsError(fmt.Sprintf("No environments found in %s", ghrepo.FullName(repo)))
	}

	if err := opts.IO.StartPager(); err == nil {
		defer opts.IO.StopPager()
	} else {
		fmt.Fprintf(opts.IO.Out, "Failed to start pager: %v\n", err)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, result.Environments)
	}

	if opts.IO.IsStdoutTTY() {
		fmt.Fprintf(opts.IO.Out, "\nShowing %d of %s in %s\n\n", len(result.Environments), text.Pluralize(result.TotalCount, "environment"), ghrepo.FullName(repo))
		tp := tableprinter.New(opts.IO, tableprinter.WithHeader("NAME", "PROTECTION RULES", "SECRETS", "VARIABLES"))
		for _, environment := range result.Environments {
			tp.AddField(environment.Name)
			tp.AddField(fmt.Sprintf("%d", len(environment.ProtectionRules)))
			tp.AddField(fmt.Sprintf("%d", environment.SecretsTotalCount))
			tp.AddField(fmt.Sprintf("%d", environment.VariablesTotalCount))
			tp.EndRow()
		}
		return tp.Render()
	} else {
		for _, env := range result.Environments {
			fmt.Fprintf(opts.IO.Out, "%s\n", env.Name)
		}
	}

	return nil
}

func listEnvironments(client *api.Client, repo ghrepo.Interface, limit int, tty bool) (*shared.EnvironmentPayload, error) {
	path := fmt.Sprintf("repos/%s/environments", ghrepo.FullName(repo))

	perPage := 100
	if limit > 0 && limit < 100 {
		perPage = limit
	}
	path += fmt.Sprintf("?per_page=%d", perPage)

	var result *shared.EnvironmentPayload
pagination:
	for path != "" {
		var response shared.EnvironmentPayload
		var err error
		path, err = client.RESTWithNext(repo.RepoHost(), "GET", path, nil, &response)
		if err != nil {
			return nil, err
		}

		if result == nil {
			result = &response
		} else {
			result.Environments = append(result.Environments, response.Environments...)
		}

		if limit > 0 && len(result.Environments) >= limit {
			result.Environments = result.Environments[:limit]
			break pagination
		}
	}

	if tty {
		for idx, environment := range result.Environments {
			secretsTotalCount, err := getSecretsTotalCount(client, repo, environment.Name)
			if err != nil {
				return nil, err
			}
			variablesTotalCount, err := getVariablesTotalCount(client, repo, environment.Name)
			if err != nil {
				return nil, err
			}
			result.Environments[idx].SecretsTotalCount = secretsTotalCount
			result.Environments[idx].VariablesTotalCount = variablesTotalCount
		}
	}

	return result, nil
}

func getSecretsTotalCount(client *api.Client, repo ghrepo.Interface, environment string) (int, error) {
	path := fmt.Sprintf("repos/%s/environments/%s/secrets?per_page=1", ghrepo.FullName(repo), environment)

	var response struct {
		TotalCount int `json:"total_count"`
	}

	err := client.REST(repo.RepoHost(), "GET", path, nil, &response)
	if err != nil {
		return 0, fmt.Errorf("could not get secrets total count for %s environment: %w", environment, err)
	}

	return response.TotalCount, nil
}

func getVariablesTotalCount(client *api.Client, repo ghrepo.Interface, environment string) (int, error) {
	path := fmt.Sprintf("repos/%s/environments/%s/variables?per_page=1", ghrepo.FullName(repo), environment)

	var response struct {
		TotalCount int `json:"total_count"`
	}

	err := client.REST(repo.RepoHost(), "GET", path, nil, &response)
	if err != nil {
		return 0, fmt.Errorf("could not get variables total count for %s environment: %w", environment, err)
	}

	return response.TotalCount, nil
}
