package middlewares

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	. "gopkg.in/check.v1"
)

type SuiteSlack struct {
	BaseSuite
}

var _ = Suite(&SuiteDiscord{})

func (s *SuiteDiscord) TestNewDiscordEmpty(c *C) {
	c.Assert(NewDiscord(&SlackConfig{}), IsNil)
}

func (s *SuiteDiscord) TestRunSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m discordMessage
		json.Unmarshal([]byte(r.FormValue(discordPayloadVar)), &m)
		c.Assert(m.Embeds[0].Title, Equals, "Job Succeeded")
	}))

	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(nil)

	m := NewDiscord(&DiscordConfig{DiscordWebhook: ts.URL})
	c.Assert(m.Run(s.ctx), IsNil)
}

func (s *SuiteDiscord) TestRunSuccessFailed(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m discordMessage
		json.Unmarshal([]byte(r.FormValue(discordPayloadVar)), &m)
		c.Assert(m.Embeds[0].Title, Equals, "Job Failed")
	}))

	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(errors.New("foo"))

	m := NewDiscord(&DiscordConfig{DiscordWebhook: ts.URL})
	c.Assert(m.Run(s.ctx), IsNil)
}

func (s *SuiteDiscord) TestRunSuccessOnError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(true, Equals, false)
	}))

	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(nil)

	m := NewDiscord(&DiscordConfig{DiscordWebhook: ts.URL, DiscordOnlyOnError: true})
	c.Assert(m.Run(s.ctx), IsNil)
}
