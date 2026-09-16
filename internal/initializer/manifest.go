package initializer

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/buding00/springhere/internal/registry"
	"github.com/buding00/springhere/internal/source"
	"github.com/buding00/springhere/internal/version"
)

type Manifest struct {
	SchemaVersion int                `yaml:"schema_version"`
	ProjectName   string             `yaml:"project_name"`
	CLIVersion    string             `yaml:"cli_version"`
	CreatedAt     string             `yaml:"created_at"`
	Combination   string             `yaml:"combination"`
	Components    ManifestComponents `yaml:"components"`
	Backend       ManifestBackend    `yaml:"backend"`
	Frontend      ManifestFrontend   `yaml:"frontend"`
	AuthMode      string             `yaml:"auth_mode"`
	APIBasePath   string             `yaml:"api_base_path"`
	OpenAPISha256 string             `yaml:"openapi_sha256"`
	ManagedFiles  map[string]string  `yaml:"managed_files"`
}

type ManifestComponents struct {
	Backend  ManifestComponent `yaml:"backend"`
	Frontend ManifestComponent `yaml:"frontend"`
}

type ManifestComponent struct {
	ID             string `yaml:"id"`
	Source         string `yaml:"source"`
	URL            string `yaml:"url"`
	Ref            string `yaml:"ref"`
	Commit         string `yaml:"commit"`
	ContractSHA256 string `yaml:"contract_sha256"`
}

type ManifestBackend struct {
	Module   string `yaml:"module"`
	Language string `yaml:"language"`
}

type ManifestFrontend struct {
	PackageName string `yaml:"package_name"`
}

func writeManifest(root string, m Manifest) error {
	if m.ManagedFiles == nil {
		m.ManagedFiles = map[string]string{}
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, ".springhere.yaml"), data, 0o644)
}

func newManifest(cfg Config, combo *registry.Combination, backend, frontend *source.Fetched, backendContract, frontendContract string) Manifest {
	return Manifest{
		SchemaVersion: 1,
		ProjectName:   cfg.ProjectName,
		CLIVersion:    version.Version,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		Combination:   combo.ID,
		Components: ManifestComponents{
			Backend: ManifestComponent{
				ID:             cfg.Backend,
				Source:         backend.Kind,
				URL:            backend.URL,
				Ref:            backend.Ref,
				Commit:         backend.Commit,
				ContractSHA256: backendContract,
			},
			Frontend: ManifestComponent{
				ID:             cfg.Frontend,
				Source:         frontend.Kind,
				URL:            frontend.URL,
				Ref:            frontend.Ref,
				Commit:         frontend.Commit,
				ContractSHA256: frontendContract,
			},
		},
		Backend: ManifestBackend{
			Module:   cfg.Module,
			Language: "go",
		},
		Frontend: ManifestFrontend{
			PackageName: cfg.FrontendPackage,
		},
		AuthMode:      combo.AuthMode,
		APIBasePath:   combo.APIBasePath,
		OpenAPISha256: combo.OpenAPISha256,
		ManagedFiles:  map[string]string{},
	}
}
