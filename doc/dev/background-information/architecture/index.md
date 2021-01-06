# Sourcegraph architecture overview

This document provides a high level overview of Sourcegraph's architecture so you can understand how our systems fit together.

## Code syncing

At its core, Sourcegraph maintains a persistent cache of all the code that is connected to it. It is persistent, because this data is critical for Sourcegraph to function, but it is ultimately a cache because the code host is the source of truth and our cache is eventually consistent.

- [gitserver](../../../../cmd/gitserver/README.md) is the sharded service that stores the code and makes it accessible to other Sourcegraph services.
- [repo-updater](../../../../cmd/repo-updater/README.md) is the singleton service that is responsible for ensuring all the code in gitserver is as up-to-date as possible while respecting code host rate limits. It is also responsible for syncing code repository metadata from the code host that is stored in the `repo` table of our Postgres database.

If you want to learn more about how code is synchronized, read [Life of a repository](life-of-a-repository.md).

## Search

Devs can search across all the code that is connected to their Sourcegraph instance.

By default, Sourcegraph uses [zoekt](https://github.com/sourcegraph/zoekt) to create a trigram index of the default branch of every repository so that searches are fast. This trigram index is the reason why Sourcegraph search is more powerful and faster than what is usually provided by code hosts. 

- [zoekt-indexserver](https://sourcegraph.com/github.com/sourcegraph/zoekt/-/tree/cmd/zoekt-sourcegraph-indexserver) 
- [zoekt-webserver](https://sourcegraph.com/github.com/sourcegraph/zoekt/-/tree/cmd/zoekt-webserver)

Sourcegraph also has a fast search path for code that isn't indexed yet, or for code that will never be indexed (for example: code that is not on a default branch). Indexing every branch of every repository isn't a pragmatic use of resources for most customers, so this decision balances optimizing the common case (searching all default branches) with space savings (not indexing everything).

- [searcher](../../../../cmd/searcher/README.md) implements the non-indexed search.

Syntax highlighting for any code view, including search results, is provided by [Syntect server](https://sourcegraph.com/github.com/sourcegraph/syntect_server).

If you want to learn more about search:

- [Code search product documentation](../../../code_search/index.md)
- [Life of a search query](life-of-a-search-query.md)
- [Search pagination](search-pagination.md)

## Code intelligence

Code intelligence surfaces data (for example: doc comments for a symbol) and actions (for example: go to definition, find references) based on our semantic understanding of code (unlike search, which is completely text based).

By default, Sourcegraph provides imprecise [search-based code intelligence](../../../code_intelligence/explanations/search_based_code_intelligence.md). This reuses all the architecture that makes search fast, but it can result in false positives (for example: finding two definitions for a symbol, or references that aren't actually references), or false negatives (for example: not able to find the definition or all references). This is the default because it works with no extra configuration and is pretty good for many use cases and languages. We support a lot of languages this way because it only requires writing a few regular expressions.

With some setup, customer can enable [precise code intelligence](../../../code_intelligence/explanations/precise_code_intelligence.md). Repositories add a step to their build pipeline that computes the index for that revision of code and uploads it to Sourcegraph. We have to write language specific indexers, so adding precise code intel support for new languages is a non-trivial task. 

If you want to learn more about code intelligence:

- [Code intelligence product documentation](../../../code_intelligence/index.md)
- [Code intelligence developer documentation](../codeintel/index.md)
- [Available indexers](../../../code_intelligence/references/indexers.md)

## Campaigns

TODO

## Code insights

TODO

## Code monitoring

TODO

## Browser extensions

TODO

## Sourcegraph extension API

TODO

If you want to learn more about our extension API:

- [Sourcegraph extension architecture](sourcegraph-extensions.md)

## src-cli

TODO

## Editor extensions

TODO

## Deployment

TODO

## Observability

Observability encapsulates the monitoring and debugging of Sourcegraph deployments.
Sourcegraph is designed, and ships with, a number of observability tools and capabilities out-of-the box to enable visibility into the health and state of a Sourcegraph deployment.

Monitoring includes [metrics and dashboards](../../../admin/observability/metrics.md), [alerting](../../../admin/observability/alerting.md), and [health checking](../../../admin/observability/health_checks.md) capabilities.
Learn more about monitoring in the [monitoring architecture overview](https://about.sourcegraph.com/handbook/engineering/observability/monitoring_architecture).

- [grafana](../observability/grafana.md) is the frontend for service metrics, and ships with customized dashboards for Sourcegraph services.
- [prometheus](../observability/prometheus.md) handles scraping of service metrics, and ships with recording rules, alert rules, and alerting capabilities.
- [Health checks are provided by each Sourcegraph service](../../../admin/observability/health_checks.md).

Debugging includes [tracing](../../../admin/observability/tracing.md) and [logging](../../../admin/observability/logs.md).

- [jaeger](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/tree/docker-images/jaeger-all-in-one) is the distributed tracing service used by Sourcegraph.
- [jaeger-agent](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/tree/docker-images/jaeger-agent) is a sidecar used in Kubernetes deployments for collecting traces from services.
- [Logs are provided by each Sourcegraph service](../../../admin/observability/logs.md).

If you want to learn more about observability:

- [Observability for site administrators](../../../admin/observability/index.md)
- [Observability developer documentation](../observability/index.md)
- [Observability at Sourcegraph](https://about.sourcegraph.com/handbook/engineering/observability)

<!-- content below here has not been reorganized or refreshed yet -->

## Diagram

You can click on each component to jump to its respective code repository or subtree.

<object data="/dev/background-information/architecture/architecture.svg" type="image/svg+xml" style="width:100%; height: 100%">
</object>

Note that almost every service has a link back to the frontend, from which is gathers configuration updates.
These edges are omitted for clarity.

## Clients

We maintain multiple Sourcegraph clients:

- [Web application](https://github.com/sourcegraph/sourcegraph/tree/main/client/web)
- [Browser extensions](https://github.com/sourcegraph/sourcegraph/tree/main/client/browser)
- [src-cli](https://github.com/sourcegraph/src-cli)
- [Editor integrations](https://docs.sourcegraph.com/integration/editor)
  - [Visual Studio Code](https://github.com/sourcegraph/sourcegraph-vscode)
  - [Atom](https://github.com/sourcegraph/sourcegraph-atom)
  - [JetBrains IDEs](https://github.com/sourcegraph/sourcegraph-jetbrains)
  - [Sublime Text 3](https://github.com/sourcegraph/sourcegraph-sublime)

These clients generally communicate with a Sourcegraph instance (either https://sourcegraph.com or a private customer instance) through our [GraphQL API](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/cmd/frontend/graphqlbackend/schema.graphql). There are also a small number of REST endpoints for specific use-cases.

## Services

Our backend is composed of multiple services:

- Most are Go services found in the [cmd](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/tree/cmd) and [enterprise/cmd](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/tree/enterprise/cmd) folders.

## Infrastructure

- [sourcegraph/infrastructure](https://github.com/sourcegraph/infrastructure) contains Terraform configurations for Cloudflare DNS and Site 24x7 monitoring, as well as build steps for various Docker images. Only private Docker images should be added here, public ones belong in the main repository.
- [sourcegraph/deploy-sourcegraph](https://github.com/sourcegraph/deploy-sourcegraph) contains YAML that can be used by customers to deploy Sourcegraph to a Kubernetes cluster.
- [sourcegraph/deploy-sourcegraph-docker](https://github.com/sourcegraph/deploy-sourcegraph-docker) contains a pure-Docker cluster deployment reference that some one-off customers use to deploy Sourcegraph to a non-Kubernetes cluster.
  - [sourcegraph/deploy-sourcegraph-dot-com](https://github.com/sourcegraph/deploy-sourcegraph-dot-com) is a fork of the above that is used to deploy to the Kubernetes cluster that serves https://sourcegraph.com.

## References

Here are some references to help you understand how multiple systems fit together:

- [Life of a ping](life-of-a-ping.md)
