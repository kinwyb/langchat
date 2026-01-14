package skills

import (
	"fmt"
	"log"
	"os"

	"github.com/tmc/langchaingo/tools"
)

// Skill stores basic info about a skill
type Skill struct {
	Name        string
	Description string
	Package     *Package
	Tools       []tools.Tool // Cached tools for the skill
	Loaded      bool         // Whether tools have been loaded
}

// LoadSkills 加载技能
func LoadSkills(skillsDir string) ([]*Skill, error) {
	var skills []*Skill
	if _, err := os.Stat(skillsDir); err == nil {
		packages, err := ParseSkillPackages(skillsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to parse skills packages: %w", err)
		}
		for _, skill := range packages {
			sk := &Skill{
				Name:        skill.Meta.Name,
				Description: skill.Meta.Description,
				Package:     skill,
				Loaded:      false,
			}
			sk.Tools, err = Tools(skill)
			if err != nil {
				log.Printf("Failed to load skill '%s' tools: %v", sk.Name, err)
			}
			skills = append(skills, sk)
		}
		log.Printf("Loaded %d skills info", len(packages))
	} else {
		return nil, fmt.Errorf("skills directory not found at %s", skillsDir)
	}
	return skills, nil
}
