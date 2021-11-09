package middlewares

import (
	"encoding/json"
	"fmt"
	"bytes"
	"net/http"
	"mime/multipart"
	"net/textproto"

	"github.com/ereti/ofelia/core"
)

var (
	discordPayloadVar = "payload_json"
)

// SlackConfig configuration for the Slack middleware
type DiscordConfig struct {
	DiscordWebhook     string `gcfg:"discord-webhook" mapstructure:"discord-webhook"`
	DiscordOnlyOnError bool   `gcfg:"discord-only-on-error" mapstructure:"discord-only-on-error"`
	DiscordAttachOutput bool `gcfg:"discord-attach-output" mapstructure:"discord-attach-output"`
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
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	jsonPartHeader := textproto.MIMEHeader{}
	jsonDisp := fmt.Sprintf("form-data; name=\"%s\"", discordPayloadVar)
	jsonPartHeader.Add("Content-Disposition", jsonDisp)
	jsonPartHeader.Add("Content-Type", "application/json")
	jsonPart, err := writer.CreatePart(jsonPartHeader)

	if err != nil {
		ctx.Logger.Errorf("Error creating json part for Discord")
		return
	}

	msg := m.buildMessage(ctx)

	content, _ := json.Marshal(msg)
	jsonPart.Write(content)

	if ctx.Execution.Skipped || !m.DiscordAttachOutput {
		writer.Close()

		r, err := http.Post(m.DiscordWebhook, writer.FormDataContentType(), body)
		if err != nil {
			ctx.Logger.Errorf("Discord error calling %q error: %q", m.DiscordWebhook, err)
		} else if r.StatusCode < 200 || r.StatusCode > 299 {
			ctx.Logger.Errorf("Discord error non-200-range status code calling %q (got %d)", m.DiscordWebhook, r.StatusCode)
		}
	}

	stdoutPartHeader := textproto.MIMEHeader{}
	stdoutDisp := fmt.Sprintf("form-data; name=\"files[0]\"; filename=\"%s\"", msg.Attachments[0].Filename)
	stdoutPartHeader.Add("Content-Disposition", jsonDisp)
	stdoutPartHeader.Add("Content-Type", "text/plain")
	stdoutPart, err := writer.CreatePart(stdoutPartHeader)

	if err != nil {
		ctx.Logger.Errorf("Error creating stdout part for Discord")
		return
	}

	stdoutPart.Write(ctx.Execution.OutputStream.Bytes())

	stderrPartHeader := textproto.MIMEHeader{}
	stderrDisp := fmt.Sprintf("form-data; name=\"files[1]\"; filename=\"%s\"", msg.Attachments[1].Filename)
	stderrPartHeader.Add("Content-Disposition", jsonDisp)
	stderrPartHeader.Add("Content-Type", "text/plain")
	stderrPart, err := writer.CreatePart(stderrPartHeader)

	if err != nil {
		ctx.Logger.Errorf("Error creating stderr part for Discord")
		return
	}

	stderrPart.Write(ctx.Execution.ErrorStream.Bytes())

	writer.Close()

	r, err := http.Post(m.DiscordWebhook, writer.FormDataContentType(), &body)
	if err != nil {
		ctx.Logger.Errorf("Discord error calling %q error: %q", m.DiscordWebhook, err)
	} else if r.StatusCode < 200 || r.StatusCode > 299 {
		ctx.Logger.Errorf("Discord error non-200-range status code calling %q (got %d)", m.DiscordWebhook, r.StatusCode)
	}
}

func (m *Discord) buildMessage(ctx *core.Context) *discordMessage {
	msg := &discordMessage{
	}

	if ctx.Execution.Skipped {
		msg.Embeds = append(msg.Embeds, discordEmbed{
			Title: "Job Skipped",
			Description: fmt.Sprintf("Job `%q` was skipped", ctx.Job.GetName()),
			Color: 0x555555,
		})

		return msg
	}


	if ctx.Execution.Failed {
		msg.Embeds = append(msg.Embeds, discordEmbed{
			Title: "Job Failed",
			Description: fmt.Sprintf("Job `%q` failed, took `%s`", ctx.Job.GetName(), ctx.Execution.Duration),
			Color: 0xFF0000,
		})

	} else {
		msg.Embeds = append(msg.Embeds, discordEmbed{
			Title: "Job Succeeded",
			Description: fmt.Sprintf("Job `%q` succeeded, took `%s`", ctx.Job.GetName(), ctx.Execution.Duration),
			Color: 0x00FF00,
		})
	}

	if m.DiscordAttachOutput {
		name := fmt.Sprintf(
			"%s_%s",
			ctx.Execution.Date.Format("20060102_150405"), ctx.Job.GetName(),
		)

		msg.Attachments = append(msg.Attachments, discordAttachment{
			Id: 0,
			Filename: fmt.Sprintf("%s.stdout.log", name),
			Description: "Standard out log for the job.",
		})

		msg.Attachments = append(msg.Attachments, discordAttachment{
			Id: 1,
			Filename: fmt.Sprintf("%s.stderr.log", name),
			Description: "Standard error log for the job.",
		})
	}

	return msg
}

type discordMessage struct {
	Content     string          `json:"content,omitempty"`
	Embeds		[]discordEmbed	`json:"embeds,omitempty"`
	Attachments []discordAttachment `json"attachments,omitempty`
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

type discordAttachment struct {
	Id 	string	`json:"id"`
	Filename string `json:"filename"`
	Description string `json:"description,omitempty"`
}

type discordEmbedFooter struct {
	Text 	string `json:"text"`
}
