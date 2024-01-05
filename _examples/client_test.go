package authz_test

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"testing"

	// This is the generated client
	authz "github.com/ben-mays/spicegen/examples"
	"github.com/ben-mays/spicegen/examples/permissions/document"
	"github.com/ben-mays/spicegen/examples/permissions/organization"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	spicedb "github.com/authzed/spicedb/e2e/spice"
	"github.com/stretchr/testify/assert"
)

//go:embed schema.text
var schematxt string

// requires ./spicedb to be available, use `ln -s /opt/homebrew/bin/spicedb spicedb`
func NewSpiceDB() spiceDBv1Client {
	ctx := context.Background()
	node := spicedb.NewNodeWithOptions(spicedb.WithTestDefaults(), spicedb.WithDatastore("memory"))
	err := node.Start(ctx, "spicedb")
	if err != nil {
		panic(err)
	}
	fmt.Printf("spawned spicedb at pid %d on port %d\n", node.Pid, node.GrpcPort)
	err = node.Connect(ctx, os.Stdout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("connected to spicedb at pid %d\n", node.Pid)
	v1Client := node.Client()
	return spiceDBv1Client{v1Client.V1().Permissions(), v1Client.V1().Schema()}
}

// shim e2e client into SpiceDBClient interface
type spiceDBv1Client struct {
	v1.PermissionsServiceClient
	v1.SchemaServiceClient
}

func (s *spiceDBv1Client) Permissions() v1.PermissionsServiceClient {
	return s
}

func (s *spiceDBv1Client) Schema() v1.SchemaServiceClient {
	return s
}

func TestSpiceDB(t *testing.T) {
	spicedb := NewSpiceDB()
	_, err := spicedb.WriteSchema(context.Background(), &v1.WriteSchemaRequest{Schema: schematxt})
	if err != nil {
		panic(err)
	}
	svc := authz.NewClient(spicedb)
	ctx := context.Background()

	// Add user:ben to organization:nike
	resp, err := svc.AddOrganizationRelationship(
		ctx, authz.NewOrganizationResource("nike"),
		organization.AdministratorRelation,
		authz.NewUserResource("ben"), nil)
	assert.Nil(t, err)
	assert.True(t, resp)

	// Add doc:readme to organization:nike
	resp, err = svc.AddDocumentRelationship(ctx,
		authz.NewDocumentResource("readme"),
		document.DocorgRelation,
		authz.NewOrganizationResource("nike"), nil)

	// Can org admin Ben view doc:readme? Expectation: yes
	resp, err = svc.CheckDocumentPermission(ctx,
		authz.NewDocumentResource("readme"),
		document.ViewPermission,
		authz.NewUserResource("ben"), nil)
	assert.Nil(t, err)
	assert.True(t, resp)

	// Can Alice view doc:readme? Expectation: no
	resp, err = svc.CheckDocumentPermission(ctx,
		authz.NewDocumentResource("readme"),
		document.ViewPermission,
		authz.NewUserResource("alice"), nil)
	assert.Nil(t, err)
	assert.False(t, resp)

	// Can Alice view doc:readme? Expectation: no
	resp, err = svc.CheckDocumentPermission(ctx,
		authz.NewDocumentResource("readme"),
		document.ViewPermission,
		authz.NewUserResource("alice"), nil)
	assert.Nil(t, err)
	assert.True(t, resp)
}
