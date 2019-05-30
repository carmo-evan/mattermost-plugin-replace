package main

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTestPlugin(t *testing.T, api *plugintest.API) *Plugin {
	p := &Plugin{}

	p.SetAPI(api)

	return p
}

type testCase struct {
	message          string
	command          string
	rootId           string
	expectedResponse string
	shouldFail       bool
}

type testAPIConfig struct {
	User    *model.User
	Posts   []*model.Post
	Post    *model.Post
	Channel *model.Channel
}

func setupAPI(api *plugintest.API, config *testAPIConfig) {

	api.On("GetServerVersion").Return(minServerVersion)

	api.On("GetUser", mock.Anything).Return(config.User, nil)

	api.On("GetChannel", mock.Anything).Return(config.Channel, nil)

	api.On("SearchPostsInTeam", mock.Anything, mock.Anything).Return(config.Posts, nil)

	api.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(nil)

	api.On("UpdatePost", mock.Anything).Return(config.Post, nil)

}

// TestExecuteCommand mocks the API calls (by using the private method setupAPI) and validates the inputs given
func TestExecuteCommand(t *testing.T) {

	cases := []testCase{
		{"message to bee replaced", "s/bee/be", "", `Replaced "bee" for "be"`, false},
		{"message to bee replaced", "s/bee/be", "123", `Replaced "bee" for "be"`, false},
		{"baaad input", "s/bad", "", usage, true},
		{"more baaad input", "s/baaad/", "", usage, true},
	}

	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {

			c := &plugin.Context{}
			post := &model.Post{
				UserId:    "testUserId",
				Message:   tc.command,
				ChannelId: "testChannelId",
			}

			api := &plugintest.API{}

			defer api.AssertExpectations(t)

			config := &testAPIConfig{
				User:    &model.User{},
				Posts:   []*model.Post{&model.Post{}},
				Post:    &model.Post{},
				Channel: &model.Channel{},
			}

			//needs to test input before setting API expectations
			if _, err := splitAndValidateInput(tc.command); err != nil && tc.shouldFail {
				assert.NotNil(t, err)
				return
			}

			setupAPI(api, config)

			p := setupTestPlugin(t, api)

			p.OnActivate()

			returnedPost, err := p.MessageWillBePosted(c, post)
			assert.Nil(t, returnedPost)
			assert.Equal(t, err, "plugin.message_will_be_posted.dismiss_post")
		})
	}
}

func TestPluginOnActivate(t *testing.T) {

	api := &plugintest.API{}

	api.On("GetServerVersion").Return(minServerVersion)

	defer api.AssertExpectations(t)

	p := setupTestPlugin(t, api)

	err := p.OnActivate()

	assert.Nil(t, err)
}

func TestServeHTTP(t *testing.T) {
	assert := assert.New(t)
	plugin := Plugin{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	plugin.ServeHTTP(nil, w, r)

	result := w.Result()
	assert.NotNil(result)
	bodyBytes, err := ioutil.ReadAll(result.Body)
	assert.Nil(err)
	bodyString := string(bodyBytes)

	assert.Equal("please log in\n", bodyString)
}
