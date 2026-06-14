package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vastimofeev/yandex-tracker-cli/internal/app"
	"github.com/vastimofeev/yandex-tracker-cli/internal/auth"
	"github.com/vastimofeev/yandex-tracker-cli/internal/client"
	"github.com/vastimofeev/yandex-tracker-cli/internal/config"
	"github.com/vastimofeev/yandex-tracker-cli/internal/model"
	"github.com/vastimofeev/yandex-tracker-cli/internal/service"
	"github.com/vastimofeev/yandex-tracker-cli/internal/version"
)

const loovTeamOrgID = "7942431"

type rootOptions struct {
	ConfigPath string
	BaseURL    string
	Token      string
	TokenType  string
	OrgID      string
	OrgHeader  string
	JSON       bool
	Debug      bool
}

func NewRootCommand(application *app.App) *cobra.Command {
	opts := &rootOptions{}
	cmd := &cobra.Command{
		Use:   "yt",
		Short: "Yandex Tracker CLI for issues, queues, entities, and metadata",
		Long: `Yandex Tracker CLI exposes the Tracker API as a compact command-line tool.

Use --json for agent-friendly output. Text output is intended for quick manual inspection.

Core areas:
  auth       authenticate and inspect the current session
  issue      read, search, create, edit, and transition issues
  queue      inspect queues and queue field metadata
  entity     list and inspect projects, portfolios, and goals
  field      inspect global field definitions
  issuetype  inspect available issue types
  status     inspect workflow statuses
  priority   inspect priority values`,
		Example: strings.TrimSpace(`
  yt auth login
  yt me --json
  yt issue search --query "Queue: DV" --per-page 10 --json
  yt issue create --queue DV --summary "Smoke test" --description "Created from CLI"
  yt entity list --type project --per-page 20 --page 2 --json`),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&opts.ConfigPath, "config", "", "Path to the local config file")
	flags.StringVar(&opts.BaseURL, "base-url", "", "Override the Tracker API base URL")
	flags.StringVar(&opts.Token, "token", "", "Tracker OAuth token override for the current command")
	flags.StringVar(&opts.TokenType, "token-type", "", "Token type override: OAuth or Bearer")
	flags.StringVar(&opts.OrgID, "org-id", "", "Organization ID override for the current command")
	flags.StringVar(&opts.OrgHeader, "org-header", "", "Organization header override: X-Org-ID or X-Cloud-Org-ID")
	flags.BoolVar(&opts.JSON, "json", false, "Render machine-readable JSON for agent workflows")
	flags.BoolVar(&opts.Debug, "debug", false, "enable debug logging")

	cmd.AddCommand(
		newVersionCommand(application, opts),
		newAuthCommand(application, opts),
		newMeCommand(application, opts),
		newIssueCommand(application, opts),
		newQueueCommand(application, opts),
		newFieldCommand(application, opts),
		newIssueTypeCommand(application, opts),
		newStatusCommand(application, opts),
		newPriorityCommand(application, opts),
		newEntityCommand(application, opts),
	)

	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return fmt.Errorf("%s: %w", c.CommandPath(), err)
	})

	return cmd
}

func newVersionCommand(application *app.App, opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show build and version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, err := application.BuildRuntime(cmd.Context(), cliOptions(opts))
			if err != nil {
				return err
			}
			info := version.Current()
			return runtime.Printer.Print(info)
		},
	}
}

func cliOptions(opts *rootOptions) config.CLIOptions {
	return config.CLIOptions{
		ConfigPath: opts.ConfigPath,
		BaseURL:    opts.BaseURL,
		Token:      opts.Token,
		TokenType:  opts.TokenType,
		OrgID:      opts.OrgID,
		OrgHeader:  opts.OrgHeader,
		JSON:       opts.JSON,
		Debug:      opts.Debug,
	}
}

func runtimeFor(cmd *cobra.Command, application *app.App, opts *rootOptions) (*app.Runtime, *service.TrackerService, error) {
	runtime, err := application.BuildRuntime(cmd.Context(), cliOptions(opts))
	if err != nil {
		return nil, nil, err
	}
	if runtime.Config.Auth.Token == "" {
		return runtime, nil, errors.New("tracker credentials are not configured; run `yt auth login` or pass flags/env vars")
	}
	return runtime, service.New(runtime.Client), nil
}

func newAuthCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Tracker authentication",
		Long: `Authenticate the CLI against Yandex Tracker.

If --token is omitted, 'yt auth login' opens the Yandex OAuth page and asks you to paste the token shown by OAuth.
The saved auth context is stored in the system keyring.`,
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "login",
			Short: "Validate credentials and save them to the keyring",
			Example: strings.TrimSpace(`
  yt auth login
  yt --token <oauth-token> auth login
  yt --token <oauth-token> --org-id 123456 auth login
  yt --token <oauth-token> --org-id <cloud-org-id> --org-header X-Cloud-Org-ID auth login`),
			RunE: func(cmd *cobra.Command, args []string) error {
				runtime, err := application.BuildRuntime(cmd.Context(), cliOptions(opts))
				if err != nil {
					return err
				}
				reader := bufio.NewReader(runtime.Input)

				tokenInput := firstNonEmptyString(opts.Token, os.Getenv(config.EnvToken))
				if tokenInput == "" {
					authURL, err := auth.BuildTokenAuthorizeURL(auth.BrowserLoginOptions{
						ClientID:      runtime.Config.OAuth.ClientID,
						RedirectURI:   runtime.Config.OAuth.RedirectURI,
						Scope:         runtime.Config.OAuth.Scope,
						OptionalScope: runtime.Config.OAuth.OptionalScope,
						LoginHint:     runtime.Config.OAuth.LoginHint,
						ResponseType:  runtime.Config.OAuth.ResponseType,
					})
					if err != nil {
						return err
					}
					_, _ = fmt.Fprintf(runtime.Err, "Open this URL to continue login:\n%s\n", authURL)
					if err := auth.OpenBrowserURL(authURL); err != nil {
						return err
					}
					_, _ = fmt.Fprint(runtime.Err, "Paste OAuth token: ")
					tokenInput, err = readToken(reader)
					if err != nil {
						return err
					}
				}

				runtime.Config.Auth.Token = tokenInput
				runtime.Config.Auth.TokenType = config.DefaultTokenType
				oauthUser, err := auth.FetchUserInfo(cmd.Context(), tokenInput, nil)
				if err != nil {
					return err
				}
				if runtime.Config.Auth.OrgID == "" {
					if inferredOrgID, ok := inferOrgIDFromOAuthUser(oauthUser); ok {
						runtime.Config.Auth.OrgID = inferredOrgID
					}
				}
				if runtime.Config.Auth.OrgID == "" {
					_, _ = fmt.Fprint(runtime.Err, "Enter org ID: ")
					orgID, err := readToken(reader)
					if err != nil {
						return err
					}
					runtime.Config.Auth.OrgID = orgID
				}
				runtime.Config.Auth.OrgHeader = config.NormalizeOrgHeader(runtime.Config.Auth.OrgHeader)
				runtime.Client = client.New(runtime.Config.BaseURL, runtime.Config.Auth, nil)
				svc := service.New(runtime.Client)
				user, err := svc.ValidateAuth(cmd.Context())
				if err != nil {
					return err
				}

				if err := runtime.Store.Save(cmd.Context(), service.BuildStoredConfig(runtime.Config)); err != nil {
					return err
				}
				return runtime.Printer.Print(map[string]any{
					"status":    "ok",
					"user":      user,
					"baseURL":   runtime.Config.BaseURL,
					"orgID":     runtime.Config.Auth.OrgID,
					"orgHeader": runtime.Config.Auth.OrgHeader,
					"authStore": currentStoreKind(runtime.Store, runtime.Config.AuthStore),
				})
			},
		},
		&cobra.Command{
			Use:     "status",
			Short:   "Show the saved auth context and validate it when possible",
			Example: "  yt auth status --json",
			RunE: func(cmd *cobra.Command, args []string) error {
				runtime, err := application.BuildRuntime(cmd.Context(), cliOptions(opts))
				if err != nil {
					return err
				}
				status := map[string]any{
					"authStore":         currentStoreKind(runtime.Store, runtime.Config.AuthStore),
					"baseURL":           runtime.Config.BaseURL,
					"configured":        runtime.Config.Auth.Complete(),
					"tokenPresent":      runtime.Config.Auth.Token != "",
					"orgPresent":        runtime.Config.Auth.OrgID != "" && runtime.Config.Auth.OrgHeader != "",
					"orgID":             runtime.Config.Auth.OrgID,
					"orgHeader":         runtime.Config.Auth.OrgHeader,
					"tokenType":         runtime.Config.Auth.TokenType,
					"savedConfigured":   runtime.Stored.Auth.Complete(),
					"savedTokenPresent": runtime.Stored.Auth.Token != "",
					"savedOrgPresent":   runtime.Stored.Auth.OrgID != "" && runtime.Stored.Auth.OrgHeader != "",
					"savedOrgID":        runtime.Stored.Auth.OrgID,
					"savedOrgHeader":    runtime.Stored.Auth.OrgHeader,
					"savedTokenType":    runtime.Stored.Auth.TokenType,
				}
				if runtime.Config.Auth.Token != "" {
					svc := service.New(runtime.Client)
					user, validateErr := svc.ValidateAuth(cmd.Context())
					if validateErr == nil {
						status["user"] = user
						status["validated"] = true
					} else {
						status["validationError"] = validateErr.Error()
						status["validated"] = false
					}
				}
				return runtime.Printer.Print(status)
			},
		},
		&cobra.Command{
			Use:     "logout",
			Short:   "Remove the saved auth context from the keyring",
			Example: "  yt auth logout",
			RunE: func(cmd *cobra.Command, args []string) error {
				runtime, err := application.BuildRuntime(cmd.Context(), cliOptions(opts))
				if err != nil {
					return err
				}
				if err := runtime.Store.Clear(cmd.Context()); err != nil {
					return err
				}
				return runtime.Printer.Print(map[string]string{"status": "logged out"})
			},
		},
	)
	return cmd
}

