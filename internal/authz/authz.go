// Package authz provides Casbin-based authorization for mylib.
// Roles and permissions are defined as embedded policy — no external
// files needed. The Enforcer is safe for concurrent use.
package authz

import (
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	stringadapter "github.com/casbin/casbin/v2/persist/string-adapter"
)

const casbinModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`

const casbinPolicy = `
p, admin, books, read
p, admin, books, create
p, admin, books, edit
p, admin, books, delete
p, admin, books, upload
p, admin, books, enrich
p, admin, users, manage
p, admin, scan, trigger
p, admin, admin, access

p, reader, books, read
p, reader, books, edit
p, reader, books, upload
p, reader, books, enrich
p, reader, scan, trigger
`

// Authorizer wraps a Casbin enforcer with a simple API.
type Authorizer struct {
	enforcer *casbin.Enforcer
}

// New creates an Authorizer with the embedded model and policy.
func New() (*Authorizer, error) {
	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		return nil, fmt.Errorf("parse casbin model: %w", err)
	}
	sa := stringadapter.NewAdapter(casbinPolicy)
	e, err := casbin.NewEnforcer(m, sa)
	if err != nil {
		return nil, fmt.Errorf("create casbin enforcer: %w", err)
	}
	return &Authorizer{enforcer: e}, nil
}

// Can reports whether the given role is allowed to perform action on resource.
func (a *Authorizer) Can(role, resource, action string) bool {
	ok, err := a.enforcer.Enforce(role, resource, action)
	if err != nil {
		return false
	}
	return ok
}

// PermissionsForRole returns all "resource:action" strings the role
// is allowed. Used by the /api/auth/permissions endpoint.
func (a *Authorizer) PermissionsForRole(role string) []string {
	policies, _ := a.enforcer.GetFilteredPolicy(0, role)
	out := make([]string, 0, len(policies))
	for _, p := range policies {
		if len(p) >= 3 {
			out = append(out, p[1]+":"+p[2])
		}
	}
	return out
}

// AllPermissions returns the full set of unique "resource:action"
// strings across all roles. Useful for documentation.
func (a *Authorizer) AllPermissions() []string {
	seen := make(map[string]struct{})
	var out []string
	allPolicies, _ := a.enforcer.GetPolicy()
	for _, p := range allPolicies {
		if len(p) >= 3 {
			key := p[1] + ":" + p[2]
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				out = append(out, key)
			}
		}
	}
	return out
}

// RoleString converts a library.Role to the string Casbin expects.
func RoleString(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}
