package monitoring

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/grafana-tools/sdk"
)

// Container describes a Docker container to be observed.
//
// These correspond to dashboards in Grafana.
type Container struct {
	// Name of the Docker container, e.g. "syntect-server".
	Name string

	// Title of the Docker container, e.g. "Syntect Server".
	Title string

	// Description of the Docker container. It should describe what the container
	// is responsible for, so that the impact of issues in it is clear.
	Description string

	// Groups of observable information about the container.
	Groups []Group
}

func (c *Container) validate() error {
	if !isValidGrafanaUID(c.Name) {
		return fmt.Errorf("Name must be lowercase alphanumeric + dashes; found \"%s\"", c.Name)
	}
	if c.Title != strings.Title(c.Title) {
		return fmt.Errorf("Title must be in Title Case; found \"%s\" want \"%s\"", c.Title, strings.Title(c.Title))
	}
	if c.Description != withPeriod(c.Description) || c.Description != upperFirst(c.Description) {
		return fmt.Errorf("Description must be sentence starting with an uppercas eletter and ending with period; found \"%s\"", c.Description)
	}
	for i, g := range c.Groups {
		if err := g.validate(); err != nil {
			return fmt.Errorf("Group %d %q: %v", i, g.Title, err)
		}
	}
	return nil
}

