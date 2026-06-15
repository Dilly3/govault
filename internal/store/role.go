package store

import (
	"strings"

	"github.com/dilly3/govault/internal/models"
	"gorm.io/gorm"
)

type RoleStore struct {
	DB *gorm.DB
}

func NewRoleStore(db *gorm.DB) *RoleStore {
	return &RoleStore{DB: db}
}

func (s *RoleStore) GetRole(id string) (*models.Role, error) {
	var role models.Role
	if err := s.DB.First(&role, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *RoleStore) GetRoleByName(name string) (*models.Role, error) {
	var role models.Role
	name = strings.ToLower(name)
	if err := s.DB.First(&role, "name = ?", name).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *RoleStore) CreateRole(role *models.Role) error {
	role.Name = strings.ToLower(role.Name)
	return s.DB.Create(role).Error
}

func (s *RoleStore) UpdateRole(role *models.Role) error {
	return s.DB.Save(role).Error
}

func (s *RoleStore) DeleteRole(id string) error {
	return s.DB.Delete(&models.Role{}, "id = ?", id).Error
}
