package main

// from prometheus alertmanager config package

// Config is the top-level configuration for Alertmanager's config files.
type Config struct {
	Global       *GlobalConfig  `yaml:"global,omitempty" json:"global,omitempty"`
	Route        *Route         `yaml:"route,omitempty" json:"route,omitempty"`
	InhibitRules []*InhibitRule `yaml:"inhibit_rules,omitempty" json:"inhibit_rules,omitempty"`
	Receivers    []*Receiver    `yaml:"receivers,omitempty" json:"receivers,omitempty"`
	Templates    []string       `yaml:"templates" json:"templates"`
}

// GlobalConfig defines configuration parameters that are valid globally
// unless overwritten.
type GlobalConfig struct {
	// ResolveTimeout is the time after which an alert is declared resolved
	// if it has not been updated.
	ResolveTimeout string `yaml:"resolve_timeout,omitempty" json:"resolve_timeout"`

	SMTPFrom         string `yaml:"smtp_from,omitempty" json:"smtp_from"`
	SMTPSmarthost    string `yaml:"smtp_smarthost,omitempty" json:"smtp_smarthost"`
	SMTPAuthUsername string `yaml:"smtp_auth_username,omitempty" json:"smtp_auth_username"`
	SMTPAuthPassword string `yaml:"smtp_auth_password,omitempty" json:"smtp_auth_password"`
	SMTPAuthSecret   string `yaml:"smtp_auth_secret,omitempty" json:"smtp_auth_secret"`
	SMTPAuthIdentity string `yaml:"smtp_auth_identity,omitempty" json:"smtp_auth_identity"`
	SMTPRequireTLS   bool   `yaml:"smtp_require_tls" json:"smtp_require_tls"`
	SlackAPIURL      string `yaml:"slack_api_url,omitempty" json:"slack_api_url"`
	PagerdutyURL     string `yaml:"pagerduty_url,omitempty" json:"pagerduty_url"`
	HipchatURL       string `yaml:"hipchat_url,omitempty" json:"hipchat_url"`
	HipchatAuthToken string `yaml:"hipchat_auth_token,omitempty" json:"hipchat_auth_token"`
	OpsGenieAPIHost  string `yaml:"opsgenie_api_host,omitempty" json:"opsgenie_api_host"`
	VictorOpsAPIURL  string `yaml:"victorops_api_url,omitempty" json:"victorops_api_url"`
}

// A Route is a node that contains definitions of how to handle alerts.
type Route struct {
	Receiver string   `yaml:"receiver,omitempty" json:"receiver,omitempty"`
	GroupBy  []string `yaml:"group_by,omitempty" json:"group_by,omitempty"`

	Match    map[string]string `yaml:"match,omitempty" json:"match,omitempty"`
	MatchRE  map[string]string `yaml:"match_re,omitempty" json:"match_re,omitempty"`
	Continue bool              `yaml:"continue,omitempty" json:"continue,omitempty"`
	Routes   []*Route          `yaml:"routes,omitempty" json:"routes,omitempty"`

	GroupWait      *string `yaml:"group_wait,omitempty" json:"group_wait,omitempty"`
	GroupInterval  *string `yaml:"group_interval,omitempty" json:"group_interval,omitempty"`
	RepeatInterval *string `yaml:"repeat_interval,omitempty" json:"repeat_interval,omitempty"`
}

// InhibitRule defines an inhibition rule that mutes alerts that match the
// target labels if an alert matching the source labels exists.
// Both alerts have to have a set of labels being equal.
type InhibitRule struct {
	// SourceMatch defines a set of labels that have to equal the given
	// value for source alerts.
	SourceMatch map[string]string `yaml:"source_match,omitempty" json:"source_match"`
	// SourceMatchRE defines pairs like SourceMatch but does regular expression
	// matching.
	SourceMatchRE map[string]string `yaml:"source_match_re,omitempty" json:"source_match_re"`
	// TargetMatch defines a set of labels that have to equal the given
	// value for target alerts.
	TargetMatch map[string]string `yaml:"target_match,omitempty" json:"target_match"`
	// TargetMatchRE defines pairs like TargetMatch but does regular expression
	// matching.
	TargetMatchRE map[string]string `yaml:"target_match_re,omitempty" json:"target_match_re"`
	// A set of labels that must be equal between the source and target alert
	// for them to be a match.
	Equal string `yaml:"equal,omitempty" json:"equal,omitempty"`
}

// Receiver configuration provides configuration on how to contact a receiver.
type Receiver struct {
	// A unique identifier for this receiver.
	Name string `yaml:"name" json:"name"`

	EmailConfigs []*EmailConfig `yaml:"email_configs,omitempty" json:"email_configs,omitempty"`
	//PagerdutyConfigs []*PagerdutyConfig `yaml:"pagerduty_configs,omitempty" json:"pagerduty_configs,omitempty"`
	//HipchatConfigs   []*HipchatConfig   `yaml:"hipchat_configs,omitempty" json:"hipchat_configs,omitempty"`
	//SlackConfigs     []*SlackConfig     `yaml:"slack_configs,omitempty" json:"slack_configs,omitempty"`
	WebhookConfigs []*WebhookConfig `yaml:"webhook_configs,omitempty" json:"webhook_configs,omitempty"`
	//OpsGenieConfigs  []*OpsGenieConfig  `yaml:"opsgenie_configs,omitempty" json:"opsgenie_configs,omitempty"`
	//PushoverConfigs  []*PushoverConfig  `yaml:"pushover_configs,omitempty" json:"pushover_configs,omitempty"`
	//VictorOpsConfigs []*VictorOpsConfig `yaml:"victorops_configs,omitempty" json:"victorops_configs,omitempty"`
}

// NotifierConfig contains base options common across all notifier configurations.
type NotifierConfig struct {
	VSendResolved bool `yaml:"send_resolved" json:"send_resolved"`
}

// EmailConfig configures notifications via mail.
type EmailConfig struct {
	NotifierConfig `yaml:",inline" json:",inline"`

	// Email address to notify.
	To           string            `yaml:"to" json:"to"`
	From         string            `yaml:"from,omitempty" json:"from"`
	Smarthost    string            `yaml:"smarthost,omitempty" json:"smarthost,omitempty"`
	AuthUsername string            `yaml:"auth_username,omitempty" json:"auth_username"`
	AuthPassword string            `yaml:"auth_password,omitempty" json:"auth_password"`
	AuthSecret   string            `yaml:"auth_secret,omitempty" json:"auth_secret"`
	AuthIdentity string            `yaml:"auth_identity,omitempty" json:"auth_identity"`
	Headers      map[string]string `yaml:"headers,omitempty" json:"headers"`
	HTML         string            `yaml:"html,omitempty" json:"html"`
	RequireTLS   *bool             `yaml:"require_tls,omitempty" json:"require_tls,omitempty"`
}

// WebhookConfig configures notifications via a generic webhook.
type WebhookConfig struct {
	NotifierConfig `yaml:",inline" json:",inline"`

	// URL to send POST request to.
	URL string `yaml:"url" json:"url"`
}