// renderDashboard generates the Grafana renderDashboard for this container.
func (c *Container) renderDashboard() *sdk.Board {
	board := sdk.NewBoard(c.Title)
	board.Version = uint(rand.Uint32())
	board.UID = c.Name
	board.ID = 0
	board.Timezone = "utc"
	board.Timepicker.RefreshIntervals = []string{"5s", "10s", "30s", "1m", "5m", "15m", "30m", "1h", "2h", "1d"}
	board.Time.From = "now-6h"
	board.Time.To = "now"
	board.SharedCrosshair = true
	board.AddTags("builtin")
	board.Templating.List = []sdk.TemplateVar{
		{
			Label:      "Filter alert level",
			Name:       "alert_level",
			AllValue:   ".*",
			Current:    sdk.Current{Text: "all", Value: "$__all"},
			IncludeAll: true,
			Options: []sdk.Option{
				{Text: "all", Value: "$__all", Selected: true},
				{Text: "critical", Value: "critical"},
				{Text: "warning", Value: "warning"},
			},
			Query: "critical,warning",
			Type:  "custom",
		},
	}
	board.Editable = false

	description := sdk.NewText("")
	description.Title = "" // Removes vertical space the title would otherwise take up
	setPanelSize(description, 24, 3)
	description.TextPanel.Mode = "html"
	description.TextPanel.Content = fmt.Sprintf(`
	<div style="text-align: left;">
	  <img src="https://sourcegraphstatic.com/sourcegraph-logo-light.png" style="height:30px; margin:0.5rem"></img>
	  <div style="margin-left: 1rem; margin-top: 0.5rem; font-size: 20px;"><span style="color: #8e8e8e">%s:</span> %s <a style="font-size: 15px" target="_blank" href="https://docs.sourcegraph.com/dev/background-information/architecture">(⧉ architecture diagram)</a></span>
	</div>
	`, c.Name, c.Description)
	board.Panels = append(board.Panels, description)

	alertsDefined := sdk.NewTable("Alerts defined")
	setPanelSize(alertsDefined, 9, 5)
	setPanelPos(alertsDefined, 0, 3)
	alertsDefined.TablePanel.Sort = &sdk.Sort{Desc: true, Col: 4}
	alertsDefined.TablePanel.Styles = []sdk.ColumnStyle{
		{
			Pattern: "Time",
			Type:    "hidden",
		},
		{
			Pattern: "level",
			Type:    "hidden",
		},
		{
			Pattern: "_01_level",
			Alias:   stringPtr("level"),
		},
		{
			Pattern:     "Value",
			Alias:       stringPtr("firing?"),
			ColorMode:   stringPtr("row"),
			Colors:      &[]string{"rgba(50, 172, 45, 0.97)", "rgba(237, 129, 40, 0.89)", "rgba(245, 54, 54, 0.9)"},
			Thresholds:  &[]string{"0.99999", "1"},
			Type:        "string",
			MappingType: 1,
			ValueMaps: []sdk.ValueMap{
				{TextType: "false", Value: "0"},
				{TextType: "true", Value: "1"},
			},
		},
	}
	alertsDefined.AddTarget(&sdk.Target{
		Expr:    fmt.Sprintf(`label_replace(sum(max by (level,service_name,name,description)(alert_count{service_name="%s",name!="",level=~"$alert_level"})) by (level,description), "_01_level", "$1", "level", "(.*)")`, c.Name),
		Format:  "table",
		Instant: true,
	})
	board.Panels = append(board.Panels, alertsDefined)

	alertsFiring := sdk.NewGraph("Alerts firing")
	setPanelSize(alertsFiring, 15, 5)
	setPanelPos(alertsFiring, 9, 3)
	alertsFiring.GraphPanel.Legend.Show = true
	alertsFiring.GraphPanel.Fill = 1
	alertsFiring.GraphPanel.Bars = true
	alertsFiring.GraphPanel.NullPointMode = "null"
	alertsFiring.GraphPanel.Pointradius = 2
	alertsFiring.GraphPanel.AliasColors = map[string]string{}
	alertsFiring.GraphPanel.Xaxis = sdk.Axis{
		Show: true,
	}
	alertsFiring.GraphPanel.Yaxes = []sdk.Axis{
		{
			Decimals: 0,
			Format:   "short",
			LogBase:  1,
			Max:      sdk.NewFloatString(1),
			Min:      sdk.NewFloatString(0),
			Show:     true,
		},
		{
			Format:  "short",
			LogBase: 1,
			Show:    true,
		},
	}
	alertsFiring.AddTarget(&sdk.Target{
		Expr:         fmt.Sprintf(`sum by (service_name,level,name)(max by (level,service_name,name,description)(alert_count{service_name="%s",name!="",level=~"$alert_level"}) >= 1)`, c.Name),
		LegendFormat: "{{level}}: {{name}}",
	})
	board.Panels = append(board.Panels, alertsFiring)

	baseY := 8
	offsetY := baseY
	for groupIndex, group := range c.Groups {
		// Non-general groups are shown as collapsible panels.
		var rowPanel *sdk.Panel
		if group.Title != "General" {
			rowPanel = &sdk.Panel{RowPanel: &sdk.RowPanel{}}
			rowPanel.OfType = sdk.RowType
			rowPanel.Type = "row"
			rowPanel.Title = group.Title
			offsetY++
			setPanelPos(rowPanel, 0, offsetY)
			rowPanel.Collapsed = group.Hidden
			rowPanel.Panels = []sdk.Panel{} // cannot be null
			board.Panels = append(board.Panels, rowPanel)
		}

		// Generate a panel for displaying each observable in each row.
		for rowIndex, row := range group.Rows {
			panelWidth := 24 / len(row)
			offsetY++
			for i, o := range row {
				panelTitle := strings.ToTitle(string([]rune(o.Description)[0])) + string([]rune(o.Description)[1:])
				panel := sdk.NewGraph(panelTitle)
				panel.ID = observablePanelID(groupIndex, rowIndex, i)
				setPanelSize(panel, panelWidth, 5)
				setPanelPos(panel, i*panelWidth, offsetY)
				panel.GraphPanel.Legend.Show = true
				panel.GraphPanel.Fill = 1
				panel.GraphPanel.Lines = true
				panel.GraphPanel.Linewidth = 1
				panel.GraphPanel.NullPointMode = "connected"
				panel.GraphPanel.Pointradius = 2
				panel.GraphPanel.AliasColors = map[string]string{}
				panel.GraphPanel.Xaxis = sdk.Axis{
					Show: true,
				}

				// Add reference links
				panel.Links = []sdk.Link{{
					Title:       "Panel reference",
					URL:         stringPtr(fmt.Sprintf("%s#%s", canonicalDashboardsDocsURL, observableDocAnchor(c, o))),
					TargetBlank: boolPtr(true),
				}}
				if !o.NoAlert {
					panel.Links = append(panel.Links, sdk.Link{
						Title:       "Alerts reference",
						URL:         stringPtr(fmt.Sprintf("%s#%s", canonicalAlertSolutionsURL, observableDocAnchor(c, o))),
						TargetBlank: boolPtr(true),
					})
				}

				opt := o.PanelOptions.withDefaults()
				leftAxis := sdk.Axis{
					Decimals: 0,
					Format:   string(opt.unitType),
					LogBase:  1,
					Show:     true,
				}

				if o.Warning != nil && o.Warning.greaterThan != nil {
					// Warning threshold
					panel.GraphPanel.Thresholds = append(panel.GraphPanel.Thresholds, sdk.Threshold{
						Value:     float32(*o.Warning.greaterThan),
						Op:        "gt",
						ColorMode: "custom",
						Line:      true,
						LineColor: "rgba(255, 73, 53, 0.8)",
					})
				}
				if o.Critical != nil && o.Critical.greaterThan != nil {
					// Critical threshold
					panel.GraphPanel.Thresholds = append(panel.GraphPanel.Thresholds, sdk.Threshold{
						Value:     float32(*o.Critical.greaterThan),
						Op:        "gt",
						ColorMode: "custom",
						Line:      true,
						LineColor: "rgba(255, 17, 36, 0.8)",
					})
				}
				if o.Warning != nil && o.Warning.lessThan != nil {
					// Warning threshold
					panel.GraphPanel.Thresholds = append(panel.GraphPanel.Thresholds, sdk.Threshold{
						Value:     float32(*o.Warning.lessThan),
						Op:        "lt",
						ColorMode: "custom",
						Line:      true,
						LineColor: "rgba(255, 73, 53, 0.8)",
					})
				}
				if o.Critical != nil && o.Critical.lessThan != nil {
					// Critical threshold
					panel.GraphPanel.Thresholds = append(panel.GraphPanel.Thresholds, sdk.Threshold{
						Value:     float32(*o.Critical.lessThan),
						Op:        "lt",
						ColorMode: "custom",
						Line:      true,
						LineColor: "rgba(255, 17, 36, 0.8)",
					})
				}

				if opt.min != nil {
					leftAxis.Min = sdk.NewFloatString(*opt.min)
				}
				if opt.max != nil {
					leftAxis.Max = sdk.NewFloatString(*opt.max)
				}
				panel.GraphPanel.Yaxes = []sdk.Axis{
					leftAxis,
					{
						Format:  "short",
						LogBase: 1,
						Show:    true,
					},
				}
				panel.AddTarget(&sdk.Target{
					Expr:         o.Query,
					LegendFormat: opt.legendFormat,
					Interval:     opt.interval,
				})
				if rowPanel != nil && group.Hidden {
					rowPanel.RowPanel.Panels = append(rowPanel.RowPanel.Panels, *panel)
				} else {
					board.Panels = append(board.Panels, panel)
				}
			}
		}
	}
	return board
}

