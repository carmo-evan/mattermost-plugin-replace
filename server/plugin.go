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

const (
	minServerVersion  string = "5.10.0" // dependent on method SearchPostsInTeam
	usage             string = `Usage: s/{text to be replaced}/{new text}`
	noPostsFoundError string = "`s/ Command: No previous post to be replaced.`"
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

func (p *Plugin) getLastPost(user *model.User, teamId string, rootId string) (*model.Post, string) {
	// if we have a rootId, it means we are in a thread.
	if rootId != "" {
		postThread, err := p.API.GetPostThread(rootId)
		if err != nil {
			return nil, err.Error()
		}

		//HACK: adding Orders to the postThread to be able to sort it
		// because API.GetPostThread returns a postList without the Orders
		for _, post := range postThread.Posts {
			postThread.AddOrder(post.Id)
		}

		postThread.SortByCreateAt()

		for _, key := range postThread.Order {
			post := postThread.Posts[key]
			if post.UserId == user.Id {
				return post, ""
			}
		}

		return nil, noPostsFoundError
	}

	searchParams := model.ParseSearchParams("from:"+user.Username, 0)

	posts, err := p.API.SearchPostsInTeam(teamId, searchParams)

	if err != nil {
		return nil, err.Error()
	}

	if len(posts) < 1 {
		return nil, noPostsFoundError
	}

	return posts[0], ""
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
	lastPost, errId := p.getLastPost(user, ch.TeamId, post.RootId)
	if errId != "" {
		notification.Message = errId
		p.API.SendEphemeralPost(user.Id, notification)
		return nil, ""
	}

	lastPost.Message = strings.ReplaceAll(lastPost.Message, old, new)

	_, appErr = p.API.UpdatePost(lastPost)

	if appErr != nil {
		return nil, ""
	}

	notification.Message = `s/ Replaced "` + old + `" for "` + new + `"`
	p.API.SendEphemeralPost(user.Id, notification)

	return nil, "plugin.message_will_be_posted.dismiss_post"
}
