package main

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"

	// "github.com/blang/semver"

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
	expectedResponse string
	shouldFail       bool
}

type testAPIConfig struct {
	User  *model.User
	Posts []*model.Post
	Post  *model.Post
}

func setupAPI(api *plugintest.API, config *testAPIConfig) {

	api.On("GetServerVersion").Return(minServerVersion)

	api.On("GetUser", mock.Anything).Return(config.User, nil)

	api.On("SearchPostsInTeam", mock.Anything, mock.Anything).Return(config.Posts, nil)

	api.On("UpdatePost", mock.Anything).Return(config.Post, nil)

	api.On("RegisterCommand", mock.Anything).Return(nil)
}

//TestExecuteCommand mocks the API calls (by using the private method setupAPI) and validates
//the inputs given,
func TestExecuteCommand(t *testing.T) {

	cases := []testCase{
		{"message to bee replaced", "/s bee/be", `Replaced "bee" for "be"`, false},
		{"baaad input", "/s bad", "Usage: /s {text to be replaced}/{new text}", true},
		{"more baaad input", "/s baaad/", "Usage: /s {text to be replaced}/{new text}", true},
	}

	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {

			c := &plugin.Context{}
			args := &model.CommandArgs{
				UserId:  "testUserId",
				Command: tc.command,
				TeamId:  "testTeamId",
			}

			api := &plugintest.API{}

			defer api.AssertExpectations(t)

			config := &testAPIConfig{
				User:  &model.User{},
				Posts: []*model.Post{&model.Post{}},
				Post:  &model.Post{},
			}

			//needs to test input before setting API expectations
			if _, err := splitAndValidateInput(tc.command); err != nil && tc.shouldFail {
				assert.NotNil(t, err)
				return
			}

			setupAPI(api, config)

			p := setupTestPlugin(t, api)

			p.OnActivate()

			cr, err := p.ExecuteCommand(c, args)

			assert.Nil(t, err)
			assert.Equal(t, tc.expectedResponse, cr.Text)
		})
	}
}

func TestPluginOnActivate(t *testing.T) {

	api := &plugintest.API{}

	api.On("GetServerVersion").Return(minServerVersion)

	defer api.AssertExpectations(t)

	api.On("RegisterCommand", &model.Command{
		Trigger:          "s",
		AutoComplete:     true,
		AutoCompleteDesc: "Finds and replaces text in your last post.",
	}).Return(nil)

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