// alertDescription generates an alert description for the specified coontainer's alert.
func (c *Container) alertDescription(o Observable, alert *ObservableAlertDefinition) (string, error) {
	if alert.isEmpty() {
		return "", errors.New("cannot generate description for empty alert")
	}
	var description string

	// description based on thresholds. no special description for 'alert.strictCompare',
	// because the description is pretty ambiguous to fit different alerts.
	units := o.PanelOptions.unitType.short()
	if alert.greaterThan != nil {
		// e.g. "zoekt-indexserver: 20+ indexed search request errors every 5m by code"
		description = fmt.Sprintf("%s: %v%s+ %s", c.Name, *alert.greaterThan, units, o.Description)
	} else if alert.lessThan != nil {
		// e.g. "zoekt-indexserver: less than 20 indexed search requests every 5m by code"
		description = fmt.Sprintf("%s: less than %v%s %s", c.Name, *alert.lessThan, units, o.Description)
	} else {
		return "", fmt.Errorf("unable to generate description for observable %+v", o)
	}

	// add information about "for"
	if alert.duration > 0 {
		return fmt.Sprintf("%s for %s", description, alert.duration), nil
	}
	return description, nil
}

// renderRules generates the Prometheus rules file which defines our
// high-level alerting metrics for the container. For more information about
// how these work, see:
//
// https://docs.sourcegraph.com/admin/observability/metrics#high-level-alerting-metrics
//
func (c *Container) renderRules() (*promRulesFile, error) {
	group := promGroup{Name: c.Name}
	for groupIndex, g := range c.Groups {
		for rowIndex, r := range g.Rows {
			for observableIndex, o := range r {
				for level, a := range map[string]*ObservableAlertDefinition{
					"warning":  o.Warning,
					"critical": o.Critical,
				} {
					if a.isEmpty() {
						continue
					}

					// The alertQuery must contribute a query that returns true when it should be firing.
					var alertQuery string
					if a.greaterThan != nil {
						comparator := ">="
						if a.strictCompare {
							comparator = ">"
						}
						alertQuery = fmt.Sprintf("(%s) %s %v", o.Query, comparator, *a.greaterThan)
					} else if a.lessThan != nil {
						comparator := "<="
						if a.strictCompare {
							comparator = "<"
						}
						alertQuery = fmt.Sprintf("(%s) %s %v", o.Query, comparator, *a.lessThan)
					}

					// If the data must exist, we alert if the query returns no value as well
					if o.DataMustExist {
						alertQuery = fmt.Sprintf("(%s) OR (absent(%s) == 1)", alertQuery, o.Query)
					}

					description, err := c.alertDescription(o, a)
					if err != nil {
						return nil, fmt.Errorf("%s.%s.%s: unable to generate labels: %+v",
							c.Name, o.Name, level, err)
					}
					group.appendRow(alertQuery, map[string]string{
						"name":         o.Name,
						"level":        level,
						"service_name": c.Name,
						"description":  description,
						"owner":        string(o.Owner),

						// in the corresponding dashboard, this label should indicate
						// the panel associated with this rule
						"grafana_panel_id": strconv.Itoa(int(observablePanelID(groupIndex, rowIndex, observableIndex))),
					}, a.duration)
				}
			}
		}
	}
	if err := group.validate(); err != nil {
		return nil, err
	}
	return &promRulesFile{
		Groups: []promGroup{group},
	}, nil
}

