package main

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/mux"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

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

//OnActivate registers the /s command with the API
func (p *Plugin) OnActivate() error {
	return p.API.RegisterCommand(&model.Command{
		Trigger:          "s",
		AutoComplete:     true,
		AutoCompleteDesc: "Finds and replaces text in your last post.",
	})
}

//ExecuteCommand parses the input,
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	//Validate input

	input := strings.TrimSpace(strings.TrimPrefix(args.Command, "/s"))

	if input == "" {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         `Usage: s/ {text to be replaced}/{new text}`,
		}, nil
	}

	oldAndNew := strings.Split(input, "/")

	if len(oldAndNew) < 2 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         `Usage: s/ {text to be replaced}/{new text}`,
		}, nil
	}

	old := oldAndNew[0]

	new := oldAndNew[1]

	//Get username
	user, err := p.API.GetUser(args.UserId)
	if err != nil {
		return nil, err
	}

	//Find last post

	searchParams := model.ParseSearchParams("from:"+user.Username, 0)

	posts, err := p.API.SearchPostsInTeam(args.TeamId, searchParams)
	if err != nil {
		return nil, err
	}

	lastPost := posts[0]

	lastPost.Message = strings.ReplaceAll(lastPost.Message, old, new)

	p.API.UpdatePost(lastPost)

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         "Replaced " + old + " for " + new,
	}, nil
}
