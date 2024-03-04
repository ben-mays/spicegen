package authz_test

import (
	"context"
	_ "embed"
	"fmt"
	"testing"

	authz "github.com/ben-mays/spicegen/examples"
	"github.com/ben-mays/spicegen/examples/permissions/document"
	"github.com/ben-mays/spicegen/examples/permissions/organization"
	"github.com/ben-mays/spicegen/examples/permissions/team"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/spicedb/pkg/cmd/datastore"
	"github.com/authzed/spicedb/pkg/cmd/server"
	"github.com/authzed/spicedb/pkg/cmd/util"
)

//go:embed schema.text
var schematxt string

func NewEmbeddedClient(ctx context.Context, schema string) (*authzed.Client, error) {
	logger := zerolog.Ctx(ctx)
	srv, err := newServer(ctx, logger)
	if err != nil {
		return nil, fmt.Errorf("unable to init server: %w", err)
	}

	conn, err := srv.GRPCDialContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get gRPC connection: %w", err)
	}

	schemaSrv := pb.NewSchemaServiceClient(conn)

	go func() {
		if err := srv.Run(ctx); err != nil {
			logger.Error().Err(err).Msg("error running server")
			return
		}
	}()

	_, err = schemaSrv.WriteSchema(ctx, &pb.WriteSchemaRequest{
		Schema: schema,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get gRPC connection: %w", err)
	}

	return &authzed.Client{
		SchemaServiceClient:      schemaSrv,
		PermissionsServiceClient: pb.NewPermissionsServiceClient(conn),
		WatchServiceClient:       pb.NewWatchServiceClient(conn),
	}, nil

}

func newServer(ctx context.Context, logger *zerolog.Logger) (server.RunnableServer, error) {
	ds, err := datastore.NewDatastore(ctx,
		datastore.DefaultDatastoreConfig().ToOption(),
		datastore.WithRequestHedgingEnabled(false),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to start memdb datastore: %w", err)
	}

	configOpts := []server.ConfigOption{
		server.WithGRPCServer(util.GRPCServerConfig{
			Network: util.BufferedNetwork,
			Enabled: true,
		}),
		server.WithGRPCAuthFunc(func(ctx context.Context) (context.Context, error) {
			return ctx, nil
		}),
		server.WithHTTPGateway(util.HTTPServerConfig{HTTPEnabled: false}),
		server.WithMetricsAPI(util.HTTPServerConfig{HTTPEnabled: false}),
		// disable caching since it's all in memory
		server.WithDispatchCacheConfig(server.CacheConfig{Enabled: false, Metrics: false}),
		server.WithNamespaceCacheConfig(server.CacheConfig{Enabled: false, Metrics: false}),
		server.WithClusterDispatchCacheConfig(server.CacheConfig{Enabled: false, Metrics: false}),
		server.WithDatastore(ds),
	}

	return server.NewConfigWithOptionsAndDefaults(configOpts...).Complete(ctx)
}

func TestSpiceDB(t *testing.T) {
	ctx := context.Background()
	spicedb, err := NewEmbeddedClient(ctx, schematxt)
	if err != nil {
		t.Fatal(err)
	}
	svc := authz.NewClient(spicedb)

	// Add user:ben to organization:nike
	err = svc.AddOrganizationRelationship(
		ctx, authz.NewOrganizationResource("nike"),
		organization.AdministratorRelation,
		authz.NewUserResource("ben"), nil)
	assert.Nil(t, err)

	// Add doc:readme to organization:nike
	err = svc.AddDocumentRelationship(ctx,
		authz.NewDocumentResource("readme"),
		document.DocorgRelation,
		authz.NewOrganizationResource("nike"), nil)

	// Can org admin Ben view doc:readme? Expectation: yes
	allowed, err := svc.CheckDocumentPermission(ctx,
		authz.NewUserResource("ben"),
		document.ViewPermission,
		authz.NewDocumentResource("readme"), nil)
	assert.Nil(t, err)
	assert.True(t, allowed)

	// Can Alice view doc:readme? Expectation: no
	allowed, err = svc.CheckDocumentPermission(ctx,
		authz.NewUserResource("alice"),
		document.ViewPermission,
		authz.NewDocumentResource("readme"), nil)
	assert.Nil(t, err)
	assert.False(t, allowed)

	// What docs can Ben read?
	resources, _, err := svc.LookupDocumentResources(ctx,
		authz.NewUserResource("ben"),
		document.ViewPermission,
		nil)
	assert.Nil(t, err)
	assert.Equal(t, []string{"readme"}, resources)

	// What docs can Alice read?
	resources, _, err = svc.LookupDocumentResources(ctx,
		authz.NewUserResource("alice"),
		document.ViewPermission,
		nil)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(resources))

	err = svc.AddTeamRelationship(ctx, authz.NewTeamResource("nike"), team.MemberRelation, authz.NewTeamResource("ben"), nil)
	assert.NotNil(t, err)
	assert.Equal(t, "relation `member` requires an optional subject relation `member` for subject type `team`", err.Error())

	err = svc.AddTeamRelationship(ctx, authz.NewTeamResource("nike"), team.MemberRelation, authz.NewTeamResource("ben"), &authz.AddRelationshipOptions{OptionalSubjectRelation: "member"})
	assert.Nil(t, err)
}