// Group describes a group of observable information about a container.
//
// These correspond to collapsible sections in a Grafana dashboard.
type Group struct {
	// Title of the group, briefly summarizing what this group is about, or
	// "General" if the group is just about the container in general.
	Title string

	// Hidden indicates whether or not the group should be hidden by default.
	//
	// This should only be used when the dashboard is already full of information
	// and the information presented in this group is unlikely to be the cause of
	// issues and should generally only be inspected in the event that an alert
	// for that information is firing.
	Hidden bool

	// Rows of observable metrics.
	Rows []Row
}

func (g Group) validate() error {
	if g.Title != upperFirst(g.Title) || g.Title == withPeriod(g.Title) {
		return fmt.Errorf("Title must start with an uppercase letter and not end with a period; found \"%s\"", g.Title)
	}
	for i, r := range g.Rows {
		if err := r.validate(); err != nil {
			return fmt.Errorf("Row %d: %v", i, err)
		}
	}
	return nil
}

// Row of observable metrics.
//
// These correspond to a row of Grafana graphs.
type Row []Observable

func (r Row) validate() error {
	if len(r) < 1 || len(r) > 4 {
		return fmt.Errorf("row must have 1 to 4 observables only, found %v", len(r))
	}
	for i, o := range r {
		if err := o.validate(); err != nil {
			return fmt.Errorf("Observable %d %q: %v", i, o.Name, err)
		}
	}
	return nil
}

// ObservableOwner denotes a team that owns an Observable. The current teams are described in
// the handbook: https://about.sourcegraph.com/company/team/org_chart#engineering
type ObservableOwner string

const (
	ObservableOwnerSearch       ObservableOwner = "search"
	ObservableOwnerCampaigns    ObservableOwner = "campaigns"
	ObservableOwnerCodeIntel    ObservableOwner = "code-intel"
	ObservableOwnerDistribution ObservableOwner = "distribution"
	ObservableOwnerSecurity     ObservableOwner = "security"
	ObservableOwnerWeb          ObservableOwner = "web"
	ObservableOwnerCloud        ObservableOwner = "cloud"
)

