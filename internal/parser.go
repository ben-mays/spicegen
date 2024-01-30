package internal

import (
	"fmt"
	"strings"

	corev1 "github.com/authzed/spicedb/pkg/proto/core/v1"
	"github.com/authzed/spicedb/pkg/schemadsl/compiler"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/anypb"
)

type Schema struct {
	Resources map[string]Resource
	Caveats   map[string]Caveat
}

type Caveat struct {
	Name string
	Args map[string]string // arg name to arg type
}

type RelationRef struct {
	ResourceType string // i.e. team#members -> team
	Relation     string // i.e. team#members -> member
	Caveat       string // key to caveat in schema
}

type Relation struct {
	Name       string
	OutputName string
	Kind       string
	// ResourceTypes to enforce
	AllowedSubjectTypes           []string
	SubjectTypesOptionalRelations map[string]string
	OverrideAllowedSubjectTypes   []string
	// Used for resolving allowed subject types if not given in a metatag
	RelationRefs []RelationRef
}

type Resource struct {
	Name        string
	Permissions map[string]Relation
	Relations   map[string]Relation

	// Either a specific resource type or "Resource"
	PermissionSubjectType string
	// Either a specific resource type or "Resource"
	RelationSubjectType string
}

func set(arr ...string) []string {
	res := map[string]bool{}
	for _, x := range arr {
		res[x] = true
	}
	return maps.Keys(res)
}

func BuildSchema(compiledSchema *compiler.CompiledSchema) Schema {
	// state tracks top-level resource definitions (i.e. definition statements) and their permissions/relations.
	state := map[string]Resource{}
	// Walk all objects and write their permissions/relations to state. Note, we don't resolve relation types here,
	// so a relation may have a non-resource type. We need to resolve that later in a second pass.
	for _, sd := range compiledSchema.ObjectDefinitions {
		permissions := make(map[string]Relation, 0)
		relations := make(map[string]Relation, 0)
		for _, rel := range sd.Relation {
			relation := handleRelation(sd.Name, rel)
			if relation.Kind == "permission" {
				permissions[relation.Name] = relation
			} else {
				relations[relation.Name] = relation
			}
		}
		state[sd.Name] = Resource{
			Name:                  sd.Name,
			Permissions:           permissions,
			Relations:             relations,
			PermissionSubjectType: "resource",
			RelationSubjectType:   "resource",
		}
	}
	// Second pass to map RelationRefs to allowedSubjectTypes
	for resourceName, resource := range state {
		all := maps.Values(resource.Relations)
		all = append(all, maps.Values(resource.Permissions)...)
		for _, relation := range all {
			relation.SubjectTypesOptionalRelations = map[string]string{}
			relation.AllowedSubjectTypes = []string{}
			if len(relation.OverrideAllowedSubjectTypes) == 0 {
				allowed := map[string]bool{}
				// This logic is super wonky -- it's attempting to resolve the relation refs into concrete subject types.
				// The gist: we build up allowed[] to point to all the allowed subject types for this relation. We also
				// track any optional relations that are allowed on this relation. If there are no allowed subject types
				// inferred we allow wildcard. Note: we could merge the maps of allowed subject types and optional relations
				// but I bolted this on and didn't want to refactor.
				for _, ref := range relation.RelationRefs {
					allowed[ref.ResourceType] = true
					if ref.Relation == "..." {
						// If the ref is concrete, we still append an optional relation equal to ... to make generation easier :shrug:
						relation.SubjectTypesOptionalRelations[ref.ResourceType] = ref.Relation
					} else {
						// Does this relation exist on _this_ resource type? If so, resolve it to the allowed subject types.
						// If it's on a different resource type, we keep the relation as-is.
						if indirectRef, ok := state[ref.ResourceType].Relations[ref.Relation]; ok && len(indirectRef.AllowedSubjectTypes) != 0 {
							for _, allowedSubjectType := range indirectRef.AllowedSubjectTypes {
								allowed[allowedSubjectType] = true
							}
						} else {
							relation.SubjectTypesOptionalRelations[ref.ResourceType] = ref.Relation
						}
					}
				}
				// If there are no allowed subject types, we default to wildcard
				if len(allowed) == 0 {
					relation.AllowedSubjectTypes = []string{"*"}
				} else {
					for k := range allowed {
						relation.AllowedSubjectTypes = append(relation.AllowedSubjectTypes, k)
					}
				}
			} else {
				relation.AllowedSubjectTypes = relation.OverrideAllowedSubjectTypes
			}
			if relation.Kind == "permission" {
				state[resourceName].Permissions[relation.Name] = relation
			} else {
				state[resourceName].Relations[relation.Name] = relation
			}
		}
	}

	// Final pass to find opportunities to make subjects concrete
	for resourceName, resource := range state {
		all := maps.Values(resource.Relations)
		all = append(all, maps.Values(resource.Permissions)...)
		allowedPerms := []string{}
		allowedRelations := []string{}
		for _, relation := range all {
			if relation.Kind == "permission" {
				allowedPerms = append(allowedPerms, relation.AllowedSubjectTypes...)
				allowedPerms = set(allowedPerms...)
			} else {
				allowedRelations = append(allowedRelations, relation.AllowedSubjectTypes...)
				allowedRelations = set(allowedRelations...)
			}
		}
		// If there is only one allowed subject type and it's not wildcard, set it as the permission subject type for this relation
		if len(allowedPerms) == 1 && allowedPerms[0] != "*" {
			resource.PermissionSubjectType = fmt.Sprintf("%s_resource", allowedPerms[0])
		}
		// Likewise for relation subject types
		if len(allowedRelations) == 1 && allowedRelations[0] != "*" {
			resource.RelationSubjectType = fmt.Sprintf("%s_resource", allowedRelations[0])
		}
		state[resourceName] = resource
	}

	return Schema{Resources: state}
}

