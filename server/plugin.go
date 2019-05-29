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
const usage = `Usage: s/{text to be replaced}/{new text}`

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
	return p.checkServerVersion()
}

func splitAndValidateInput(message string) ([]string, error) {

	input := strings.TrimSpace(strings.TrimPrefix(message, "s/"))

	if input == "" {
		return nil, errors.New("No input")
	}

	strs := strings.Split(input, "/")

	if len(strs) < 2 || len(strs[0]) < 1 || len(strs[1]) < 1 {
		return nil, errors.New("Bad user input")
	}

	return strs, nil
}

// MessageWillBePosted parses every post. If our s/ command is present, it replaces the last post.
func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {

	//notification that will be sent as an ephemeral post
	notification := &model.Post{ChannelId: post.ChannelId, CreateAt: model.GetMillis()}

	//Validate input
	oldAndNew, err := splitAndValidateInput(post.Message)

	//if no valid input, just publish post normally
	if err != nil {
		return nil, ""
	}

	old := oldAndNew[0]

	new := oldAndNew[1]

	//Get user data
	user, appErr := p.API.GetUser(post.UserId)

	if appErr != nil {
		return nil, ""
	}

	//Find channel to get access to teamId
	ch, appErr := p.API.GetChannel(post.ChannelId)

	if appErr != nil {
		return nil, ""
	}

	// find posts by user name
	searchParams := model.ParseSearchParams("from:"+user.Username, 0)

	posts, appErr := p.API.SearchPostsInTeam(ch.TeamId, searchParams)

	if appErr != nil {
		return nil, ""
	}

	if len(posts) < 1 {
		notification.Message = `s/ Command: No previous post to be replaced.`
		p.API.SendEphemeralPost(user.Id, notification)
		return nil, ""
	}

	lastPost := posts[0]

	lastPost.Message = strings.ReplaceAll(lastPost.Message, old, new)

	_, appErr = p.API.UpdatePost(lastPost)

	if appErr != nil {
		return nil, ""
	}

	notification.Message = `s/ Replaced "` + old + `" for "` + new + `"`
	p.API.SendEphemeralPost(user.Id, notification)

	return nil, "plugin.message_will_be_posted.dismiss_post"
}