// Observable describes a metric about a container that can be observed. For example, memory usage.
//
// These correspond to Grafana graphs.
type Observable struct {
	// Name is a short and human-readable lower_snake_case name describing what is being observed.
	//
	// It must be unique relative to the service name.
	//
	// Good examples:
	//
	//  github_rate_limit_remaining
	// 	search_error_rate
	//
	// Bad examples:
	//
	//  repo_updater_github_rate_limit
	// 	search_error_rate_over_5m
	//
	Name string

	// Description is a human-readable description of exactly what is being observed.
	//
	// Good examples:
	//
	// 	"remaining GitHub API rate limit quota"
	// 	"number of search errors every 5m"
	//  "90th percentile search request duration over 5m"
	//
	// Bad examples:
	//
	// 	"GitHub rate limit"
	// 	"search errors[5m]"
	// 	"P90 search latency"
	//
	Description string

	// Owner indicates the team that owns this Observable (including its alerts and maintainence).
	Owner ObservableOwner

	// Query is the actual Prometheus query that should be observed.
	Query string

	// DataMustExist indicates if the query must return data.
	//
	// For example, repo_updater_memory_usage should always have data present and an alert should
	// fire if for some reason that query is not returning any data, so this would be set to true.
	// In contrast, search_error_rate would depend on users actually performing searches and we
	// would not want an alert to fire if no data was present, so this will not need to be set.
	DataMustExist bool

	// Warning and Critical alert definitions.
	// Consider adding at least a Warning or Critical alert to each Observable to make it
	// easy to identify when the target of this metric is misbehaving. If no alerts are
	// provided, NoAlert must be set and Interpretation must be provided.
	Warning, Critical *ObservableAlertDefinition

	// NoAlerts must be set by Observables that do not have any alerts.
	// This ensures the omission of alerts is intentional. If set to true, an Interpretation
	// must be provided in place of PossibleSolutions.
	NoAlert bool

	// PossibleSolutions is Markdown describing possible solutions in the event that the
	// alert is firing. This field not required if no alerts are attached to this Observable.
	// If there is no clear potential resolution or there is no alert configured, "none"
	// must be explicitly stated.
	//
	// Use the Interpretation field for additional guidance on understanding this Observable that isn't directly related to solving it.
	// it, the Interpretation field can be provided as well.
	//
	// Contacting support should not be mentioned as part of a possible solution, as it is
	// communicated elsewhere.
	//
	// To make writing the Markdown more friendly in Go, string literals like this:
	//
	// 	Observable{
	// 		PossibleSolutions: `
	// 			- Foobar 'some code'
	// 		`
	// 	}
	//
	// Becomes:
	//
	// 	- Foobar `some code`
	//
	// In other words:
	//
	// 1. The preceding newline is removed.
	// 2. The indentation in the string literal is removed (based on the last line).
	// 3. Single quotes become backticks.
	// 4. The last line (which is all indention) is removed.
	// 5. Non-list items are converted to a list.
	//
	PossibleSolutions string

	// Interpretation is Markdown that can serve as a reference for interpreting this
	// observable. For example, Interpretation could provide guidance on what sort of
	// patterns to look for in the observable's graph and document why this observable is
	// usefule.
	//
	// If no alerts are configured for an observable, this field is required. If the
	// Description is sufficient to capture what this Observable describes, "none" must be
	// explicitly stated.
	//
	// To make writing the Markdown more friendly in Go, string literal processing as
	// PossibleSolutions is provided, though the output is not converted to a list.
	Interpretation string

	// PanelOptions describes some options for how to render the metric in the Grafana panel.
	PanelOptions ObservablePanelOptions
}