// captures spicegen metatag info
type metatag struct {
	allowedSubjectTypes []string
	rename              string
}

func parseMetatags(comments []*anypb.Any) metatag {
	m := metatag{}
	for _, c := range comments {
		if c.TypeUrl == "type.googleapis.com/impl.v1.DocComment" && strings.Contains(string(c.Value), "//spicegen:") {
			// hacky string cutting
			meta := strings.Trim(strings.Trim(strings.Trim(string(c.Value), "/"), "*"), " ")
			tag := strings.Split(meta, ":")[1]
			var tagval string
			if strings.Contains(tag, "=") {
				split := strings.Split(tag, "=")
				tag = split[0]
				tagval = split[1]
			}
			switch tag {
			// Override subject type inference. This type does not have to exist in the schema!
			case "subject_type":
				m.allowedSubjectTypes = append(m.allowedSubjectTypes, tagval)
			// Rename public type but use value for spicedb
			case "rename":
				m.rename = tagval
			}
		}
	}
	return m
}

func parseKind(comments []*anypb.Any) string {
	for _, c := range comments {
		if strings.Contains(c.String(), "kind:PERMISSION") {
			return "permission"
		}
		if strings.Contains(c.String(), "kind:RELATION") {
			return "relation"
		}
	}
	return "unknown"
}

func resolveRelationGraph(nodeResourceType string, root *corev1.SetOperation) []RelationRef {
	// walks until there are no more children, returns the resulting allow types
	result := []RelationRef{}
	queue := []*corev1.SetOperation{root}
	for len(queue) != 0 {
		node := queue[0]
		queue = queue[1:]
		if node != nil {
			for _, child := range node.GetChild() {
				if child == nil {
					continue
				}
				if _, ok := child.GetChildType().(*corev1.SetOperation_Child_XThis); ok {
					continue
				}
				if val, ok := child.GetChildType().(*corev1.SetOperation_Child_ComputedUserset); ok {
					result = append(result, RelationRef{ResourceType: nodeResourceType, Relation: val.ComputedUserset.Relation})
				}
				if val, ok := child.GetChildType().(*corev1.SetOperation_Child_TupleToUserset); ok {
					result = append(result, RelationRef{ResourceType: val.TupleToUserset.Tupleset.Relation, Relation: val.TupleToUserset.ComputedUserset.Relation})
				}
				// recurse
				if val, ok := child.GetChildType().(*corev1.SetOperation_Child_UsersetRewrite); ok {
					queue = append(queue, val.UsersetRewrite.GetUnion())
					queue = append(queue, val.UsersetRewrite.GetExclusion())
					queue = append(queue, val.UsersetRewrite.GetIntersection())
				}
			}
		}
	}
	return result
}

func handleRelation(resourceType string, rel *corev1.Relation) Relation {
	relation := Relation{Name: rel.Name}
	relation.Kind = parseKind(rel.Metadata.MetadataMessage)
	metatag := parseMetatags(rel.Metadata.MetadataMessage)
	if metatag.rename != "" {
		relation.OutputName = metatag.rename
	} else {
		relation.OutputName = relation.Name
	}
	if metatag.allowedSubjectTypes != nil {
		relation.OverrideAllowedSubjectTypes = metatag.allowedSubjectTypes
	} else {
		// Resolve the relation refs. For example, given a relation like: owner: user | group, we want to resolve the user and group refs
		// to get a concrete type so we can generate a client that is typesafe. Ideally we'd produce something like `User` or `UserOrGroup`
		// but Go generics don't support composing union types without a cardinality explosion. If there are more than one assignable concrete
		// type (i.e. a ObjectDefinition, referred to as Resources in this code) then we just use the wildcard type and enforce at runtime.
		refs := make([]RelationRef, 0)
		rewrite := rel.GetUsersetRewrite()
		if rewrite != nil {
			for _, node := range []*corev1.SetOperation{rewrite.GetExclusion(), rewrite.GetUnion(), rewrite.GetIntersection()} {
				refs = append(refs, resolveRelationGraph(resourceType, node)...)
			}
		}
		if rel.GetTypeInformation() != nil {
			for _, m := range rel.TypeInformation.AllowedDirectRelations {
				r := RelationRef{
					ResourceType: m.Namespace,
					Relation:     m.GetRelation(),
				}
				if m.RequiredCaveat != nil {
					r.Caveat = m.RequiredCaveat.CaveatName
				}
				refs = append(refs, r)
			}
		}
		fmt.Printf("Resolved relation refs for %s: %v\n", relation.Name, refs)
		relation.RelationRefs = refs
	}
	return relation
}
