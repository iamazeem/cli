package list

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmd/env/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wants    ListOptions
		wantsErr string
	}{
		{
			name:  "no arguments",
			input: "",
			wants: ListOptions{
				Limit: 30,
			},
		},
		{
			name:  "with limit",
			input: "--limit 100",
			wants: ListOptions{
				Limit: 100,
			},
		},
		{
			name:     "invalid limit",
			input:    "-L 0",
			wantsErr: "invalid limit: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &cmdutil.Factory{}
			argv, err := shlex.Split(tt.input)
			assert.NoError(t, err)
			var gotOpts *ListOptions
			cmd := NewCmdList(f, func(opts *ListOptions) error {
				gotOpts = opts
				return nil
			})
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantsErr != "" {
				assert.EqualError(t, err, tt.wantsErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wants.Limit, gotOpts.Limit)
		})
	}
}

func TestListRun(t *testing.T) {
	tests := []struct {
		name       string
		opts       ListOptions
		stubs      func(*httpmock.Registry)
		tty        bool
		wantErr    bool
		wantErrMsg string
		wantStderr string
		wantStdout string
	}{
		{
			name: "displays results tty",
			tty:  true,
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/environments"),
					httpmock.JSONResponse(shared.EnvironmentPayload{
						Environments: []shared.Environment{
							{
								Name: "dev",
							},
							{
								Name: "prod",
							},
						},
						TotalCount: 2,
					}),
				)
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/environments/dev/secrets"),
					httpmock.StringResponse(`{"total_count":0}`),
				)
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/environments/prod/secrets"),
					httpmock.StringResponse(`{"total_count":0}`),
				)
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/environments/dev/variables"),
					httpmock.StringResponse(`{"total_count":0}`),
				)
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/environments/prod/variables"),
					httpmock.StringResponse(`{"total_count":0}`),
				)
			},
			wantStdout: heredoc.Doc(`

				Showing 2 of 2 environments in OWNER/REPO

				NAME  PROTECTION RULES  SECRETS  VARIABLES
				dev   0                 0        0
				prod  0                 0        0
			`),
		},
		{
			name: "displays results non-tty",
			tty:  false,
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/environments"),
					httpmock.JSONResponse(shared.EnvironmentPayload{
						Environments: []shared.Environment{
							{
								Name: "dev",
							},
							{
								Name: "prod",
							},
						},
						TotalCount: 2,
					}),
				)
			},
			wantStdout: "dev\nprod\n",
		},
		{
			name: "displays no results when there is a tty",
			tty:  true,
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/environments"),
					httpmock.JSONResponse(shared.EnvironmentPayload{
						Environments: []shared.Environment{},
						TotalCount:   0,
					}),
				)
			},
			wantErr:    true,
			wantErrMsg: "No environments found in OWNER/REPO",
		},
		{
			name: "displays list error",
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/environments"),
					httpmock.StatusStringResponse(404, "Not Found"),
				)
			},
			wantErr:    true,
			wantErrMsg: "X Failed to get environments: HTTP 404 (https://api.github.com/repos/OWNER/REPO/environments?per_page=100)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &httpmock.Registry{}
			if tt.stubs != nil {
				tt.stubs(reg)
			}
			tt.opts.HttpClient = func() (*http.Client, error) {
				return &http.Client{Transport: reg}, nil
			}
			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			ios.SetStdinTTY(tt.tty)
			ios.SetStderrTTY(tt.tty)
			tt.opts.IO = ios
			tt.opts.BaseRepo = func() (ghrepo.Interface, error) {
				return ghrepo.New("OWNER", "REPO"), nil
			}
			defer reg.Verify(t)

			err := listRun(&tt.opts)
			if tt.wantErr {
				if tt.wantErrMsg != "" {
					assert.EqualError(t, err, tt.wantErrMsg)
				} else {
					assert.Error(t, err)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantStdout, stdout.String())
			assert.Equal(t, tt.wantStderr, stderr.String())
		})
	}
}