func (o Observable) validate() error {
	if strings.Contains(o.Name, " ") || strings.ToLower(o.Name) != o.Name {
		return fmt.Errorf("Name must be in lower_snake_case; found \"%s\"", o.Name)
	}
	if v := string([]rune(o.Description)[0]); v != strings.ToLower(v) {
		return fmt.Errorf("Description must be lowercase; found \"%s\"", o.Description)
	}
	if o.Owner == "" {
		return errors.New("Owner must be defined")
	}

	allAlertsEmpty := (o.Warning.isEmpty() && o.Critical.isEmpty())
	if allAlertsEmpty || o.NoAlert {
		// Ensure lack of alerts is intentional
		if allAlertsEmpty && !o.NoAlert {
			return fmt.Errorf("Warning or Critical must be set or explicitly disable alerts with NoAlert")
		} else if !allAlertsEmpty && o.NoAlert {
			return fmt.Errorf("No Warning or Critical alert is set, but NoAlert is also true")
		}
		// PossibleSolutions if there are no alerts is redundant and likely an error
		if o.PossibleSolutions != "" {
			return fmt.Errorf(`PossibleSolutions is not required if no alerts are configured - did you mean to provide an Interpretation instead?`)
		}
		// Interpretation must be provided and valid
		if o.Interpretation == "" {
			return fmt.Errorf("Interpretation must be provided if no alerts are set")
		} else if o.Interpretation != "none" {
			if _, err := toMarkdown(o.Interpretation, false); err != nil {
				return fmt.Errorf("Interpretation cannot be converted to Markdown: %w", err)
			}
		}
	} else {
		// Ensure alerts are valid
		for alertLevel, alert := range map[string]*ObservableAlertDefinition{
			"Warning":  o.Warning,
			"Critical": o.Critical,
		} {
			if err := alert.validate(); err != nil {
				return fmt.Errorf("%s Alert: %w", alertLevel, err)
			}
		}
		// PossibleSolutions must be provided and valid
		if o.PossibleSolutions == "" {
			return fmt.Errorf(`PossibleSolutions must list solutions or an explicit "none"`)
		} else if o.PossibleSolutions != "none" {
			if solutions, err := toMarkdown(o.PossibleSolutions, true); err != nil {
				return fmt.Errorf("PossibleSolutions cannot be converted to Markdown: %w", err)
			} else if l := strings.ToLower(solutions); strings.Contains(l, "contact support") || strings.Contains(l, "contact us") {
				return fmt.Errorf("PossibleSolutions should not include mentions of contacting support")
			}
		}
	}

	return nil
}

// Alert provides a builder for defining alerting on an Observable.
func Alert() *ObservableAlertDefinition {
	return &ObservableAlertDefinition{}
}

// ObservableAlertDefinition defines when an alert would be considered firing.
type ObservableAlertDefinition struct {
	greaterThan   *float64
	lessThan      *float64
	strictCompare bool

	duration time.Duration
}

// GreaterOrEqual indicates the alert should fire when greater or equal the given value.
func (a *ObservableAlertDefinition) GreaterOrEqual(f float64) *ObservableAlertDefinition {
	a.greaterThan = &f
	a.strictCompare = false
	return a
}

// LessOrEqual indicates the alert should fire when less than or equal to the given value.
func (a *ObservableAlertDefinition) LessOrEqual(f float64) *ObservableAlertDefinition {
	a.lessThan = &f
	a.strictCompare = false
	return a
}

// Greater indicates the alert should fire when strictly greater to this value.
func (a *ObservableAlertDefinition) Greater(f float64) *ObservableAlertDefinition {
	a.greaterThan = &f
	a.strictCompare = true
	return a
}

// Less indicates the alert should fire when strictly less than this value.
func (a *ObservableAlertDefinition) Less(f float64) *ObservableAlertDefinition {
	a.lessThan = &f
	a.strictCompare = true
	return a
}

// For indicates how long the given thresholds must be exceeded for this alert to be
// considered firing. Defaults to 0s (immediately alerts when threshold is exceeded).
func (a *ObservableAlertDefinition) For(d time.Duration) *ObservableAlertDefinition {
	a.duration = d
	return a
}

func (a *ObservableAlertDefinition) isEmpty() bool {
	return a == nil || (*a == ObservableAlertDefinition{}) || (a.greaterThan == nil && a.lessThan == nil)
}

