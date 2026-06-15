package models

import (
	"slices"
	"strings"
)

type Permission struct {
	ID          string
	Name        string
	Description string
}

type Role struct {
	ID          string
	Name        string
	Description string
	Permissions []Permission
}

func (r *Role) HasPermission(permission string) bool {
	return slices.ContainsFunc(r.Permissions, func(p Permission) bool {
		return strings.EqualFold(p.Name, permission)
	})
}
