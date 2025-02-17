package resolvers

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go/log"

	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/autoindex/config"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/autoindex/enqueuer"
	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

// Resolver is the main interface to code intel-related operations exposed to the GraphQL API.
// This resolver consolidates the logic for code intel operations and is not itself concerned
// with GraphQL/API specifics (auth, validation, marshaling, etc.). This resolver is wrapped
// by a symmetrics resolver in this package's graphql subpackage, which is exposed directly
// by the API.
type Resolver interface {
	GetUploadByID(ctx context.Context, id int) (store.Upload, bool, error)
	GetIndexByID(ctx context.Context, id int) (store.Index, bool, error)
	UploadConnectionResolver(opts store.GetUploadsOptions) *UploadsResolver
	IndexConnectionResolver(opts store.GetIndexesOptions) *IndexesResolver
	DeleteUploadByID(ctx context.Context, uploadID int) error
	DeleteIndexByID(ctx context.Context, id int) error
	IndexConfiguration(ctx context.Context, repositoryID int) (store.IndexConfiguration, error)
	UpdateIndexConfigurationByRepositoryID(ctx context.Context, repositoryID int, configuration string) error
	QueueAutoIndexJobForRepo(ctx context.Context, repositoryID int) error
	QueryResolver(ctx context.Context, args *gql.GitBlobLSIFDataArgs) (QueryResolver, error)
}

type resolver struct {
	dbStore      DBStore
	lsifStore    LSIFStore
	codeIntelAPI CodeIntelAPI
	hunkCache    HunkCache
	operations   *operations
	enqueuer     *enqueuer.IndexEnqueuer
}

// NewResolver creates a new resolver with the given services.
func NewResolver(
	dbStore DBStore,
	lsifStore LSIFStore,
	codeIntelAPI CodeIntelAPI,
	hunkCache HunkCache,
	observationContext *observation.Context,
	gitClient enqueuer.GitserverClient,
	enqueuerDBStore enqueuer.DBStore,
) Resolver {
	return &resolver{
		dbStore:      dbStore,
		lsifStore:    lsifStore,
		codeIntelAPI: codeIntelAPI,
		hunkCache:    hunkCache,
		operations:   makeOperations(observationContext),
		enqueuer:     enqueuer.NewIndexEnqueuer(enqueuerDBStore, gitClient, observationContext),
	}
}

func (r *resolver) GetUploadByID(ctx context.Context, id int) (store.Upload, bool, error) {
	return r.dbStore.GetUploadByID(ctx, id)
}

func (r *resolver) GetIndexByID(ctx context.Context, id int) (store.Index, bool, error) {
	return r.dbStore.GetIndexByID(ctx, id)
}

func (r *resolver) UploadConnectionResolver(opts store.GetUploadsOptions) *UploadsResolver {
	return NewUploadsResolver(r.dbStore, opts)
}

func (r *resolver) IndexConnectionResolver(opts store.GetIndexesOptions) *IndexesResolver {
	return NewIndexesResolver(r.dbStore, opts)
}

func (r *resolver) DeleteUploadByID(ctx context.Context, uploadID int) error {
	_, err := r.dbStore.DeleteUploadByID(ctx, uploadID)
	return err
}

func (r *resolver) DeleteIndexByID(ctx context.Context, id int) error {
	_, err := r.dbStore.DeleteIndexByID(ctx, id)
	return err
}

func (r *resolver) IndexConfiguration(ctx context.Context, repositoryID int) (store.IndexConfiguration, error) {
	configuration, ok, err := r.dbStore.GetIndexConfigurationByRepositoryID(ctx, repositoryID)
	if err != nil || !ok {
		return store.IndexConfiguration{}, err
	}

	return configuration, nil
}

func (r *resolver) UpdateIndexConfigurationByRepositoryID(ctx context.Context, repositoryID int, configuration string) error {
	if _, err := config.UnmarshalJSON([]byte(configuration)); err != nil {
		return err
	}

	return r.dbStore.UpdateIndexConfigurationByRepositoryID(ctx, repositoryID, []byte(configuration))
}

func (r *resolver) QueueAutoIndexJobForRepo(ctx context.Context, repositoryID int) error {
	return r.enqueuer.ForceQueueIndex(ctx, repositoryID)
}

const slowQueryResolverRequestThreshold = time.Second

// QueryResolver determines the set of dumps that can answer code intel queries for the
// given repository, commit, and path, then constructs a new query resolver instance which
// can be used to answer subsequent queries.
func (r *resolver) QueryResolver(ctx context.Context, args *gql.GitBlobLSIFDataArgs) (_ QueryResolver, err error) {
	ctx, endObservation := observeResolver(ctx, &err, "QueryResolver", r.operations.queryResolver, slowQueryResolverRequestThreshold, observation.Args{
		LogFields: []log.Field{
			log.Int("repositoryID", int(args.Repo.ID)),
			log.String("commit", string(args.Commit)),
			log.String("path", args.Path),
			log.Bool("exactPath", args.ExactPath),
			log.String("toolName", args.ToolName),
		},
	})
	defer endObservation()

	dumps, err := r.codeIntelAPI.FindClosestDumps(
		ctx,
		int(args.Repo.ID),
		string(args.Commit),
		args.Path,
		args.ExactPath,
		args.ToolName,
	)
	if err != nil || len(dumps) == 0 {
		return nil, err
	}

	return NewQueryResolver(
		r.dbStore,
		r.lsifStore,
		r.codeIntelAPI,
		NewPositionAdjuster(args.Repo, string(args.Commit), r.hunkCache),
		int(args.Repo.ID),
		string(args.Commit),
		args.Path,
		dumps,
		r.operations,
	), nil
}