func (a *ObservableAlertDefinition) validate() error {
	if a.isEmpty() {
		return nil
	}
	if a.greaterThan != nil && a.lessThan != nil {
		return errors.New("only one bound (greater or less) can be set")
	}
	return nil
}

// UnitType for controlling the unit type display on graphs.
type UnitType string

// short returns the short string description of the unit, for qualifying a
// number of this unit type as human-readable.
func (u UnitType) short() string {
	switch u {
	case Number, "":
		return ""
	case Milliseconds:
		return "ms"
	case Seconds:
		return "s"
	case Percentage:
		return "%"
	case Bytes:
		return "B"
	case BitsPerSecond:
		return "bps"
	default:
		panic("never here")
	}
}

// From https://sourcegraph.com/github.com/grafana/grafana@b63b82976b3708b082326c0b7d42f38d4bc261fa/-/blob/packages/grafana-data/src/valueFormats/categories.ts#L23
const (
	// Number is the default unit type.
	Number UnitType = "short"

	// Milliseconds for representing time.
	Milliseconds UnitType = "dtdurationms"

	// Seconds for representing time.
	Seconds UnitType = "dtdurations"

	// Percentage in the range of 0-100.
	Percentage UnitType = "percent"

	// Bytes in IEC (1024) format, e.g. for representing storage sizes.
	Bytes UnitType = "bytes"

	// BitsPerSecond, e.g. for representing network and disk IO.
	BitsPerSecond UnitType = "bps"
)

// ObservablePanelOptions declares options for visualizing an Observable.
type ObservablePanelOptions struct {
	min, max     *float64
	minAuto      bool
	legendFormat string
	unitType     UnitType
	interval     string
}

// PanelOptions provides a builder for customizing an Observable visualization.
func PanelOptions() ObservablePanelOptions { return ObservablePanelOptions{} }

// Min sets the minimum value of the Y axis on the panel. The default is zero.
func (p ObservablePanelOptions) Min(min float64) ObservablePanelOptions {
	p.min = &min
	return p
}

// Min sets the minimum value of the Y axis on the panel to auto, instead of
// the default zero.
//
// This is generally only useful if trying to show negative numbers.
func (p ObservablePanelOptions) MinAuto() ObservablePanelOptions {
	p.minAuto = true
	return p
}

// Max sets the maximum value of the Y axis on the panel. The default is auto.
func (p ObservablePanelOptions) Max(max float64) ObservablePanelOptions {
	p.max = &max
	return p
}

// LegendFormat sets the panel's legend format, which may use Go template strings to select
// labels from the Prometheus query.
func (p ObservablePanelOptions) LegendFormat(format string) ObservablePanelOptions {
	p.legendFormat = format
	return p
}

// Unit sets the panel's Y axis unit type.
func (p ObservablePanelOptions) Unit(t UnitType) ObservablePanelOptions {
	p.unitType = t
	return p
}

// Interval declares the panel's interval in milliseconds.
func (p ObservablePanelOptions) Interval(ms int) ObservablePanelOptions {
	p.interval = fmt.Sprintf("%dms", ms)
	return p
}

func (p ObservablePanelOptions) withDefaults() ObservablePanelOptions {
	if p.min == nil && !p.minAuto {
		defaultMin := 0.0
		p.min = &defaultMin
	}
	if p.legendFormat == "" {
		// Important: We use "value" as the default legend format and not, say, "{{instance}}" or
		// an empty string (Grafana defaults to all labels in that case) because:
		//
		// 1. Using "{{instance}}" is often wrong, see: https://about.sourcegraph.com/handbook/engineering/observability/monitoring_pillars#faq-why-can-t-i-create-a-graph-panel-with-more-than-5-cardinality-labels
		// 2. More often than not, you actually do want to aggregate your whole query with `sum()`, `max()` or similar.
		// 3. If "{{instance}}" or similar was the default, it would be easy for people to say "I guess that's intentional"
		//    instead of seeing multiple "value" labels on their dashboard (which immediately makes them think
		//    "how can I fix that?".)
		//
		p.legendFormat = "value"
	}
	if p.unitType == "" {
		p.unitType = Number
	}
	return p
}
