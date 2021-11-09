package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ereti/ofelia/core"
)

var (
	discordPayloadVar = "payload_json"
)

// SlackConfig configuration for the Slack middleware
type DiscordConfig struct {
	DiscordWebhook     string `gcfg:"discord-webhook" mapstructure:"discord-webhook"`
	DiscordOnlyOnError bool   `gcfg:"discord-only-on-error" mapstructure:"discord-only-on-error"`
}

// NewSlack returns a Slack middleware if the given configuration is not empty
func NewDiscord(c *DiscordConfig) core.Middleware {
	var m core.Middleware
	if !IsEmpty(c) {
		m = &Discord{*c}
	}

	return m
}

// Slack middleware calls to a Slack input-hook after every execution of a job
type Discord struct {
	DiscordConfig
}

// ContinueOnStop return allways true, we want alloways report the final status
func (m *Discord) ContinueOnStop() bool {
	return true
}

// Run sends a message to the slack channel, its close stop the exection to
// collect the metrics
func (m *Discord) Run(ctx *core.Context) error {
	err := ctx.Next()
	ctx.Stop(err)

	if ctx.Execution.Failed || !m.DiscordOnlyOnError {
		m.pushMessage(ctx)
	}

	return err
}

func (m *Discord) pushMessage(ctx *core.Context) {
	values := make(url.Values, 0)
	content, _ := json.Marshal(m.buildMessage(ctx))
	values.Add(discordPayloadVar, string(content))

	r, err := http.PostForm(m.DiscordWebhook, values)
	if err != nil {
		ctx.Logger.Errorf("Slack error calling %q error: %q", m.DiscordWebhook, err)
	} else if r.StatusCode != 200 {
		ctx.Logger.Errorf("Slack error non-200 status code calling %q", m.DiscordWebhook)
	}
}

func (m *Discord) buildMessage(ctx *core.Context) *discordMessage {
	msg := &discordMessage{
		Content: "embeds",
	}

	if ctx.Execution.Failed {
		msg.Embeds = append(msg.Embeds, discordEmbed{
			Title: "Job Failed",
			Description: fmt.Sprintf("Job `$q` failed, took `%s`", ctx.Job.GetName(), ctx.Execution.Duration),
			Color: 0xFF0000,
		})

	} else if ctx.Execution.Skipped {
		msg.Embeds = append(msg.Embeds, discordEmbed{
			Title: "Job Skipped",
			Description: fmt.Sprintf("Job `$q` was skipped", ctx.Job.GetName()),
			Color: 0x555555,
		})
	} else {
		msg.Embeds = append(msg.Embeds, discordEmbed{
			Title: "Job Succeeded",
			Description: fmt.Sprintf("Job `$q` succeeded, took `%s`", ctx.Job.GetName(), ctx.Execution.Duration),
			Color: 0x00FF00,
		})
	}



	return msg
}

type discordMessage struct {
	Content     string          `json:"content,omitempty"`
	Embeds		[]discordEmbed	`json:"embeds,omitempty"`
}

type discordEmbed struct {
	Title 		string 	`json:"title,omitempty"`
	Description	string	`json:"description,omitempty"`
	Color		int64 	`json:"color,omitempty"`
	Fields		[]discordEmbedField 	`json:"fields,omitempty"`
	Footer		discordEmbedFooter		`json:"footer,omitempty"`
}

type discordEmbedField struct {
	Name 	string	`json:"name"`
	Value	string	`json:"value"`
	Inline	bool	`json:"inline,omitempty"`
}

type discordEmbedFooter struct {
	Text 	string `json:"text"`
}
