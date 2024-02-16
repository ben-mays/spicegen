package internal

import (
	"fmt"
	"testing"

	"github.com/authzed/spicedb/pkg/schemadsl/compiler"
	"github.com/stretchr/testify/assert"
)

func TestBuildSchema(t *testing.T) {
	tests := []struct {
		name      string
		schematxt string
		err       error
		validate  func(schema Schema) error
	}{
		{
			name: "simple schema",
			schematxt: `definition user {}
						definition document {
							relation reader: user
							permission view = reader
						}`,
			validate: func(schema Schema) error {
				// assert basic structure
				if len(schema.Resources) != 2 {
					return assert.AnError
				}
				if len(schema.Resources["user"].Permissions) != 0 {
					return assert.AnError
				}
				if len(schema.Resources["user"].Relations) != 0 {
					return assert.AnError
				}
				if len(schema.Resources["document"].Permissions) != 1 {
					return assert.AnError
				}
				if len(schema.Resources["document"].Relations) != 1 {
					return assert.AnError
				}
				readerRel := schema.Resources["document"].Relations["reader"]
				if readerRel.Name != "reader" ||
					readerRel.OutputName != "reader" ||
					readerRel.Kind != "relation" ||
					len(readerRel.AllowedSubjectTypes) != 1 ||
					readerRel.AllowedSubjectTypes["user"] != "..." ||
					len(readerRel.OverrideAllowedSubjectTypes) != 0 ||
					len(readerRel.RelationRefs) != 1 ||
					readerRel.RelationRefs[0].ResourceType != "user" ||
					readerRel.RelationRefs[0].Relation != "..." ||
					readerRel.RelationRefs[0].Caveat != "" {
					return fmt.Errorf("unexpected reader relation: %+v", readerRel)
				}
				viewPerm := schema.Resources["document"].Permissions["view"]
				if viewPerm.Name != "view" ||
					viewPerm.OutputName != "view" ||
					viewPerm.Kind != "permission" ||
					len(viewPerm.AllowedSubjectTypes) != 1 ||
					viewPerm.AllowedSubjectTypes["user"] != "..." ||
					len(viewPerm.OverrideAllowedSubjectTypes) != 0 ||
					len(viewPerm.RelationRefs) != 1 ||
					viewPerm.RelationRefs[0].ResourceType != "document" ||
					viewPerm.RelationRefs[0].Relation != "reader" ||
					viewPerm.RelationRefs[0].Caveat != "" {
					return fmt.Errorf("unexpected view permission: %+v", viewPerm)
				}
				return nil
			},
		},
		{
			name: "simple schema with union",
			schematxt: `definition user {}
						definition document {
							relation reader: user | document
							permission view = reader
						}`,
			validate: func(schema Schema) error {
				readerRel := schema.Resources["document"].Relations["reader"]
				if readerRel.Name != "reader" ||
					readerRel.OutputName != "reader" ||
					readerRel.Kind != "relation" ||
					len(readerRel.AllowedSubjectTypes) != 2 ||
					readerRel.AllowedSubjectTypes["user"] != "..." ||
					readerRel.AllowedSubjectTypes["document"] != "..." ||
					len(readerRel.OverrideAllowedSubjectTypes) != 0 ||
					len(readerRel.RelationRefs) != 2 ||
					readerRel.RelationRefs[0].ResourceType != "user" ||
					readerRel.RelationRefs[0].Relation != "..." ||
					readerRel.RelationRefs[0].Caveat != "" ||
					readerRel.RelationRefs[1].ResourceType != "document" ||
					readerRel.RelationRefs[1].Relation != "..." ||
					readerRel.RelationRefs[1].Caveat != "" {
					return fmt.Errorf("unexpected reader relation: %+v", readerRel)
				}
				viewPerm := schema.Resources["document"].Permissions["view"]
				if viewPerm.Name != "view" ||
					viewPerm.OutputName != "view" ||
					viewPerm.Kind != "permission" ||
					len(viewPerm.AllowedSubjectTypes) != 2 ||
					viewPerm.AllowedSubjectTypes["user"] != "..." ||
					viewPerm.AllowedSubjectTypes["document"] != "..." ||
					len(viewPerm.OverrideAllowedSubjectTypes) != 0 ||
					len(viewPerm.RelationRefs) != 1 ||
					viewPerm.RelationRefs[0].ResourceType != "document" ||
					viewPerm.RelationRefs[0].Relation != "reader" ||
					viewPerm.RelationRefs[0].Caveat != "" {
					return fmt.Errorf("unexpected view permission: %+v", viewPerm)
				}
				return nil
			},
		},
		{
			name: "simple schema with optional subject relation",
			schematxt: `definition user {}
			            definition team {
							relation member: user
						}
						definition document {
							relation reader: team#member
							permission view = reader
						}`,
			validate: func(schema Schema) error {
				readerRel := schema.Resources["document"].Relations["reader"]
				if readerRel.Name != "reader" ||
					readerRel.OutputName != "reader" ||
					readerRel.Kind != "relation" ||
					len(readerRel.AllowedSubjectTypes) != 1 ||
					readerRel.AllowedSubjectTypes["team"] != "member" ||
					len(readerRel.RelationRefs) != 1 ||
					readerRel.RelationRefs[0].ResourceType != "team" ||
					readerRel.RelationRefs[0].Relation != "member" ||
					readerRel.RelationRefs[0].Caveat != "" {
					return fmt.Errorf("unexpected reader relation: %+v", readerRel)
				}
				viewPerm := schema.Resources["document"].Permissions["view"]
				if viewPerm.Name != "view" ||
					viewPerm.OutputName != "view" ||
					viewPerm.Kind != "permission" ||
					len(viewPerm.AllowedSubjectTypes) != 1 ||
					viewPerm.AllowedSubjectTypes["user"] != "..." ||
					len(viewPerm.OverrideAllowedSubjectTypes) != 0 ||
					len(viewPerm.RelationRefs) != 1 ||
					viewPerm.RelationRefs[0].ResourceType != "document" ||
					viewPerm.RelationRefs[0].Relation != "reader" ||
					viewPerm.RelationRefs[0].Caveat != "" {
					return fmt.Errorf("unexpected view permission: %+v", viewPerm)
				}
				return nil
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prefix := ""
			compiledSchema, _ := compiler.Compile(compiler.InputSchema{SchemaString: string(tc.schematxt)}, compiler.ObjectTypePrefix(prefix))
			schema := BuildSchema(compiledSchema)
			err := tc.validate(schema)
			if tc.err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
