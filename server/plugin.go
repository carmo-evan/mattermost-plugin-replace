package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/blang/semver"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const minServerVersion = "5.10.0" // dependent on method SearchPostsInTeam

type Plugin struct {
	plugin.MattermostPlugin

	router *mux.Router

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Mattermost-User-Id") == "" {
		http.Error(w, "please log in", http.StatusForbidden)
		return
	}

	p.router.ServeHTTP(w, r)
}

// checkServerVersion checks Mattermost Server has at least the required version
func (p *Plugin) checkServerVersion() error {
	serverVersion, err := semver.Parse(p.API.GetServerVersion())
	if err != nil {
		return errors.Wrap(err, "failed to parse server version")
	}

	r := semver.MustParseRange(">=" + minServerVersion)
	if !r(serverVersion) {
		return fmt.Errorf("this plugin requires Mattermost v%s or later", minServerVersion)
	}

	return nil
}

//OnActivate registers the /s command with the API
func (p *Plugin) OnActivate() error {
	if err := p.checkServerVersion(); err != nil {
		return err
	}
	return p.API.RegisterCommand(&model.Command{
		Trigger:          "s",
		AutoComplete:     true,
		AutoCompleteDesc: "Finds and replaces text in your last post.",
	})
}

func splitAndValidateInput(command string) ([]string, error) {

	input := strings.TrimSpace(strings.TrimPrefix(command, "/s"))

	if input == "" {
		return nil, errors.New("No input")
	}

	strs := strings.Split(input, "/")

	if len(strs) < 2 || len(strs[0]) < 1 || len(strs[1]) < 1 {
		return nil, errors.New("Bad user input")
	}

	return strs, nil
}

//ExecuteCommand parses the input,
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	//Validate input

	oldAndNew, err := splitAndValidateInput(args.Command)

	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         `Usage: /s {text to be replaced}/{new text}`,
		}, nil
	}

	old := oldAndNew[0]

	new := oldAndNew[1]

	//Get username
	user, appErr := p.API.GetUser(args.UserId)
	if err != nil {
		return nil, appErr
	}

	//Find last post

	searchParams := model.ParseSearchParams("from:"+user.Username, 0)

	posts, appErr := p.API.SearchPostsInTeam(args.TeamId, searchParams)

	if appErr != nil {
		return nil, appErr
	}

	if len(posts) < 1 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         `No previous post to be replaced.`,
		}, nil
	}

	lastPost := posts[0]

	lastPost.Message = strings.ReplaceAll(lastPost.Message, old, new)

	_, appErr = p.API.UpdatePost(lastPost)

	if appErr != nil {
		return nil, appErr
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         `Replaced "` + old + `" for "` + new + `"`,
	}, nil
}
