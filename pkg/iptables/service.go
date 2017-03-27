// Package iptables is a wrapper around the iptables binary
package iptables

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/kontainerooo/kontainer.ooo/pkg/abstraction"
)

// Service handles iptables rules
type Service interface {
	// CreateChain creates a new chain
	CreateChain(name string) error
	// AddRule adds a given iptables rule
	AddRule(refid uint, rule Rule) error
	// RemoveRule removes a given iptables rule
	RemoveRule(id string) error
	// GetRulesForUser returns a list of all rules for a given user
	GetRulesForUser(refid uint) ([]Rule, error)
	// CreateIPTablesBackup creates an iptables backup file
	CreateIPTablesBackup() string
	// LoadIPTablesBackup restores iptables from backup file
	LoadIPTablesBackup() error
}

type dbAdapter interface {
	abstraction.DBAdapter
	AutoMigrate(...interface{}) error
	Where(interface{}, ...interface{}) error
	Create(interface{}) error
	First(interface{}, ...interface{}) error
	Find(interface{}, ...interface{}) error
	Delete(interface{}, ...interface{}) error
}

type service struct {
	iptPath string
	db      dbAdapter
}

func (s *service) executeIPTableCommand(c string) error {
	cmd := ExecCommand(s.iptPath, c)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (s *service) InitializeDatabases() error {
	return s.db.AutoMigrate(&iptablesEntry{})
}

func (s *service) ruleExists(r Rule) bool {
	s.db.Where("ID = ?", r.GetHash())
	res := s.db.GetValue()
	if res != nil {
		return true
	}
	return false
}

func (s *service) CreateChain(name string) error {
	cmd := fmt.Sprintf("-N %s", name)
	err := s.executeIPTableCommand(cmd)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) AddRule(refid uint, rule Rule) error {
	r, err := rule.ToString()
	if err != nil {
		return err
	}

	if s.ruleExists(rule) {
		return errors.New("Rule already exists")
	}

	entry := &iptablesEntry{
		Refid: refid,
		ID:    rule.GetHash(),
		Rule:  rule,
	}

	err = s.executeIPTableCommand(r)
	if err != nil {
		return err
	}

	err = s.db.Create(entry)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) RemoveRule(id string) error {
	rule := &iptablesEntry{}

	s.db.Where("ID = ?", id)
	err := s.db.First(rule)
	if err != nil {
		return err
	}

	rule.Operation = "-D"
	r, err := rule.ToString()
	if err != nil {
		return err
	}

	err = s.executeIPTableCommand(r)
	if err != nil {
		return err
	}

	err = s.db.Delete(&iptablesEntry{
		ID: id,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) GetRulesForUser(refid uint) ([]Rule, error) {
	rules := []Rule{}
	ipt := []iptablesEntry{}
	err := s.db.Find(&ipt)
	if err != nil {
		return []Rule{}, err
	}

	for _, entry := range ipt {
		rules = append(rules, entry.Rule)
	}
	return rules, nil
}

func (s *service) CreateIPTablesBackup() string {
	// TODO: implement
	return ""
}

func (s *service) LoadIPTablesBackup() error {
	// TODO: implement
	return nil
}

// ExecCommand is a wrapper around exec.Command used for testing
var ExecCommand = exec.Command

// NewService creates a new iptables service
func NewService(iptPath string, db dbAdapter) (Service, error) {
	s := &service{
		iptPath: iptPath,
		db:      db,
	}

	cmd := ExecCommand(iptPath, "--version")
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	err = s.InitializeDatabases()
	if err != nil {
		return nil, err
	}

	return s, nil
}
