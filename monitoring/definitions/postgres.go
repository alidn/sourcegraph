package definitions

import (
	"time"

	"github.com/sourcegraph/sourcegraph/monitoring/definitions/shared"
	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

func Postgres() *monitoring.Container {
	// In Kubernetes, codeintel-db container is called codeintel-db and the
	// default Postgres container is called pgsql
	// Because of this, we track all database cAdvisor metrics in a single panel using this
	// container name regex to ensure we have observability on all platforms.
	const databaseContainerNames = "(pgsql|codeintel-db)"

	return &monitoring.Container{
		Name:        "postgres",
		Title:       "Postgres",
		Description: "Postgres metrics, exported from postgres_exporter (only available on Kubernetes).",
		Groups: []monitoring.Group{
			{
				Title: "General",
				Rows: []monitoring.Row{{
					monitoring.Observable{
						Name:              "connections",
						Description:       "active connections",
						Owner:             monitoring.ObservableOwnerCloud,
						Query:             `sum by (datname) (pg_stat_activity_count{datname!~"template.*|postgres|cloudsqladmin"})`,
						PanelOptions:      monitoring.PanelOptions().LegendFormat("{{datname}}"),
						Warning:           monitoring.Alert().LessOrEqual(5).For(5 * time.Minute),
						PossibleSolutions: "none",
					},
					monitoring.Observable{
						Name:              "transaction_durations",
						Description:       "maximum transaction durations",
						Owner:             monitoring.ObservableOwnerCloud,
						Query:             `sum by (datname) (pg_stat_activity_max_tx_duration{datname!~"template.*|postgres|cloudsqladmin"})`,
						PanelOptions:      monitoring.PanelOptions().LegendFormat("{{datname}}").Unit(monitoring.Milliseconds),
						Warning:           monitoring.Alert().GreaterOrEqual(300).For(5 * time.Minute),
						Critical:          monitoring.Alert().GreaterOrEqual(500).For(10 * time.Minute),
						PossibleSolutions: "none",
					},
				},
				},
			},
			{
				Title: "Database and collector status", Hidden: true,
				Rows: []monitoring.Row{
					{
						monitoring.Observable{
							Name:              "postgres_up",
							Description:       "database availability",
							Owner:             monitoring.ObservableOwnerCloud,
							Query:             "pg_up",
							PanelOptions:      monitoring.PanelOptions().LegendFormat("{{app}}"),
							Critical:          monitoring.Alert().LessOrEqual(0).For(5 * time.Minute),
							PossibleSolutions: "none",
							Interpretation:    "A non-zero value indicates the database is online.",
						},
						monitoring.Observable{
							Name:         "pg_exporter_err",
							Description:  "errors scraping postgres exporter",
							Owner:        monitoring.ObservableOwnerCloud,
							Query:        "pg_exporter_last_scrape_error",
							PanelOptions: monitoring.PanelOptions().LegendFormat("{{app}}"),
							Warning:      monitoring.Alert().GreaterOrEqual(1).For(5 * time.Minute),
							PossibleSolutions: `
								- Ensure the Postgres exporter can access the Postgres database. Also, check the Postgres exporter logs for errors.`,
							Interpretation: "This value indicates issues retrieving metrics from postgres_exporter.",
						},
						monitoring.Observable{
							Name:           "migration_in_progress",
							Description:    "active schema migration",
							Owner:          monitoring.ObservableOwnerCloud,
							Query:          "pg_sg_migration_status",
							PanelOptions:   monitoring.PanelOptions().LegendFormat("{{app}}"),
							Critical:       monitoring.Alert().GreaterOrEqual(1).For(5 * time.Minute),
							Interpretation: "A 0 value indicates that no migration is in progress.",
							PossibleSolutions: `
								The database migration has been in progress for 5 or more minutes - please contact Sourcegraph if this persists.
							`,
						},
						// TODO(@daxmc99): Blocked by https://github.com/sourcegraph/sourcegraph/issues/13300
						// need to enable `pg_stat_statements` in Postgres conf
						// monitoring.Observable{
						//	Name:            "cache_hit_ratio",
						//	Description:     "ratio of cache hits over 5m",
						//	Owner:           monitoring.ObservableOwnerCloud,
						//	Query:           `avg(rate(pg_stat_database_blks_hit{datname!~"template.*|postgres|cloudsqladmin"}[5m]) / (rate(pg_stat_database_blks_hit{datname!~"template.*|postgres|cloudsqladmin"}[5m]) + rate(pg_stat_database_blks_read{datname!~"template.*|postgres|cloudsqladmin"}[5m]))) by (datname) * 100`,
						//	DataMayNotExist: true,
						//	Warning:         monitoring.Alert().LessOrEqual(0.98).For(5 * time.Minute),
						//	PossibleSolutions: "Cache hit ratio should be at least 99%, please [open an issue](https://github.com/sourcegraph/sourcegraph/issues/new/choose) " +
						//		"to add additional indexes",
						//	PanelOptions: monitoring.PanelOptions().Unit(monitoring.Percentage)},
					},
				},
			},
			{
				Title:  "Provisioning indicators (not available on server)",
				Hidden: true,
				// See docstring for databaseContainerNames
				Rows: []monitoring.Row{
					{
						shared.ProvisioningCPUUsageLongTerm(databaseContainerNames, monitoring.ObservableOwnerCloud).Observable(),
						shared.ProvisioningMemoryUsageLongTerm(databaseContainerNames, monitoring.ObservableOwnerCloud).Observable(),
					},
					{
						shared.ProvisioningCPUUsageShortTerm(databaseContainerNames, monitoring.ObservableOwnerCloud).Observable(),
						shared.ProvisioningMemoryUsageShortTerm(databaseContainerNames, monitoring.ObservableOwnerCloud).Observable(),
					},
				},
			},
			{
				Title:  "Kubernetes monitoring (ignore if using Docker Compose or server)",
				Hidden: true,
				Rows: []monitoring.Row{
					{
						shared.KubernetesPodsAvailable(databaseContainerNames, monitoring.ObservableOwnerCloud).Observable(),
					},
				},
			},
		},
	}
}