func newMeCommand(application *app.App, opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:     "me",
		Short:   "Show the authenticated Tracker user",
		Example: "  yt me --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			user, err := svc.ValidateAuth(cmd.Context())
			if err != nil {
				return err
			}
			return runtime.Printer.Print(user)
		},
	}
}

func newIssueCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage Tracker issues",
		Long: `Work with Tracker issues and issue subresources.

The issue command covers:
  get/search/count       read issues
  create/edit            write issue fields
  transitions/transition inspect and execute workflow changes
  comment                manage issue comments
  worklog                manage issue worklog entries
  links                  inspect and create issue links
  checklist              manage checklist items`,
	}

	var searchQuery string
	var searchFilters []string
	var searchKeys []string
	var perPage, scrollPer, scrollTTL, searchLimit int
	var scrollType, scrollID, searchSort, searchOrder string
	var searchSelect []string
	searchCmd := &cobra.Command{
		Use:   "search",
		Short: "Search issues by query, keys, or Tracker filters",
		Example: strings.TrimSpace(`
  yt issue search --query "Queue: DV"
  yt issue search --key DV-1 --key DV-2 --json
  yt issue search --filter queue=DV --filter assignee=v.timofeev --per-page 20 --json
  yt issue search --query '"Project": 766' --sort start --order asc --limit 1 --select key,start,summary,status.display --json`),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			result, err := svc.SearchIssues(cmd.Context(), searchQuery, searchFilters, searchKeys, perPage, scrollType, scrollPer, scrollTTL, scrollID)
			if err != nil {
				return err
			}
			presented, err := presentIssueSearch(result, searchSort, searchOrder, searchLimit, searchSelect)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(presented)
		},
	}
	searchCmd.Flags().StringVar(&searchQuery, "query", "", "query language expression")
	searchCmd.Flags().StringArrayVar(&searchFilters, "filter", nil, "repeatable filter key=value")
	searchCmd.Flags().StringArrayVar(&searchKeys, "key", nil, "search by issue key")
	searchCmd.Flags().IntVar(&perPage, "per-page", 50, "page size")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 0, "limit the number of returned issues after filtering and sorting")
	searchCmd.Flags().StringVar(&searchSort, "sort", "", "sort returned issues by a field path such as start, createdAt, status.display, or assignee.display")
	searchCmd.Flags().StringVar(&searchOrder, "order", "asc", "sort order: asc or desc")
	searchCmd.Flags().StringSliceVar(&searchSelect, "select", nil, "comma-separated field paths to project from each issue, for example key,start,summary,status.display")
	searchCmd.Flags().StringVar(&scrollType, "scroll-type", "", "scroll type: sorted or unsorted")
	searchCmd.Flags().IntVar(&scrollPer, "per-scroll", 0, "scroll page size")
	searchCmd.Flags().IntVar(&scrollTTL, "scroll-ttl-millis", 0, "scroll lifetime in milliseconds")
	searchCmd.Flags().StringVar(&scrollID, "scroll-id", "", "continue scroll with a prior scroll ID")

	var schemaQueue string
	schemaCmd := &cobra.Command{
		Use:   "schema",
		Short: "Show queue-specific issue schema information for agents and automation",
		Long: `Inspect the fields and metadata that shape issues in a queue.

This command is intended to help an agent discover:
  - available queue fields
  - queue-local custom fields
  - required fields
  - allowed issue types
  - default type and priority`,
		Example: strings.TrimSpace(`
  yt issue schema --queue DV --json
  yt issue schema --queue DV`),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			queue, err := svc.GetQueue(cmd.Context(), schemaQueue, []string{"types"})
			if err != nil {
				return err
			}
			fields, err := svc.GetQueueFields(cmd.Context(), schemaQueue)
			if err != nil {
				return err
			}
			localFields, err := svc.GetQueueLocalFields(cmd.Context(), schemaQueue)
			if err != nil {
				return err
			}
			requiredFields := make([]string, 0)
			for _, field := range fields {
				if isRequiredSchemaField(field) && !field.ReadOnly {
					requiredFields = append(requiredFields, field.Key)
				}
			}
			return runtime.Printer.Print(map[string]any{
				"queue":           queue,
				"fields":          fields,
				"localFields":     localFields,
				"requiredFields":  requiredFields,
				"issueTypes":      queue.IssueTypes,
				"defaultType":     queue.DefaultType,
				"defaultPriority": queue.DefaultPriority,
			})
		},
	}
	schemaCmd.Flags().StringVar(&schemaQueue, "queue", "", "queue key to inspect")
	_ = schemaCmd.MarkFlagRequired("queue")

	var countQuery string
	var countFilters []string
	countCmd := &cobra.Command{
		Use:     "count",
		Short:   "Count issues matching a query or filter without returning the full list",
		Example: "  yt issue count --query \"Queue: DV AND Status: \\\"Open\\\"\" --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			result, err := svc.CountIssues(cmd.Context(), countQuery, countFilters)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(result)
		},
	}
	countCmd.Flags().StringVar(&countQuery, "query", "", "query language expression")
	countCmd.Flags().StringArrayVar(&countFilters, "filter", nil, "repeatable filter key=value")

	var createQueue, createType, createSummary, createDescription, createBody string
	var createFields []string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an issue with common flags plus repeatable custom fields",
		Example: strings.TrimSpace(`
  yt issue create --queue DV --summary "CLI smoke test"
  yt issue create --queue DV --type task --summary "Agent task" --description "Created by yt"
  yt issue create --queue DV --summary "With fields" --field assignee=v.timofeev --field tags=cli,test`),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			issue, err := svc.CreateIssue(cmd.Context(), createQueue, createType, createSummary, createDescription, createFields, createBody)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(issue)
		},
	}
	createCmd.Flags().StringVar(&createQueue, "queue", "", "queue key")
	createCmd.Flags().StringVar(&createType, "type", "", "issue type")
	createCmd.Flags().StringVar(&createSummary, "summary", "", "issue summary")
	createCmd.Flags().StringVar(&createDescription, "description", "", "issue description")
	createCmd.Flags().StringArrayVar(&createFields, "field", nil, "repeatable custom field key=value")
	createCmd.Flags().StringVar(&createBody, "body-json", "", "raw JSON payload overlay")
	_ = createCmd.MarkFlagRequired("queue")
	_ = createCmd.MarkFlagRequired("summary")

	var editSummary, editDescription, editBody string
	var editFields []string
	editCmd := &cobra.Command{
		Use:   "edit <issue-key>",
		Short: "Edit issue fields with flags, repeatable custom fields, or raw JSON overlay",
		Example: strings.TrimSpace(`
  yt issue edit DV-1 --summary "Updated title"
  yt issue edit DV-1 --description "Updated description"
  yt issue edit DV-1 --field assignee=v.timofeev --field tags=updated,cli`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			issue, err := svc.EditIssue(cmd.Context(), args[0], editSummary, editDescription, editFields, editBody)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(issue)
		},
	}
	editCmd.Flags().StringVar(&editSummary, "summary", "", "updated summary")
	editCmd.Flags().StringVar(&editDescription, "description", "", "updated description")
	editCmd.Flags().StringArrayVar(&editFields, "field", nil, "repeatable custom field key=value")
	editCmd.Flags().StringVar(&editBody, "body-json", "", "raw JSON payload overlay")

	transitionsCmd := &cobra.Command{
		Use:     "transitions <issue-key>",
		Short:   "List transitions that can be executed for the current issue state",
		Example: "  yt issue transitions DV-1 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			transitions, err := svc.ListTransitions(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return runtime.Printer.Print(transitions)
		},
	}

	var transitionID, transitionTo, transitionComment, transitionBody string
	var transitionFields []string
	transitionCmd := &cobra.Command{
		Use:   "transition <issue-key>",
		Short: "Execute an issue transition by transition ID or target status name",
		Example: strings.TrimSpace(`
  yt issue transition DV-1 --id reopen
  yt issue transition DV-1 --to "In Progress" --comment "Restarting work"`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			issue, err := svc.ExecuteTransition(cmd.Context(), args[0], transitionTo, transitionID, transitionComment, transitionFields, transitionBody)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(issue)
		},
	}
	transitionCmd.Flags().StringVar(&transitionID, "id", "", "transition ID")
	transitionCmd.Flags().StringVar(&transitionTo, "to", "", "target transition or status display")
	transitionCmd.Flags().StringVar(&transitionComment, "comment", "", "optional transition comment")
	transitionCmd.Flags().StringArrayVar(&transitionFields, "field", nil, "repeatable transition field key=value")
	transitionCmd.Flags().StringVar(&transitionBody, "body-json", "", "raw JSON payload overlay")

	getCmd := &cobra.Command{
		Use:     "get <issue-key>",
		Short:   "Get a single issue by key",
		Example: "  yt issue get DV-1 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			issue, err := svc.GetIssue(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return runtime.Printer.Print(issue)
		},
	}

	cmd.AddCommand(getCmd, searchCmd, countCmd, createCmd, editCmd, transitionsCmd, transitionCmd, schemaCmd)
	cmd.AddCommand(newCommentCommand(application, opts))
	cmd.AddCommand(newLinksCommand(application, opts))
	cmd.AddCommand(newChecklistCommand(application, opts))
	cmd.AddCommand(newWorklogCommand(application, opts))
	return cmd
}

func newCommentCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage issue comments",
		Long:  "List, add, and edit comments on a specific issue.",
	}

	var text string
	var summonees []string
	var addToFollowers bool
	addCmd := &cobra.Command{
		Use:   "add <issue-key>",
		Short: "Add a markdown comment to an issue",
		Example: strings.TrimSpace(`
  yt issue comment add DV-1 --text "Investigating"
  yt issue comment add DV-1 --text "Need review" --summonee v.timofeev`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			var value *bool
			if cmd.Flags().Changed("add-to-followers") {
				value = &addToFollowers
			}
			comment, err := svc.AddComment(cmd.Context(), args[0], text, summonees, value)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(comment)
		},
	}
	addCmd.Flags().StringVar(&text, "text", "", "comment text")
	addCmd.Flags().StringArrayVar(&summonees, "summonee", nil, "repeatable login or UID")
	addCmd.Flags().BoolVar(&addToFollowers, "add-to-followers", true, "add author to followers")
	_ = addCmd.MarkFlagRequired("text")

	var perPage, page int
	listCmd := &cobra.Command{
		Use:     "list <issue-key>",
		Short:   "List issue comments with pagination",
		Example: "  yt issue comment list DV-1 --per-page 20 --page 2 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			comments, err := svc.ListComments(cmd.Context(), args[0], perPage, page)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(comments)
		},
	}
	listCmd.Flags().IntVar(&perPage, "per-page", 50, "page size")
	listCmd.Flags().IntVar(&page, "page", 1, "page number")

	var editText string
	var editSummonees []string
	var version int
	editCmd := &cobra.Command{
		Use:   "edit <issue-key> <comment-id>",
		Short: "Edit an existing comment",
		Example: strings.TrimSpace(`
  yt issue comment edit DV-1 123 --text "Updated comment"
  yt issue comment edit DV-1 123 --text "Paging reviewer" --summonee v.timofeev`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			commentID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			comment, err := svc.EditComment(cmd.Context(), args[0], commentID, editText, editSummonees, version)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(comment)
		},
	}
	editCmd.Flags().StringVar(&editText, "text", "", "comment text")
	editCmd.Flags().StringArrayVar(&editSummonees, "summonee", nil, "repeatable login or UID")
	editCmd.Flags().IntVar(&version, "version", 0, "comment version for optimistic update")
	_ = editCmd.MarkFlagRequired("text")

	cmd.AddCommand(addCmd, listCmd, editCmd)
	return cmd
}

func newWorklogCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worklog",
		Short: "Manage issue worklogs",
		Long:  "List and add worklog entries for a specific issue.",
	}

	var duration, comment, start string
	addCmd := &cobra.Command{
		Use:   "add <issue-key>",
		Short: "Add a worklog entry",
		Example: strings.TrimSpace(`
  yt issue worklog add DV-1 --duration PT1H
  yt issue worklog add DV-1 --duration PT30M --comment "Bug triage"`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			entry, err := svc.AddWorklog(cmd.Context(), args[0], duration, comment, start)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(entry)
		},
	}
	addCmd.Flags().StringVar(&duration, "duration", "", "ISO-8601 duration")
	addCmd.Flags().StringVar(&comment, "comment", "", "optional worklog comment")
	addCmd.Flags().StringVar(&start, "start", "", "optional start time")
	_ = addCmd.MarkFlagRequired("duration")

	var perPage, page int
	listCmd := &cobra.Command{
		Use:     "list <issue-key>",
		Short:   "List worklog entries with pagination",
		Example: "  yt issue worklog list DV-1 --per-page 20 --page 2 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			entries, err := svc.ListWorklog(cmd.Context(), args[0], perPage, page)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(entries)
		},
	}
	listCmd.Flags().IntVar(&perPage, "per-page", 50, "page size")
	listCmd.Flags().IntVar(&page, "page", 1, "page number")

	cmd.AddCommand(addCmd, listCmd)
	return cmd
}

func newLinksCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "links",
		Short: "Manage issue links",
		Long:  "Inspect links on an issue and create new links to related issues.",
	}

	listCmd := &cobra.Command{
		Use:     "list <issue-key>",
		Short:   "List links for an issue",
		Example: "  yt issue links list DV-1 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			links, err := svc.ListLinks(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return runtime.Printer.Print(links)
		},
	}

	var relationship, issue string
	addCmd := &cobra.Command{
		Use:     "add <issue-key>",
		Short:   "Create a link from one issue to another",
		Example: "  yt issue links add DV-1 --relationship relates --issue DV-2",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			link, err := svc.AddLink(cmd.Context(), args[0], relationship, issue)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(link)
		},
	}
	addCmd.Flags().StringVar(&relationship, "relationship", "", "relationship type ID")
	addCmd.Flags().StringVar(&issue, "issue", "", "linked issue key or ID")
	_ = addCmd.MarkFlagRequired("relationship")
	_ = addCmd.MarkFlagRequired("issue")

	cmd.AddCommand(listCmd, addCmd)
	return cmd
}

func newChecklistCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checklist",
		Short: "Manage issue checklist items",
		Long:  "List, add, update, check, uncheck, and delete checklist items on an issue.",
	}

	listCmd := &cobra.Command{
		Use:     "list <issue-key>",
		Short:   "List checklist items",
		Example: "  yt issue checklist list DV-1 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			items, err := svc.ListChecklist(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return runtime.Printer.Print(items)
		},
	}

	var addText, addAssignee, addDeadline string
	var addChecked bool
	addCmd := &cobra.Command{
		Use:   "add <issue-key>",
		Short: "Add a checklist item",
		Example: strings.TrimSpace(`
  yt issue checklist add DV-1 --text "Prepare release notes"
  yt issue checklist add DV-1 --text "QA signoff" --checked`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			items, err := svc.AddChecklistItem(cmd.Context(), args[0], addText, addAssignee, addDeadline, addChecked)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(items)
		},
	}
	addCmd.Flags().StringVar(&addText, "text", "", "checklist item text")
	addCmd.Flags().StringVar(&addAssignee, "assignee", "", "assignee login or UID")
	addCmd.Flags().StringVar(&addDeadline, "deadline", "", "deadline value")
	addCmd.Flags().BoolVar(&addChecked, "checked", false, "create item as checked")
	_ = addCmd.MarkFlagRequired("text")

	var updateText, updateAssignee, updateDeadline string
	var updateChecked bool
	updateCmd := &cobra.Command{
		Use:     "update <issue-key> <item-id>",
		Short:   "Update checklist item fields",
		Example: "  yt issue checklist update DV-1 item-123 --text \"Updated text\"",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			var checked *bool
			if cmd.Flags().Changed("checked") {
				checked = &updateChecked
			}
			item, err := svc.UpdateChecklistItem(cmd.Context(), args[0], args[1], updateText, updateAssignee, updateDeadline, checked)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(item)
		},
	}
	updateCmd.Flags().StringVar(&updateText, "text", "", "updated checklist item text")
	updateCmd.Flags().StringVar(&updateAssignee, "assignee", "", "assignee login or UID")
	updateCmd.Flags().StringVar(&updateDeadline, "deadline", "", "deadline value")
	updateCmd.Flags().BoolVar(&updateChecked, "checked", false, "checked state")

	checkCmd := &cobra.Command{
		Use:     "check <issue-key> <item-id>",
		Short:   "Mark a checklist item as checked",
		Example: "  yt issue checklist check DV-1 item-123",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			item, err := svc.SetChecklistItemChecked(cmd.Context(), args[0], args[1], true)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(item)
		},
	}

	uncheckCmd := &cobra.Command{
		Use:     "uncheck <issue-key> <item-id>",
		Short:   "Mark a checklist item as unchecked",
		Example: "  yt issue checklist uncheck DV-1 item-123",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			item, err := svc.SetChecklistItemChecked(cmd.Context(), args[0], args[1], false)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(item)
		},
	}

	deleteCmd := &cobra.Command{
		Use:     "delete <issue-key> <item-id>",
		Short:   "Delete a checklist item",
		Example: "  yt issue checklist delete DV-1 item-123",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			result, err := svc.DeleteChecklistItem(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			return runtime.Printer.Print(result)
		},
	}

	cmd.AddCommand(listCmd, addCmd, updateCmd, checkCmd, uncheckCmd, deleteCmd)
	return cmd
}

func newQueueCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Inspect Tracker queues",
		Long:  "List queues, inspect queue metadata, and fetch queue-specific field definitions.",
	}

	var perPage, page int
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List queues with pagination controls",
		Example: "  yt queue list --per-page 50 --page 2 --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			queues, err := svc.ListQueues(cmd.Context(), perPage, page)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(queues)
		},
	}
	listCmd.Flags().IntVar(&perPage, "per-page", 100, "page size")
	listCmd.Flags().IntVar(&page, "page", 1, "page number")

	var expand []string
	getCmd := &cobra.Command{
		Use:   "get <queue-key>",
		Short: "Get queue metadata, optionally with expanded sections",
		Example: strings.TrimSpace(`
  yt queue get DV
  yt queue get DV --expand types --expand team --json`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			queue, err := svc.GetQueue(cmd.Context(), args[0], expand)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(queue)
		},
	}
	getCmd.Flags().StringArrayVar(&expand, "expand", nil, "repeatable expand key")

	fieldsCmd := &cobra.Command{
		Use:     "fields <queue-key>",
		Short:   "Get queue fields, including required and readonly markers",
		Example: "  yt queue fields DV --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			fields, err := svc.GetQueueFields(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return runtime.Printer.Print(fields)
		},
	}

	localFieldsCmd := &cobra.Command{
		Use:     "local-fields <queue-key>",
		Short:   "Get queue-local custom fields",
		Example: "  yt queue local-fields DV --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			fields, err := svc.GetQueueLocalFields(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return runtime.Printer.Print(fields)
		},
	}

	cmd.AddCommand(listCmd, getCmd, fieldsCmd, localFieldsCmd)
	return cmd
}

func newFieldCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "field",
		Short: "Inspect Tracker field metadata",
		Long:  "List global Tracker field definitions such as assignee, priority, and project.",
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List global field definitions",
		Example: "  yt field list --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			fields, err := svc.ListGlobalFields(cmd.Context())
			if err != nil {
				return err
			}
			return runtime.Printer.Print(fields)
		},
	}

	cmd.AddCommand(listCmd)
	return cmd
}

func newIssueTypeCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issuetype",
		Short: "Inspect Tracker issue types",
		Long:  "List issue types that can be used when creating or validating Tracker issues.",
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List issue types",
		Example: "  yt issuetype list --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			items, err := svc.ListIssueTypes(cmd.Context())
			if err != nil {
				return err
			}
			return runtime.Printer.Print(items)
		},
	}

	cmd.AddCommand(listCmd)
	return cmd
}

func newStatusCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Inspect Tracker statuses",
		Long:  "List workflow statuses available in Tracker.",
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List statuses",
		Example: "  yt status list --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			items, err := svc.ListStatuses(cmd.Context())
			if err != nil {
				return err
			}
			return runtime.Printer.Print(items)
		},
	}

	cmd.AddCommand(listCmd)
	return cmd
}

func newPriorityCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "priority",
		Short: "Inspect Tracker priorities",
		Long:  "List Tracker priority values that can be used in issue payloads.",
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List priorities",
		Example: "  yt priority list --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			items, err := svc.ListPriorities(cmd.Context())
			if err != nil {
				return err
			}
			return runtime.Printer.Print(items)
		},
	}

	cmd.AddCommand(listCmd)
	return cmd
}

func newEntityCommand(application *app.App, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entity",
		Short: "Manage Tracker entities",
		Long: `List and inspect Tracker entities outside the issue model.

Supported entity types:
  project
  portfolio
  goal`,
	}

	var entityType, query string
	var fields []string
	var perPage, page int
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List project, portfolio, or goal entities with pagination",
		Example: strings.TrimSpace(`
  yt entity list --type project --json
  yt entity list --type goal --per-page 20 --page 3 --json
  yt entity list --type portfolio --query "infra"`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validEntityType(entityType) {
				return fmt.Errorf("invalid entity type %q", entityType)
			}
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			result, err := svc.ListEntities(cmd.Context(), entityType, query, fields, perPage, page)
			if err != nil {
				return err
			}
			return runtime.Printer.Print(result)
		},
	}
	listCmd.Flags().StringVar(&entityType, "type", "project", "entity type: project, portfolio, goal")
	listCmd.Flags().StringVar(&query, "query", "", "search query")
	listCmd.Flags().StringArrayVar(&fields, "field", nil, "repeatable search field key=value")
	listCmd.Flags().IntVar(&perPage, "per-page", 100, "page size")
	listCmd.Flags().IntVar(&page, "page", 1, "page number")

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a single project, portfolio, or goal entity by ID",
		Example: strings.TrimSpace(`
  yt entity get 12345 --type project
  yt entity get 67890 --type goal --json`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validEntityType(entityType) {
				return fmt.Errorf("invalid entity type %q", entityType)
			}
			runtime, svc, err := runtimeFor(cmd, application, opts)
			if err != nil {
				return err
			}
			entity, err := svc.GetEntity(cmd.Context(), entityType, args[0])
			if err != nil {
				return err
			}
			return runtime.Printer.Print(entity)
		},
	}
	getCmd.Flags().StringVar(&entityType, "type", "project", "entity type: project, portfolio, goal")

	cmd.AddCommand(listCmd, getCmd)
	return cmd
}

func validEntityType(v string) bool {
	switch v {
	case "project", "portfolio", "goal":
		return true
	default:
		return false
	}
}

func currentStoreKind(store auth.TokenStore, fallback string) string {
	if named, ok := store.(auth.StoreDescriptor); ok {
		return named.Kind()
	}
	if fallback != "" {
		return fallback
	}
	return "unknown"
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func inferOrgIDFromOAuthUser(user *auth.UserInfo) (string, bool) {
	if user == nil {
		return "", false
	}
	if orgID, ok := inferOrgIDFromIdentity(user.DefaultEmail); ok {
		return orgID, true
	}
	for _, email := range user.Emails {
		if orgID, ok := inferOrgIDFromIdentity(email); ok {
			return orgID, true
		}
	}
	return inferOrgIDFromIdentity(user.Login)
}

func inferOrgIDFromUser(user *model.User) (string, bool) {
	if user == nil {
		return "", false
	}
	return inferOrgIDFromIdentity(firstNonEmptyString(user.Email, user.Login))
}

func inferOrgIDFromIdentity(identity string) (string, bool) {
	identity = strings.TrimSpace(identity)
	if strings.HasSuffix(strings.ToLower(identity), "@loov.team") {
		return loovTeamOrgID, true
	}
	return "", false
}

func readToken(input io.Reader) (string, error) {
	reader, ok := input.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(input)
	}
	token, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		if strings.TrimSpace(token) == "" {
			return "", err
		}
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("empty OAuth token input")
	}
	return token, nil
}

func isRequiredSchemaField(field model.Field) bool {
	if field.Schema == nil {
		return false
	}
	required, ok := field.Schema["required"].(bool)
	return ok && required
}
